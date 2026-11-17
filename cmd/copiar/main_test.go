package main

import "testing"

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
