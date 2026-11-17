package fsops

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// stubFinder replaces the Finder call for the duration of the test.
func stubFinder(t *testing.T, fn func(ctx context.Context, path string) error) {
	t.Helper()
	prev := finderTrash
	finderTrash = fn
	t.Cleanup(func() { finderTrash = prev })
}

func TestTrashUsesFinderWithAbsolutePaths(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "a.txt")
	mustWrite(t, f, "x", 0o644)
	t.Chdir(dir)

	var asked []string
	stubFinder(t, func(_ context.Context, path string) error {
		asked = append(asked, path)
		return os.Remove(path)
	})
	n, err := Trash(context.Background(), []string{"a.txt"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	real, _ := filepath.EvalSymlinks(dir)
	if n != 1 || len(asked) != 1 || (asked[0] != f && asked[0] != filepath.Join(real, "a.txt")) {
		t.Fatalf("n = %d, Finder asked for %v, want absolute %s", n, asked, f)
	}
}

func TestTrashFallsBackToUserTrashWhenFinderFails(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	stubFinder(t, func(context.Context, string) error { return errors.New("not authorized") })

	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "a.txt"), "first", 0o644)
	mustWrite(t, filepath.Join(home, ".Trash", "a.txt"), "already there", 0o644)

	n, err := Trash(context.Background(), []string{filepath.Join(dir, "a.txt")}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 || exists(filepath.Join(dir, "a.txt")) {
		t.Fatalf("n = %d, source still exists = %v", n, exists(filepath.Join(dir, "a.txt")))
	}
	if got := readFile(t, filepath.Join(home, ".Trash", "a (1).txt")); got != "first" {
		t.Fatalf("trashed copy = %q", got)
	}
	if got := readFile(t, filepath.Join(home, ".Trash", "a.txt")); got != "already there" {
		t.Fatalf("existing trash entry overwritten: %q", got)
	}
}

func TestTrashMissingPathFailsBeforeFinder(t *testing.T) {
	stubFinder(t, func(context.Context, string) error {
		t.Fatal("Finder called for a missing path")
		return nil
	})
	if _, err := Trash(context.Background(), []string{filepath.Join(t.TempDir(), "ghost")}, nil); err == nil {
		t.Fatal("expected error")
	}
}
