package fsops

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"syscall"
)

// rename is os.Rename, replaceable in tests to simulate cross-device moves.
var rename = os.Rename

// Move moves each source into dstDir. Within one file system it renames;
// across file systems it copies with the same guarantees as Copy and then
// deletes the source, so a cancelled move never loses the source.
//
// Conflicts are resolved like Copy, except that overwriting a directory with
// a directory merges both. Moving a source into its own directory is a
// no-op counted as skipped.
func Move(ctx context.Context, sources []string, dstDir string, opts CopyOptions) (CopyResult, error) {
	c := &copier{ctx: ctx, opts: opts}
	var res CopyResult

	for _, src := range sources {
		if err := checkNotIntoItself(src, dstDir); err != nil {
			return res, err
		}
	}
	shares := make([]Progress, len(sources))
	for i, src := range sources {
		before := c.prog
		if err := c.measure(src); err != nil {
			return res, err
		}
		shares[i] = Progress{
			BytesTotal: c.prog.BytesTotal - before.BytesTotal,
			FilesTotal: c.prog.FilesTotal - before.FilesTotal,
		}
	}
	c.report("")

	for i, src := range sources {
		if err := ctx.Err(); err != nil {
			return res, err
		}
		srcInfo, err := os.Lstat(src)
		if err != nil {
			return res, err
		}
		dst, skip, err := c.target(src, dstDir, srcInfo, false)
		if err != nil {
			return res, err
		}
		if skip {
			res.Skipped++
			continue
		}
		if err := c.moveEntry(src, dst, srcInfo, shares[i]); err != nil {
			return res, err
		}
		res.Copied++
	}
	return res, nil
}

func (c *copier) moveEntry(src, dst string, info fs.FileInfo, share Progress) error {
	dstInfo, statErr := os.Lstat(dst)
	dstExists := statErr == nil
	if statErr != nil && !errors.Is(statErr, fs.ErrNotExist) {
		return statErr
	}

	if dstExists && dstInfo.IsDir() && info.IsDir() {
		// Overwriting a directory with a directory merges them.
		if err := c.copyEntry(src, dst, info); err != nil {
			return err
		}
		return os.RemoveAll(src)
	}

	// rename cannot replace a directory with a file or the other way
	// around: park the old entry aside and restore it if the move fails.
	aside := ""
	if dstExists && (dstInfo.IsDir() || info.IsDir()) {
		var err error
		if aside, err = parkAside(dst); err != nil {
			return err
		}
	}
	// Once the new entry is complete at dst, the parked one is dropped even
	// if removing the source fails afterwards.
	dropAside := func() error {
		if aside == "" {
			return nil
		}
		return os.RemoveAll(aside)
	}

	err := rename(src, dst)
	switch {
	case err == nil:
		c.prog.BytesDone += share.BytesTotal
		c.prog.FilesDone += share.FilesTotal
		c.report(src)
		return dropAside()
	case errors.Is(err, syscall.EXDEV):
		if err := c.copyEntry(src, dst, info); err != nil {
			if aside != "" {
				restoreAside(dst, aside)
			} else if !dstExists {
				_ = os.RemoveAll(dst)
			}
			return err
		}
		return errors.Join(dropAside(), os.RemoveAll(src))
	default:
		restoreAside(dst, aside)
		return err
	}
}
