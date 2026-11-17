package fsops

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

var past = time.Date(2020, 1, 2, 3, 4, 5, 0, time.UTC)

func mustWrite(t *testing.T, path, content string, perm os.FileMode) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), perm); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, perm); err != nil {
		t.Fatal(err)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func exists(path string) bool {
	_, err := os.Lstat(path)
	return err == nil
}

// failResolver fails the test when a conflict is reported unexpectedly.
func failResolver(t *testing.T) ConflictResolver {
	return func(src, dst string) (ConflictAction, error) {
		t.Fatalf("unexpected conflict %s -> %s", src, dst)
		return ActionSkip, nil
	}
}

func always(a ConflictAction) ConflictResolver {
	return func(string, string) (ConflictAction, error) { return a, nil }
}

func TestCopyFilePreservesContentPermissionsAndMtime(t *testing.T) {
	src, dst := t.TempDir(), t.TempDir()
	f := filepath.Join(src, "notes.txt")
	mustWrite(t, f, "hello", 0o640)
	if err := os.Chtimes(f, past, past); err != nil {
		t.Fatal(err)
	}

	res, err := Copy(context.Background(), []string{f}, dst, CopyOptions{Resolve: failResolver(t)})
	if err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dst, "notes.txt")
	if got := readFile(t, out); got != "hello" {
		t.Fatalf("content = %q", got)
	}
	info, _ := os.Stat(out)
	if info.Mode().Perm() != 0o640 {
		t.Fatalf("perm = %v, want 0640", info.Mode().Perm())
	}
	if !info.ModTime().Equal(past) {
		t.Fatalf("mtime = %v, want %v", info.ModTime(), past)
	}
	if res.Copied != 1 || res.Skipped != 0 {
		t.Fatalf("result = %+v", res)
	}
	if got := readFile(t, f); got != "hello" {
		t.Fatalf("source changed: %q", got)
	}
}

