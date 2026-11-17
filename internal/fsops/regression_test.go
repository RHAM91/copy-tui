package fsops

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// B1: overwriting a folder that contains the source must never destroy data.
func TestOverwriteIntoAncestorOfSourceIsRefused(t *testing.T) {
	for _, op := range []string{"copy", "move"} {
		t.Run(op, func(t *testing.T) {
			base := t.TempDir()
			mustWrite(t, filepath.Join(base, "A", "A", "A"), "inner A", 0o644)
			mustWrite(t, filepath.Join(base, "A", "A", "B"), "precious B", 0o644)
			src := filepath.Join(base, "A", "A")
			run := Copy
			if op == "move" {
				run = Move
			}
			_, err := run(context.Background(), []string{src}, base, CopyOptions{Resolve: always(ActionOverwrite)})
			if !errors.Is(err, ErrIntoItself) {
				t.Fatalf("err = %v, want ErrIntoItself", err)
			}
			for name, want := range map[string]string{"A": "inner A", "B": "precious B"} {
				if got := readFile(t, filepath.Join(src, name)); got != want {
					t.Fatalf("%s = %q, want %q", name, got, want)
				}
			}
		})
	}
}

// B2: cancelling a copy that replaces a file with a folder keeps the file.
func TestCopyCancelDirOverFileKeepsOriginal(t *testing.T) {
	base := t.TempDir()
	src := filepath.Join(base, "src", "data")
	writeFile(t, filepath.Join(src, "big"), 4*chunkSize)
	dst := filepath.Join(base, "dst")
	mustWrite(t, filepath.Join(dst, "data"), "original file", 0o644)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	_, err := Copy(ctx, []string{src}, dst, CopyOptions{
		Resolve: always(ActionOverwrite),
		OnProgress: func(p Progress) {
			if p.BytesDone > 0 {
				cancel()
			}
		},
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
	if got := readFile(t, filepath.Join(dst, "data")); got != "original file" {
		t.Fatalf("original destroyed, got %q", got)
	}
	if left, _ := os.ReadDir(dst); len(left) != 1 {
		t.Fatalf("leftovers in destination: %v", left)
	}
}

// B2 (other direction): a file replacing a folder, and a successful type swap
// leaves no parked entry behind.
func TestCopyTypeSwapLeavesNoLeftovers(t *testing.T) {
	base := t.TempDir()
	mustWrite(t, filepath.Join(base, "src", "x"), "file", 0o644)
	mustWrite(t, filepath.Join(base, "dst", "x", "inner"), "dir", 0o644)
	if _, err := Copy(context.Background(), []string{filepath.Join(base, "src", "x")}, filepath.Join(base, "dst"), CopyOptions{Resolve: always(ActionOverwrite)}); err != nil {
		t.Fatal(err)
	}
	if left, _ := os.ReadDir(filepath.Join(base, "dst")); len(left) != 1 {
		t.Fatalf("leftovers: %v", left)
	}
}

// Observation: a cross-device move whose source cannot be fully removed must
// not leave the parked ".copiar-old" entry in the destination.
func TestMoveCrossDeviceSourceRemovalFailureLeavesNoParkedEntry(t *testing.T) {
	requireEnforcedPermissions(t)
	crossDevice(t)
	base := t.TempDir()
	src := filepath.Join(base, "src", "data")
	mustWrite(t, filepath.Join(src, "ro", "f"), "f", 0o644)
	if err := os.Chmod(filepath.Join(src, "ro"), 0o555); err != nil {
		t.Fatal(err)
	}
	dst := filepath.Join(base, "dst")
	mustWrite(t, filepath.Join(dst, "data"), "old file", 0o644)
	t.Cleanup(func() {
		os.Chmod(filepath.Join(src, "ro"), 0o755)
		os.Chmod(filepath.Join(dst, "data", "ro"), 0o755)
	})

	Move(context.Background(), []string{src}, dst, CopyOptions{Resolve: always(ActionOverwrite)})
	entries, _ := os.ReadDir(dst)
	for _, e := range entries {
		if strings.Contains(e.Name(), ".copiar-old") {
			t.Fatalf("parked entry left behind: %s", e.Name())
		}
	}
}

// Observation: names close to the 255-byte limit can still be copied.
func TestCopyLongFileName(t *testing.T) {
	src, dst := t.TempDir(), t.TempDir()
	name := strings.Repeat("n", 250) + ".txt"
	mustWrite(t, filepath.Join(src, name), "long", 0o644)
	if _, err := Copy(context.Background(), []string{filepath.Join(src, name)}, dst, CopyOptions{Resolve: failResolver(t)}); err != nil {
		t.Fatal(err)
	}
	if got := readFile(t, filepath.Join(dst, name)); got != "long" {
		t.Fatalf("content = %q", got)
	}
}
