package fsops

import (
	"context"
	"path/filepath"
	"testing"
)

func TestTrashSendsFilesAndFoldersToRecycleBin(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "a.txt"), "x", 0o644)
	mustWrite(t, filepath.Join(dir, "folder", "inner.txt"), "y", 0o644)

	var last Progress
	n, err := Trash(context.Background(),
		[]string{filepath.Join(dir, "a.txt"), filepath.Join(dir, "folder")},
		func(p Progress) { last = p })
	if err != nil {
		t.Fatal(err)
	}
	if n != 2 || exists(filepath.Join(dir, "a.txt")) || exists(filepath.Join(dir, "folder")) {
		t.Fatalf("n = %d, entries left behind", n)
	}
	if last.FilesDone != 2 || last.FilesTotal != 2 {
		t.Fatalf("progress = %+v", last)
	}
}

func TestTrashMissingPathFails(t *testing.T) {
	if _, err := Trash(context.Background(), []string{filepath.Join(t.TempDir(), "ghost")}, nil); err == nil {
		t.Fatal("expected error")
	}
}
