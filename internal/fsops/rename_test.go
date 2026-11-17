package fsops

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestValidateName(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		wantErr bool
	}{
		{name: "plain name", in: "notes.txt"},
		{name: "spaces are fine", in: "my notes"},
		{name: "empty", in: "", wantErr: true},
		{name: "only spaces", in: "   ", wantErr: true},
		{name: "dot", in: ".", wantErr: true},
		{name: "dot dot", in: "..", wantErr: true},
		{name: "slash", in: "a/b", wantErr: true},
		{name: "nul byte", in: "a\x00b", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateName(tt.in)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ValidateName(%q) error = %v, wantErr %v", tt.in, err, tt.wantErr)
			}
		})
	}
}

func TestRename(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "old.txt"), "data", 0o644)
	mustWrite(t, filepath.Join(dir, "taken.txt"), "other", 0o644)

	if err := Rename(filepath.Join(dir, "old.txt"), "taken.txt"); !errors.Is(err, os.ErrExist) {
		t.Fatalf("rename onto existing name: err = %v, want os.ErrExist", err)
	}
	if got := readFile(t, filepath.Join(dir, "taken.txt")); got != "other" {
		t.Fatalf("existing file overwritten: %q", got)
	}
	if err := Rename(filepath.Join(dir, "old.txt"), "bad/name"); err == nil {
		t.Fatal("expected invalid name error")
	}
	if err := Rename(filepath.Join(dir, "old.txt"), "new.txt"); err != nil {
		t.Fatal(err)
	}
	if exists(filepath.Join(dir, "old.txt")) || readFile(t, filepath.Join(dir, "new.txt")) != "data" {
		t.Fatal("rename did not move the file")
	}
	if err := Rename(filepath.Join(dir, "new.txt"), "new.txt"); err != nil {
		t.Fatalf("renaming to the same name should be a no-op: %v", err)
	}
}

func TestMkdir(t *testing.T) {
	dir := t.TempDir()

	path, err := Mkdir(dir, "docs")
	if err != nil {
		t.Fatal(err)
	}
	if info, err := os.Stat(path); err != nil || !info.IsDir() || path != filepath.Join(dir, "docs") {
		t.Fatalf("Mkdir created %q: %v", path, err)
	}
	if _, err := Mkdir(dir, "docs"); !errors.Is(err, os.ErrExist) {
		t.Fatalf("second Mkdir err = %v, want os.ErrExist", err)
	}
	if _, err := Mkdir(dir, ".."); err == nil {
		t.Fatal("expected invalid name error")
	}
}

func TestValidateNameOnWindows(t *testing.T) {
	tests := []struct {
		in      string
		wantErr bool
	}{
		{in: "report.txt"},
		{in: "console.txt"},
		{in: "con.txt.bak", wantErr: true}, // reserved even with several extensions
		{in: `a\b`, wantErr: true},
		{in: "a:b", wantErr: true},
		{in: "what?", wantErr: true},
		{in: "star*", wantErr: true},
		{in: `"quoted"`, wantErr: true},
		{in: "<tag>", wantErr: true},
		{in: "pipe|", wantErr: true},
		{in: "CON", wantErr: true},
		{in: "nul.txt", wantErr: true},
		{in: "com1", wantErr: true},
		{in: "LPT9.log", wantErr: true},
		{in: "trailing.", wantErr: true},
		{in: "trailing ", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			err := validateName(tt.in, "windows")
			if (err != nil) != tt.wantErr {
				t.Fatalf("validateName(%q, windows) error = %v, wantErr %v", tt.in, err, tt.wantErr)
			}
			if !errors.Is(err, ErrInvalidName) && err != nil {
				t.Fatalf("error does not wrap ErrInvalidName: %v", err)
			}
		})
	}
}

func TestValidateNameAllowsWindowsCharactersOnUnix(t *testing.T) {
	for _, name := range []string{`a\b`, "a:b", "what?", "CON"} {
		if err := validateName(name, "linux"); err != nil {
			t.Fatalf("validateName(%q, linux) = %v, want nil", name, err)
		}
	}
}

func TestRenameChangingOnlyCase(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "File.txt"), "data", 0o644)

	if err := Rename(filepath.Join(dir, "File.txt"), "file.txt"); err != nil {
		t.Fatalf("case-only rename refused: %v", err)
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 || entries[0].Name() != "file.txt" {
		t.Fatalf("entries = %v, want only file.txt", entries)
	}
	if got := readFile(t, filepath.Join(dir, "file.txt")); got != "data" {
		t.Fatalf("content = %q", got)
	}
}

func TestRenameOntoDifferentFileWithOtherCaseIsRefused(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "File.txt"), "one", 0o644)
	mustWrite(t, filepath.Join(dir, "file.txt"), "two", 0o644)
	if entries, _ := os.ReadDir(dir); len(entries) != 2 {
		t.Skip("case-insensitive file system")
	}
	if err := Rename(filepath.Join(dir, "File.txt"), "file.txt"); !errors.Is(err, os.ErrExist) {
		t.Fatalf("err = %v, want os.ErrExist", err)
	}
	if got := readFile(t, filepath.Join(dir, "file.txt")); got != "two" {
		t.Fatalf("file.txt overwritten: %q", got)
	}
}
