package fsops

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// finderTimeout bounds one Finder call; the first one may wait for the user
// to allow the terminal to control Finder.
const finderTimeout = 60 * time.Second

// finderTrash asks Finder to move path to the Trash, which records where it
// came from so "Put Back" works. The path is passed as an argument, never
// interpolated into the script.
var finderTrash = func(ctx context.Context, path string) error {
	ctx, cancel := context.WithTimeout(ctx, finderTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "osascript",
		"-e", "on run argv",
		"-e", `tell application "Finder" to delete (POSIX file (item 1 of argv))`,
		"-e", "end run",
		path)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("finder: %v: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// TrashDir returns the user's Trash folder, used when Finder is not
// available (e.g. over SSH or when automation is denied).
func TrashDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".Trash")
}

// Trash moves each path to the macOS Trash through Finder, so items can be
// restored with "Put Back". If Finder cannot be used, the entry is moved
// into ~/.Trash directly (restorable by dragging it back). It returns how
// many entries were trashed.
func Trash(ctx context.Context, paths []string, onProgress func(Progress)) (int, error) {
	c := &copier{ctx: ctx, opts: CopyOptions{OnProgress: onProgress}}
	shares, err := c.measureEach(paths)
	if err != nil {
		return 0, err
	}
	c.report("")

	done := 0
	for i, p := range paths {
		if err := ctx.Err(); err != nil {
			return done, err
		}
		abs, err := filepath.Abs(p)
		if err != nil {
			return done, err
		}
		info, err := os.Lstat(abs)
		if err != nil {
			return done, err
		}
		if err := finderTrash(ctx, abs); err == nil {
			c.prog.BytesDone += shares[i].BytesTotal
			c.prog.FilesDone += shares[i].FilesTotal
			c.report(abs)
		} else {
			trash := TrashDir()
			if err := os.MkdirAll(trash, 0o700); err != nil {
				return done, err
			}
			dst := UniqueName(filepath.Join(trash, filepath.Base(abs)))
			if err := c.moveEntry(abs, dst, info, shares[i]); err != nil {
				return done, err
			}
		}
		done++
	}
	return done, nil
}
