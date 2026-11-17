package fsops

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestRemoveDeletesFilesDirectoriesAndLinksOnly(t *testing.T) {
	dir := t.TempDir()
	keep := filepath.Join(t.TempDir(), "target.txt")
	mustWrite(t, keep, "keep", 0o644)
	mustWrite(t, filepath.Join(dir, "f"), "x", 0o644)
	mustWrite(t, filepath.Join(dir, "d", "sub", "g"), "y", 0o644)
	if err := os.Symlink(keep, filepath.Join(dir, "link")); err != nil {
		t.Fatal(err)
	}

	var calls []Progress
	n, err := Remove(context.Background(),
		[]string{filepath.Join(dir, "f"), filepath.Join(dir, "d"), filepath.Join(dir, "link")},
		func(p Progress) { calls = append(calls, p) })
	if err != nil {
		t.Fatal(err)
	}
	if n != 3 {
		t.Fatalf("removed = %d, want 3", n)
	}
	if left, _ := os.ReadDir(dir); len(left) != 0 {
		t.Fatalf("entries left: %v", left)
	}
	if got := readFile(t, keep); got != "keep" {
		t.Fatal("symlink target was deleted")
	}
	if last := calls[len(calls)-1]; last.FilesDone != 3 || last.FilesTotal != 3 {
		t.Fatalf("progress = %+v", last)
	}
}

func TestRemoveStopsWhenCancelled(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "a"), "x", 0o644)
	mustWrite(t, filepath.Join(dir, "b"), "x", 0o644)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := Remove(ctx, []string{filepath.Join(dir, "a"), filepath.Join(dir, "b")}, nil); err == nil {
		t.Fatal("expected cancellation error")
	}
	if !exists(filepath.Join(dir, "a")) || !exists(filepath.Join(dir, "b")) {
		t.Fatal("files removed after cancellation")
	}
}
