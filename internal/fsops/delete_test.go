package fsops

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// isolatedTrash points the home trash to a temporary directory.
func isolatedTrash(t *testing.T) string {
	t.Helper()
	data := t.TempDir()
	t.Setenv("XDG_DATA_HOME", data)
	return filepath.Join(data, "Trash")
}

func TestTrashDirFollowsXDGDataHome(t *testing.T) {
	trash := isolatedTrash(t)
	if got := TrashDir(); got != trash {
		t.Fatalf("TrashDir() = %q, want %q", got, trash)
	}
	t.Setenv("XDG_DATA_HOME", "")
	t.Setenv("HOME", "/home/someone")
	if got := TrashDir(); got != "/home/someone/.local/share/Trash" {
		t.Fatalf("TrashDir() without XDG_DATA_HOME = %q", got)
	}
}

func TestTrashMovesEntryAndWritesInfo(t *testing.T) {
	for _, cross := range []bool{false, true} {
		t.Run(map[bool]string{false: "same device", true: "cross device"}[cross], func(t *testing.T) {
			if cross {
				crossDevice(t)
			}
			trash := isolatedTrash(t)
			dir := t.TempDir()
			f := filepath.Join(dir, "my file.txt")
			mustWrite(t, f, "bye", 0o644)

			res, err := Trash(context.Background(), []string{f}, nil)
			if err != nil {
				t.Fatal(err)
			}
			if res != 1 {
				t.Fatalf("trashed = %d, want 1", res)
			}
			if exists(f) {
				t.Fatal("source still exists")
			}
			if got := readFile(t, filepath.Join(trash, "files", "my file.txt")); got != "bye" {
				t.Fatalf("trashed content = %q", got)
			}
			info := readFile(t, filepath.Join(trash, "info", "my file.txt.trashinfo"))
			wantPath := "Path=" + strings.ReplaceAll(f, " ", "%20")
			if !strings.HasPrefix(info, "[Trash Info]\n") || !strings.Contains(info, wantPath+"\n") {
				t.Fatalf("trashinfo = %q, want header and %q", info, wantPath)
			}
			if !regexp.MustCompile(`\nDeletionDate=\d{4}-\d\d-\d\dT\d\d:\d\d:\d\d\n`).MatchString(info) {
				t.Fatalf("trashinfo lacks DeletionDate: %q", info)
			}
		})
	}
}

func TestTrashSameNameTwiceKeepsBoth(t *testing.T) {
	trash := isolatedTrash(t)
	a, b := t.TempDir(), t.TempDir()
	mustWrite(t, filepath.Join(a, "same.txt"), "first", 0o644)
	mustWrite(t, filepath.Join(b, "same.txt"), "second", 0o644)

	if _, err := Trash(context.Background(), []string{filepath.Join(a, "same.txt"), filepath.Join(b, "same.txt")}, nil); err != nil {
		t.Fatal(err)
	}
	if got := readFile(t, filepath.Join(trash, "files", "same.txt")); got != "first" {
		t.Fatalf("same.txt = %q", got)
	}
	if got := readFile(t, filepath.Join(trash, "files", "same (1).txt")); got != "second" {
		t.Fatalf("same (1).txt = %q", got)
	}
	if !exists(filepath.Join(trash, "info", "same (1).txt.trashinfo")) {
		t.Fatal("missing info for second file")
	}
}

func TestTrashDirectory(t *testing.T) {
	trash := isolatedTrash(t)
	root := filepath.Join(t.TempDir(), "d")
	mustWrite(t, filepath.Join(root, "sub", "f"), "F", 0o644)

	if _, err := Trash(context.Background(), []string{root}, nil); err != nil {
		t.Fatal(err)
	}
	if exists(root) {
		t.Fatal("directory still exists")
	}
	if got := readFile(t, filepath.Join(trash, "files", "d", "sub", "f")); got != "F" {
		t.Fatalf("trashed file = %q", got)
	}
}

func TestTrashFailureRemovesInfoFile(t *testing.T) {
	trash := isolatedTrash(t)
	missing := filepath.Join(t.TempDir(), "ghost")

	if _, err := Trash(context.Background(), []string{missing}, nil); err == nil {
		t.Fatal("expected error for missing source")
	}
	if entries, _ := os.ReadDir(filepath.Join(trash, "info")); len(entries) != 0 {
		t.Fatalf("stale info files: %v", entries)
	}
}

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
