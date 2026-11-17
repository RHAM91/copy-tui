package fsops

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

// crossDevice makes every rename fail as if src and dst were on different
// file systems, forcing Move to copy and delete.
func crossDevice(t *testing.T) {
	t.Helper()
	prev := rename
	rename = func(oldpath, newpath string) error {
		return &os.LinkError{Op: "rename", Old: oldpath, New: newpath, Err: syscall.EXDEV}
	}
	t.Cleanup(func() { rename = prev })
}

func TestMoveFile(t *testing.T) {
	for _, cross := range []bool{false, true} {
		name := "same device"
		if cross {
			name = "cross device"
		}
		t.Run(name, func(t *testing.T) {
			if cross {
				crossDevice(t)
			}
			src, dst := t.TempDir(), t.TempDir()
			f := filepath.Join(src, "a.txt")
			mustWrite(t, f, "data", 0o640)
			if err := os.Chtimes(f, past, past); err != nil {
				t.Fatal(err)
			}

			res, err := Move(context.Background(), []string{f}, dst, CopyOptions{Resolve: failResolver(t)})
			if err != nil {
				t.Fatal(err)
			}
			if exists(f) {
				t.Fatal("source still exists")
			}
			out := filepath.Join(dst, "a.txt")
			if got := readFile(t, out); got != "data" {
				t.Fatalf("content = %q", got)
			}
			info, _ := os.Stat(out)
			if (unixPerms() && info.Mode().Perm() != 0o640) || !info.ModTime().Equal(past) {
				t.Fatalf("mode/mtime not preserved: %v %v", info.Mode(), info.ModTime())
			}
			if res.Copied != 1 {
				t.Fatalf("result = %+v", res)
			}
		})
	}
}

func TestMoveDirectory(t *testing.T) {
	for _, cross := range []bool{false, true} {
		t.Run(map[bool]string{false: "same device", true: "cross device"}[cross], func(t *testing.T) {
			if cross {
				crossDevice(t)
			}
			src, dst := t.TempDir(), t.TempDir()
			root := filepath.Join(src, "d")
			mustWrite(t, filepath.Join(root, "sub", "f.txt"), "F", 0o644)
			if err := os.Symlink("sub/f.txt", filepath.Join(root, "link")); err != nil {
				t.Fatal(err)
			}

			if _, err := Move(context.Background(), []string{root}, dst, CopyOptions{Resolve: failResolver(t)}); err != nil {
				t.Fatal(err)
			}
			if exists(root) {
				t.Fatal("source directory still exists")
			}
			if got := readFile(t, filepath.Join(dst, "d", "sub", "f.txt")); got != "F" {
				t.Fatalf("f.txt = %q", got)
			}
			if target, err := os.Readlink(filepath.Join(dst, "d", "link")); err != nil || target != "sub/f.txt" {
				t.Fatalf("link = %q, %v", target, err)
			}
		})
	}
}

func TestMoveConflictActions(t *testing.T) {
	tests := []struct {
		name       string
		action     ConflictAction
		want       map[string]string
		srcRemains bool
	}{
		{name: "skip leaves both files", action: ActionSkip, want: map[string]string{"a.txt": "old"}, srcRemains: true},
		{name: "overwrite replaces the destination", action: ActionOverwrite, want: map[string]string{"a.txt": "new"}},
		{name: "rename keeps both", action: ActionRename, want: map[string]string{"a.txt": "old", "a (1).txt": "new"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src, dst := t.TempDir(), t.TempDir()
			f := filepath.Join(src, "a.txt")
			mustWrite(t, f, "new", 0o644)
			mustWrite(t, filepath.Join(dst, "a.txt"), "old", 0o644)

			if _, err := Move(context.Background(), []string{f}, dst, CopyOptions{Resolve: always(tt.action)}); err != nil {
				t.Fatal(err)
			}
			for name, want := range tt.want {
				if got := readFile(t, filepath.Join(dst, name)); got != want {
					t.Fatalf("%s = %q, want %q", name, got, want)
				}
			}
			if exists(f) != tt.srcRemains {
				t.Fatalf("source exists = %v, want %v", exists(f), tt.srcRemains)
			}
		})
	}
}

