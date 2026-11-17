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
