package fsops

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func dirNames(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}
	return names
}

func TestReplaceRestoresParkedEntryWhenInstallFails(t *testing.T) {
	dir := t.TempDir()
	dst := filepath.Join(dir, "target")
	mustWrite(t, dst, "old", 0o644)

	wantErr := errors.New("install failed")
	err := replace(dst, true, func() error {
		if _, statErr := os.Lstat(dst); !errors.Is(statErr, os.ErrNotExist) {
			t.Errorf("old entry still at dst while installing: %v", statErr)
		}
		return wantErr
	})

	if !errors.Is(err, wantErr) {
		t.Fatalf("err = %v, want %v", err, wantErr)
	}
	if got := readFile(t, dst); got != "old" {
		t.Fatalf("dst = %q, want the original %q", got, "old")
	}
	if names := dirNames(t, dir); len(names) != 1 {
		t.Fatalf("leftover entries after restore: %v", names)
	}
}

func TestReplaceDropsParkedEntryWhenInstallSucceeds(t *testing.T) {
	dir := t.TempDir()
	dst := filepath.Join(dir, "target")
	mustWrite(t, dst, "old", 0o644)

	err := replace(dst, true, func() error {
		return os.WriteFile(dst, []byte("new"), 0o644)
	})

	if err != nil {
		t.Fatal(err)
	}
	if got := readFile(t, dst); got != "new" {
		t.Fatalf("dst = %q, want %q", got, "new")
	}
	if names := dirNames(t, dir); len(names) != 1 {
		t.Fatalf("parked entry was not dropped: %v", names)
	}
}

func TestReplaceWithoutParkLeavesExistingEntryAlone(t *testing.T) {
	dir := t.TempDir()
	dst := filepath.Join(dir, "target")
	mustWrite(t, dst, "old", 0o644)

	wantErr := errors.New("install failed")
	err := replace(dst, false, func() error {
		if got := readFile(t, dst); got != "old" {
			t.Errorf("dst = %q while installing, want it untouched", got)
		}
		return wantErr
	})

	if !errors.Is(err, wantErr) {
		t.Fatalf("err = %v, want %v", err, wantErr)
	}
	if got := readFile(t, dst); got != "old" {
		t.Fatalf("dst = %q after failure, want %q", got, "old")
	}
	if names := dirNames(t, dir); len(names) != 1 {
		t.Fatalf("unexpected entries: %v", names)
	}
}

func TestMeasureReturnsTotalsWithoutTouchingProgress(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "a"), "abc", 0o644)
	mustWrite(t, filepath.Join(root, "sub", "b"), "12345", 0o644)

	c := &copier{ctx: context.Background()}
	got, err := c.measure(root)
	if err != nil {
		t.Fatal(err)
	}

	if got.FilesTotal != 2 || got.BytesTotal != 8 {
		t.Fatalf("measure = %+v, want 2 files and 8 bytes", got)
	}
	if c.prog != (Progress{}) {
		t.Fatalf("measure changed the copier progress: %+v", c.prog)
	}
}