func TestMoveOverwriteDirectoryMergesAndRemovesSource(t *testing.T) {
	src, dst := t.TempDir(), t.TempDir()
	mustWrite(t, filepath.Join(src, "d", "same.txt"), "new", 0o644)
	mustWrite(t, filepath.Join(dst, "d", "same.txt"), "old", 0o644)
	mustWrite(t, filepath.Join(dst, "d", "keep.txt"), "keep", 0o644)

	if _, err := Move(context.Background(), []string{filepath.Join(src, "d")}, dst, CopyOptions{Resolve: always(ActionOverwrite)}); err != nil {
		t.Fatal(err)
	}
	if exists(filepath.Join(src, "d")) {
		t.Fatal("source directory still exists")
	}
	for name, want := range map[string]string{"same.txt": "new", "keep.txt": "keep"} {
		if got := readFile(t, filepath.Join(dst, "d", name)); got != want {
			t.Fatalf("%s = %q, want %q", name, got, want)
		}
	}
}

func TestMoveOverwriteReplacesEntryOfDifferentType(t *testing.T) {
	src, dst := t.TempDir(), t.TempDir()
	mustWrite(t, filepath.Join(src, "x"), "file", 0o644)
	mustWrite(t, filepath.Join(dst, "x", "inner"), "dir", 0o644)

	if _, err := Move(context.Background(), []string{filepath.Join(src, "x")}, dst, CopyOptions{Resolve: always(ActionOverwrite)}); err != nil {
		t.Fatal(err)
	}
	if got := readFile(t, filepath.Join(dst, "x")); got != "file" {
		t.Fatalf("x = %q", got)
	}
	left, _ := os.ReadDir(dst)
	if len(left) != 1 {
		t.Fatalf("leftovers in destination: %v", left)
	}
}

func TestMoveIntoSameDirectoryIsNoop(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "a.txt")
	mustWrite(t, f, "data", 0o644)

	res, err := Move(context.Background(), []string{f}, dir, CopyOptions{Resolve: failResolver(t)})
	if err != nil {
		t.Fatal(err)
	}
	if got := readFile(t, f); got != "data" {
		t.Fatalf("file changed: %q", got)
	}
	if res.Skipped != 1 || res.Copied != 0 {
		t.Fatalf("result = %+v", res)
	}
	if left, _ := os.ReadDir(dir); len(left) != 1 {
		t.Fatalf("unexpected entries: %v", left)
	}
}

func TestMoveDirectoryIntoItselfFails(t *testing.T) {
	src := t.TempDir()
	root := filepath.Join(src, "d")
	mustWrite(t, filepath.Join(root, "sub", "f"), "x", 0o644)

	_, err := Move(context.Background(), []string{root}, filepath.Join(root, "sub"), CopyOptions{Resolve: failResolver(t)})
	if !errors.Is(err, ErrIntoItself) {
		t.Fatalf("err = %v, want ErrIntoItself", err)
	}
	if !exists(filepath.Join(root, "sub", "f")) {
		t.Fatal("source damaged")
	}
}

func TestMoveCancelAcrossDevicesKeepsSourceAndCleansDestination(t *testing.T) {
	crossDevice(t)
	src, dst := t.TempDir(), t.TempDir()
	root := filepath.Join(src, "d")
	mustWrite(t, filepath.Join(root, "a.bin"), strings.Repeat("a", 3*chunkSize), 0o644)
	mustWrite(t, filepath.Join(root, "b.bin"), strings.Repeat("b", 3*chunkSize), 0o644)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	opts := CopyOptions{
		Resolve: failResolver(t),
		OnProgress: func(p Progress) {
			if p.BytesDone > chunkSize {
				cancel()
			}
		},
	}
	if _, err := Move(ctx, []string{root}, dst, opts); !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
	for _, name := range []string{"a.bin", "b.bin"} {
		if !exists(filepath.Join(root, name)) {
			t.Fatalf("source %s lost after cancel", name)
		}
	}
	if left, _ := os.ReadDir(dst); len(left) != 0 {
		t.Fatalf("destination not cleaned: %v", left)
	}
}

func TestMoveReportsCompleteProgress(t *testing.T) {
	src, dst := t.TempDir(), t.TempDir()
	mustWrite(t, filepath.Join(src, "a"), "12345", 0o644)
	mustWrite(t, filepath.Join(src, "d", "b"), "123", 0o644)

	var last Progress
	opts := CopyOptions{Resolve: failResolver(t), OnProgress: func(p Progress) { last = p }}
	if _, err := Move(context.Background(), []string{filepath.Join(src, "a"), filepath.Join(src, "d")}, dst, opts); err != nil {
		t.Fatal(err)
	}
	if last.BytesDone != 8 || last.BytesTotal != 8 || last.FilesDone != 2 || last.FilesTotal != 2 {
		t.Fatalf("progress = %+v", last)
	}
}
