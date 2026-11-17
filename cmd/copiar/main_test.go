package main

import (
	"bytes"
	"context"
	"fmt"
	"io/fs"
	"runtime"
	"strings"
	"testing"

	"github.com/RHAM91/copy-tui/internal/update"
)

func TestParseArgs(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		wantIcons bool
		wantErr   bool
	}{
		{name: "icons enabled by default", args: nil, wantIcons: true},
		{name: "no-icons flag disables icons", args: []string{"--no-icons"}, wantIcons: false},
		{name: "unknown flag is an error", args: []string{"--nope"}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := parseArgs(tt.args)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if cfg.ui.Icons != tt.wantIcons {
				t.Fatalf("Icons = %v, want %v", cfg.ui.Icons, tt.wantIcons)
			}
			if cfg.showVersion {
				t.Fatal("showVersion set without --version")
			}
		})
	}
}

func TestVersionFlag(t *testing.T) {
	cfg, err := parseArgs([]string{"--version"})
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.showVersion {
		t.Fatal("--version did not request the version")
	}
}

func TestVersionStringUsesInjectedVersion(t *testing.T) {
	prev := version
	t.Cleanup(func() { version = prev })

	version = "1.2.3"
	if got := versionString(); got != "copiar 1.2.3" {
		t.Fatalf("versionString() = %q, want %q", got, "copiar 1.2.3")
	}
}

func TestParseArgsCommands(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		want    string
		wantErr bool
	}{
		{name: "no command opens the UI", args: nil, want: ""},
		{name: "update command", args: []string{"update"}, want: "update"},
		{name: "unknown command is an error", args: []string{"upgrade"}, wantErr: true},
		{name: "extra arguments are an error", args: []string{"update", "now"}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := parseArgs(tt.args)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if err == nil && cfg.command != tt.want {
				t.Fatalf("command = %q, want %q", cfg.command, tt.want)
			}
		})
	}
}

type fakeUpdater struct {
	latest  string
	newer   bool
	err     error
	applied bool
}

func (f *fakeUpdater) Check(context.Context, string) (string, bool, error) {
	return f.latest, f.newer, f.err
}

func (f *fakeUpdater) Apply(context.Context, string, string) (string, bool, error) {
	if f.err != nil {
		return "", false, f.err
	}
	f.applied = f.newer
	return f.latest, f.newer, nil
}

func TestRunUpdate(t *testing.T) {
	tests := []struct {
		name     string
		up       *fakeUpdater
		current  string
		wantCode int
		wantOut  string
		applied  bool
	}{
		{name: "installs a newer release", up: &fakeUpdater{latest: "0.2.0", newer: true}, current: "0.1.0",
			wantCode: 0, wantOut: "Updated copiar v0.1.0 → v0.2.0", applied: true},
		{name: "reports up to date", up: &fakeUpdater{latest: "0.1.0"}, current: "0.1.0",
			wantCode: 0, wantOut: "copiar is up to date (v0.1.0)"},
		{name: "explains development builds", up: &fakeUpdater{err: update.ErrDevBuild}, current: "dev",
			wantCode: 1, wantOut: "development build"},
		{name: "suggests sudo when the binary is not writable", up: &fakeUpdater{err: fmt.Errorf("installing: %w", fs.ErrPermission)}, current: "0.1.0",
			wantCode: 1, wantOut: permissionHint(runtime.GOOS)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out bytes.Buffer
			code := runUpdate(context.Background(), tt.up, tt.current, "/usr/local/bin/copiar", &out)
			if code != tt.wantCode {
				t.Fatalf("exit code = %d, want %d (output %q)", code, tt.wantCode, out.String())
			}
			if !strings.Contains(out.String(), tt.wantOut) {
				t.Fatalf("output = %q, want it to contain %q", out.String(), tt.wantOut)
			}
			if tt.up.applied != tt.applied {
				t.Fatalf("applied = %v, want %v", tt.up.applied, tt.applied)
			}
		})
	}
}

func TestStartupCheckSkippedForDevBuilds(t *testing.T) {
	if startupCheck(&fakeUpdater{latest: "9.9.9", newer: true}, "dev") != nil {
		t.Fatal("dev builds must not check for updates at startup")
	}
	check := startupCheck(&fakeUpdater{latest: "0.2.0", newer: true}, "0.1.0")
	if check == nil {
		t.Fatal("release builds must check for updates")
	}
	if got, err := check(context.Background()); err != nil || got != "0.2.0" {
		t.Fatalf("check = (%q, %v), want 0.2.0", got, err)
	}
	upToDate := startupCheck(&fakeUpdater{latest: "0.1.0"}, "0.1.0")
	if got, _ := upToDate(context.Background()); got != "" {
		t.Fatalf("up-to-date check returned %q, want empty", got)
	}
}

func TestPermissionHintPerOS(t *testing.T) {
	if got := permissionHint("linux"); !strings.Contains(got, "sudo copiar update") {
		t.Fatalf("linux hint = %q", got)
	}
	if got := permissionHint("darwin"); !strings.Contains(got, "sudo copiar update") {
		t.Fatalf("darwin hint = %q", got)
	}
	if got := permissionHint("windows"); !strings.Contains(got, "Administrator") || strings.Contains(got, "sudo") {
		t.Fatalf("windows hint = %q", got)
	}
}