func TestCopyDirectoryRecursively(t *testing.T) {
	src, dst := t.TempDir(), t.TempDir()
	root := filepath.Join(src, "project")
	mustWrite(t, filepath.Join(root, "a.txt"), "A", 0o644)
	mustWrite(t, filepath.Join(root, "deep", "er", "b.txt"), "B", 0o600)
	if err := os.MkdirAll(filepath.Join(root, "empty"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(filepath.Join(root, "deep"), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(filepath.Join(root, "deep"), past, past); err != nil {
		t.Fatal(err)
	}

	if _, err := Copy(context.Background(), []string{root}, dst, CopyOptions{Resolve: failResolver(t)}); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dst, "project")
	if got := readFile(t, filepath.Join(out, "a.txt")); got != "A" {
		t.Fatalf("a.txt = %q", got)
	}
	if got := readFile(t, filepath.Join(out, "deep", "er", "b.txt")); got != "B" {
		t.Fatalf("b.txt = %q", got)
	}
	if info, err := os.Stat(filepath.Join(out, "empty")); err != nil || !info.IsDir() {
		t.Fatalf("empty dir not copied: %v", err)
	}
	info, _ := os.Stat(filepath.Join(out, "deep"))
	if info.Mode().Perm() != 0o750 {
		t.Fatalf("dir perm = %v, want 0750", info.Mode().Perm())
	}
	if !info.ModTime().Equal(past) {
		t.Fatalf("dir mtime = %v, want %v", info.ModTime(), past)
	}
}

func TestCopyKeepsSymlinksAsLinks(t *testing.T) {
	src, dst := t.TempDir(), t.TempDir()
	root := filepath.Join(src, "dir")
	mustWrite(t, filepath.Join(root, "real.txt"), "R", 0o644)
	if err := os.Symlink("real.txt", filepath.Join(root, "link")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("../..", filepath.Join(root, "loop")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("missing", filepath.Join(root, "broken")); err != nil {
		t.Fatal(err)
	}

	if _, err := Copy(context.Background(), []string{root}, dst, CopyOptions{Resolve: failResolver(t)}); err != nil {
		t.Fatal(err)
	}
	for name, want := range map[string]string{"link": "real.txt", "loop": "../..", "broken": "missing"} {
		got, err := os.Readlink(filepath.Join(dst, "dir", name))
		if err != nil {
			t.Fatalf("%s is not a symlink: %v", name, err)
		}
		if got != want {
			t.Fatalf("%s -> %q, want %q", name, got, want)
		}
	}
}

func TestCopyReportsProgressUntilComplete(t *testing.T) {
	src, dst := t.TempDir(), t.TempDir()
	mustWrite(t, filepath.Join(src, "d", "one"), strings.Repeat("x", 3000), 0o644)
	mustWrite(t, filepath.Join(src, "d", "two"), strings.Repeat("y", 500), 0o644)
	mustWrite(t, filepath.Join(src, "three"), "zz", 0o644)

	var last Progress
	calls := 0
	opts := CopyOptions{
		Resolve:    failResolver(t),
		OnProgress: func(p Progress) { last = p; calls++ },
	}
	if _, err := Copy(context.Background(), []string{filepath.Join(src, "d"), filepath.Join(src, "three")}, dst, opts); err != nil {
		t.Fatal(err)
	}
	if calls == 0 {
		t.Fatal("OnProgress never called")
	}
	if last.BytesTotal != 3502 || last.BytesDone != 3502 {
		t.Fatalf("bytes = %d/%d, want 3502/3502", last.BytesDone, last.BytesTotal)
	}
	if last.FilesTotal != 3 || last.FilesDone != 3 {
		t.Fatalf("files = %d/%d, want 3/3", last.FilesDone, last.FilesTotal)
	}
}

func TestCopyConflictActions(t *testing.T) {
	tests := []struct {
		name        string
		action      ConflictAction
		wantContent map[string]string
		wantResult  CopyResult
	}{
		{
			name:        "skip keeps the existing file",
			action:      ActionSkip,
			wantContent: map[string]string{"a.txt": "old"},
			wantResult:  CopyResult{Skipped: 1},
		},
		{
			name:        "overwrite replaces the existing file",
			action:      ActionOverwrite,
			wantContent: map[string]string{"a.txt": "new"},
			wantResult:  CopyResult{Copied: 1},
		},
		{
			name:        "rename keeps both files",
			action:      ActionRename,
			wantContent: map[string]string{"a.txt": "old", "a (1).txt": "new"},
			wantResult:  CopyResult{Copied: 1},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src, dst := t.TempDir(), t.TempDir()
			mustWrite(t, filepath.Join(src, "a.txt"), "new", 0o644)
			mustWrite(t, filepath.Join(dst, "a.txt"), "old", 0o644)

			var asked []string
			resolve := func(s, d string) (ConflictAction, error) {
				asked = append(asked, filepath.Base(s)+"->"+d)
				return tt.action, nil
			}
			res, err := Copy(context.Background(), []string{filepath.Join(src, "a.txt")}, dst, CopyOptions{Resolve: resolve})
			if err != nil {
				t.Fatal(err)
			}
			if len(asked) != 1 || asked[0] != "a.txt->"+filepath.Join(dst, "a.txt") {
				t.Fatalf("resolver calls = %v", asked)
			}
			for name, want := range tt.wantContent {
				if got := readFile(t, filepath.Join(dst, name)); got != want {
					t.Fatalf("%s = %q, want %q", name, got, want)
				}
			}
			if res != tt.wantResult {
				t.Fatalf("result = %+v, want %+v", res, tt.wantResult)
			}
		})
	}
}

func TestCopyRenamePicksNextFreeName(t *testing.T) {
	src, dst := t.TempDir(), t.TempDir()
	mustWrite(t, filepath.Join(src, "a.txt"), "new", 0o644)
	mustWrite(t, filepath.Join(dst, "a.txt"), "0", 0o644)
	mustWrite(t, filepath.Join(dst, "a (1).txt"), "1", 0o644)

	if _, err := Copy(context.Background(), []string{filepath.Join(src, "a.txt")}, dst, CopyOptions{Resolve: always(ActionRename)}); err != nil {
		t.Fatal(err)
	}
	if got := readFile(t, filepath.Join(dst, "a (2).txt")); got != "new" {
		t.Fatalf("a (2).txt = %q", got)
	}
}

func TestCopyOverwriteDirectoryMergesContents(t *testing.T) {
	src, dst := t.TempDir(), t.TempDir()
	mustWrite(t, filepath.Join(src, "d", "same.txt"), "new", 0o644)
	mustWrite(t, filepath.Join(src, "d", "added.txt"), "added", 0o644)
	mustWrite(t, filepath.Join(dst, "d", "same.txt"), "old", 0o644)
	mustWrite(t, filepath.Join(dst, "d", "keep.txt"), "keep", 0o644)

	if _, err := Copy(context.Background(), []string{filepath.Join(src, "d")}, dst, CopyOptions{Resolve: always(ActionOverwrite)}); err != nil {
		t.Fatal(err)
	}
	for name, want := range map[string]string{"same.txt": "new", "added.txt": "added", "keep.txt": "keep"} {
		if got := readFile(t, filepath.Join(dst, "d", name)); got != want {
			t.Fatalf("%s = %q, want %q", name, got, want)
		}
	}
}

func TestCopyOverwriteReplacesEntryOfDifferentType(t *testing.T) {
	src, dst := t.TempDir(), t.TempDir()
	mustWrite(t, filepath.Join(src, "x"), "file", 0o644)
	mustWrite(t, filepath.Join(dst, "x", "inner"), "dir content", 0o644)

	if _, err := Copy(context.Background(), []string{filepath.Join(src, "x")}, dst, CopyOptions{Resolve: always(ActionOverwrite)}); err != nil {
		t.Fatal(err)
	}
	if got := readFile(t, filepath.Join(dst, "x")); got != "file" {
		t.Fatalf("x = %q, want file", got)
	}
}

func TestCopyIntoSameDirectoryDuplicatesWithoutAsking(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "a.txt")
	mustWrite(t, f, "data", 0o644)

	res, err := Copy(context.Background(), []string{f}, dir, CopyOptions{Resolve: failResolver(t)})
	if err != nil {
		t.Fatal(err)
	}
	if got := readFile(t, f); got != "data" {
		t.Fatalf("source changed: %q", got)
	}
	if got := readFile(t, filepath.Join(dir, "a (1).txt")); got != "data" {
		t.Fatalf("duplicate = %q", got)
	}
	if res.Copied != 1 {
		t.Fatalf("result = %+v", res)
	}
}

func TestCopyDirectoryIntoItselfFails(t *testing.T) {
	src := t.TempDir()
	root := filepath.Join(src, "d")
	mustWrite(t, filepath.Join(root, "sub", "f"), "x", 0o644)

	_, err := Copy(context.Background(), []string{root}, filepath.Join(root, "sub"), CopyOptions{Resolve: failResolver(t)})
	if !errors.Is(err, ErrCopyIntoItself) {
		t.Fatalf("err = %v, want ErrCopyIntoItself", err)
	}
	if exists(filepath.Join(root, "sub", "d")) {
		t.Fatal("something was copied into the source tree")
	}
}

func TestCopyCancelRemovesPartialFile(t *testing.T) {
	src, dst := t.TempDir(), t.TempDir()
	mustWrite(t, filepath.Join(src, "big.bin"), strings.Repeat("b", 5*chunkSize), 0o644)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	opts := CopyOptions{
		Resolve: failResolver(t),
		OnProgress: func(p Progress) {
			if p.BytesDone > 0 {
				cancel()
			}
		},
	}
	_, err := Copy(ctx, []string{filepath.Join(src, "big.bin")}, dst, opts)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
	left, _ := os.ReadDir(dst)
	if len(left) != 0 {
		t.Fatalf("destination not clean after cancel: %v", left)
	}
}

func TestCopyCancelDuringOverwriteKeepsOriginal(t *testing.T) {
	src, dst := t.TempDir(), t.TempDir()
	mustWrite(t, filepath.Join(src, "big.bin"), strings.Repeat("n", 5*chunkSize), 0o644)
	mustWrite(t, filepath.Join(dst, "big.bin"), "original", 0o644)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	opts := CopyOptions{
		Resolve: always(ActionOverwrite),
		OnProgress: func(p Progress) {
			if p.BytesDone > 0 {
				cancel()
			}
		},
	}
	if _, err := Copy(ctx, []string{filepath.Join(src, "big.bin")}, dst, opts); !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
	if got := readFile(t, filepath.Join(dst, "big.bin")); got != "original" {
		t.Fatalf("original overwritten by partial copy: %d bytes", len(got))
	}
	left, _ := os.ReadDir(dst)
	if len(left) != 1 {
		t.Fatalf("temporary files left behind: %v", left)
	}
}

func TestCopyResolverErrorAbortsWithoutChanges(t *testing.T) {
	src, dst := t.TempDir(), t.TempDir()
	mustWrite(t, filepath.Join(src, "a.txt"), "new", 0o644)
	mustWrite(t, filepath.Join(dst, "a.txt"), "old", 0o644)
	stop := errors.New("cancelled by user")

	_, err := Copy(context.Background(), []string{filepath.Join(src, "a.txt")}, dst, CopyOptions{
		Resolve: func(string, string) (ConflictAction, error) { return ActionSkip, stop },
	})
	if !errors.Is(err, stop) {
		t.Fatalf("err = %v, want %v", err, stop)
	}
	if got := readFile(t, filepath.Join(dst, "a.txt")); got != "old" {
		t.Fatalf("a.txt = %q, want old", got)
	}
}

func TestUniqueName(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "taken.txt"), "", 0o644)
	mustWrite(t, filepath.Join(dir, "taken (1).txt"), "", 0o644)
	mustWrite(t, filepath.Join(dir, ".bashrc"), "", 0o644)
	if err := os.Mkdir(filepath.Join(dir, "folder"), 0o755); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "free name is kept", in: "free.txt", want: "free.txt"},
		{name: "counter goes before the extension", in: "taken.txt", want: "taken (2).txt"},
		{name: "dotfile keeps its name as a whole", in: ".bashrc", want: ".bashrc (1)"},
		{name: "directory gets a counter", in: "folder", want: "folder (1)"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := UniqueName(filepath.Join(dir, tt.in))
			if got != filepath.Join(dir, tt.want) {
				t.Fatalf("UniqueName(%q) = %q, want %q", tt.in, filepath.Base(got), tt.want)
			}
		})
	}
}
