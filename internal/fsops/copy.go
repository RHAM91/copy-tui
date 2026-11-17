package fsops

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/sys/unix"
)

// chunkSize is how many bytes are copied between progress reports and
// cancellation checks.
const chunkSize = 256 * 1024

// ErrIntoItself is returned when a directory would be copied inside its
// own tree.
var ErrIntoItself = errors.New("cannot copy or move a directory into itself")

// ConflictAction tells Copy what to do when the destination already exists.
type ConflictAction int

const (
	// ActionOverwrite replaces the destination. Directories are merged.
	ActionOverwrite ConflictAction = iota
	// ActionSkip leaves the destination untouched.
	ActionSkip
	// ActionRename copies under the next free name, e.g. "a (1).txt".
	ActionRename
)

// ConflictResolver decides what to do when dst already exists. Returning an
// error aborts the whole operation.
type ConflictResolver func(src, dst string) (ConflictAction, error)

// Progress reports how far a copy has gone.
type Progress struct {
	BytesDone  int64
	BytesTotal int64
	FilesDone  int
	FilesTotal int
	// Current is the source path being copied.
	Current string
}

// CopyOptions configures Copy.
type CopyOptions struct {
	// Resolve is asked for each top-level source whose destination exists.
	// When nil, conflicts are skipped.
	Resolve ConflictResolver
	// OnProgress, when set, is called as bytes are copied.
	OnProgress func(Progress)
}

// CopyResult counts the top-level sources copied or skipped.
type CopyResult struct {
	Copied  int
	Skipped int
}

type copier struct {
	ctx  context.Context
	opts CopyOptions
	prog Progress
}

// Copy copies each source into dstDir with "cp -a" semantics: directories
// recursively, symlinks as symlinks, and permissions and modification times
// preserved. Each file is written to a temporary name and renamed into
// place, so a cancelled or failed copy never leaves a partial file behind
// nor destroys the file it was going to overwrite.
//
// Copying a source into its own directory creates a duplicate with a free
// name without asking.
func Copy(ctx context.Context, sources []string, dstDir string, opts CopyOptions) (CopyResult, error) {
	c := &copier{ctx: ctx, opts: opts}
	var res CopyResult

	for _, src := range sources {
		if err := checkNotIntoItself(src, dstDir); err != nil {
			return res, err
		}
	}
	for _, src := range sources {
		if err := c.measure(src); err != nil {
			return res, err
		}
	}
	c.report("")

	for _, src := range sources {
		if err := ctx.Err(); err != nil {
			return res, err
		}
		srcInfo, err := os.Lstat(src)
		if err != nil {
			return res, err
		}
		dst, skip, err := c.target(src, dstDir, srcInfo, true)
		if err != nil {
			return res, err
		}
		if skip {
			res.Skipped++
			continue
		}
		// On overwrite, copyEntry replaces files atomically, merges
		// directories and removes entries of another type first.
		if err := c.copyEntry(src, dst, srcInfo); err != nil {
			return res, err
		}
		res.Copied++
	}
	return res, nil
}

// target decides where src goes inside dstDir, asking the resolver when the
// name is taken. When the destination is src itself, it duplicates under a
// free name if duplicate is true, or skips otherwise. Skipped sources are
// counted as done in the progress.
func (c *copier) target(src, dstDir string, srcInfo fs.FileInfo, duplicate bool) (dst string, skip bool, err error) {
	dst = filepath.Join(dstDir, filepath.Base(src))
	dstInfo, err := os.Lstat(dst)
	if errors.Is(err, fs.ErrNotExist) {
		return dst, false, nil
	}
	if err != nil {
		return "", false, err
	}
	if os.SameFile(srcInfo, dstInfo) {
		if duplicate {
			return UniqueName(dst), false, nil
		}
		return "", true, c.skip(src)
	}
	action := ActionSkip
	if c.opts.Resolve != nil {
		if action, err = c.opts.Resolve(src, dst); err != nil {
			return "", false, err
		}
	}
	switch action {
	case ActionSkip:
		return "", true, c.skip(src)
	case ActionRename:
		return UniqueName(dst), false, nil
	}
	return dst, false, nil
}

// checkNotIntoItself fails when src is a directory and dstDir is src or lies
// inside it.
func checkNotIntoItself(src, dstDir string) error {
	info, err := os.Lstat(src)
	if err != nil || !info.IsDir() {
		return err
	}
	s, err := realPath(src)
	if err != nil {
		return err
	}
	d, err := realPath(dstDir)
	if err != nil {
		return err
	}
	if d == s || strings.HasPrefix(d, s+string(filepath.Separator)) {
		return fmt.Errorf("%w: %s", ErrIntoItself, src)
	}
	return nil
}

func realPath(p string) (string, error) {
	abs, err := filepath.Abs(p)
	if err != nil {
		return "", err
	}
	return filepath.EvalSymlinks(abs)
}

// measure adds the size and file count of src to the progress totals.
func (c *copier) measure(src string) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if err := c.ctx.Err(); err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		c.prog.FilesTotal++
		if d.Type().IsRegular() {
			info, err := d.Info()
			if err != nil {
				return err
			}
			c.prog.BytesTotal += info.Size()
		}
		return nil
	})
}

// skip counts src as done so the progress bar still reaches 100%.
func (c *copier) skip(src string) error {
	before := c.prog
	c.prog = Progress{}
	err := c.measure(src)
	skipped := c.prog
	c.prog = before
	if err != nil {
		return err
	}
	c.prog.BytesDone += skipped.BytesTotal
	c.prog.FilesDone += skipped.FilesTotal
	c.report(src)
	return nil
}

func (c *copier) report(current string) {
	c.prog.Current = current
	if c.opts.OnProgress != nil {
		c.opts.OnProgress(c.prog)
	}
}

func (c *copier) copyEntry(src, dst string, info fs.FileInfo) error {
	if err := c.ctx.Err(); err != nil {
		return err
	}
	switch {
	case info.Mode()&fs.ModeSymlink != 0:
		return c.copySymlink(src, dst, info)
	case info.IsDir():
		return c.copyDir(src, dst, info)
	case info.Mode().IsRegular():
		return c.copyFile(src, dst, info)
	default:
		return fmt.Errorf("unsupported file type: %s", src)
	}
}

func (c *copier) copyDir(src, dst string, info fs.FileInfo) error {
	if existing, err := os.Lstat(dst); err == nil && !existing.IsDir() {
		if err := os.Remove(dst); err != nil {
			return err
		}
	}
	// Create writable first; the real mode is applied once the content is in.
	if err := os.Mkdir(dst, 0o700); err != nil && !errors.Is(err, fs.ErrExist) {
		return err
	}
	children, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	for _, child := range children {
		childInfo, err := child.Info()
		if err != nil {
			return err
		}
		if err := c.copyEntry(filepath.Join(src, child.Name()), filepath.Join(dst, child.Name()), childInfo); err != nil {
			return err
		}
	}
	if err := os.Chmod(dst, preservedMode(info)); err != nil {
		return err
	}
	return os.Chtimes(dst, accessTime(info), info.ModTime())
}

func (c *copier) copySymlink(src, dst string, info fs.FileInfo) error {
	target, err := os.Readlink(src)
	if err != nil {
		return err
	}
	if err := removeExisting(dst); err != nil {
		return err
	}
	if err := os.Symlink(target, dst); err != nil {
		return err
	}
	ts := []unix.Timespec{
		unix.NsecToTimespec(accessTime(info).UnixNano()),
		unix.NsecToTimespec(info.ModTime().UnixNano()),
	}
	// Best effort: some file systems cannot set times on links.
	_ = unix.UtimesNanoAt(unix.AT_FDCWD, dst, ts, unix.AT_SYMLINK_NOFOLLOW)
	c.prog.FilesDone++
	c.report(src)
	return nil
}

func (c *copier) copyFile(src, dst string, info fs.FileInfo) (err error) {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	tmp, err := os.CreateTemp(filepath.Dir(dst), "."+filepath.Base(dst)+".copiar-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer func() {
		if err != nil {
			tmp.Close()
			os.Remove(tmpName)
		}
	}()

	buf := make([]byte, chunkSize)
	for {
		if err := c.ctx.Err(); err != nil {
			return err
		}
		n, readErr := in.Read(buf)
		if n > 0 {
			if _, err := tmp.Write(buf[:n]); err != nil {
				return err
			}
			c.prog.BytesDone += int64(n)
			c.report(src)
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return readErr
		}
	}
	if err := c.ctx.Err(); err != nil {
		return err
	}
	if err := tmp.Chmod(preservedMode(info)); err != nil {
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chtimes(tmpName, accessTime(info), info.ModTime()); err != nil {
		return err
	}
	if existing, statErr := os.Lstat(dst); statErr == nil && existing.IsDir() {
		if err := os.RemoveAll(dst); err != nil {
			return err
		}
	}
	if err := os.Rename(tmpName, dst); err != nil {
		return err
	}
	c.prog.FilesDone++
	c.report(src)
	return nil
}

// removeExisting deletes dst when it exists, whatever its type.
func removeExisting(dst string) error {
	if _, err := os.Lstat(dst); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return err
	}
	return os.RemoveAll(dst)
}

func preservedMode(info fs.FileInfo) fs.FileMode {
	return info.Mode() & (fs.ModePerm | fs.ModeSetuid | fs.ModeSetgid | fs.ModeSticky)
}

func accessTime(info fs.FileInfo) time.Time {
	if st, ok := info.Sys().(*unix.Stat_t); ok {
		return time.Unix(st.Atim.Unix())
	}
	return info.ModTime()
}

// UniqueName returns path when nothing exists there, or the first free
// variant "name (N).ext".
func UniqueName(path string) string {
	if _, err := os.Lstat(path); errors.Is(err, fs.ErrNotExist) {
		return path
	}
	dir, base := filepath.Split(path)
	isDir := false
	if info, err := os.Lstat(path); err == nil {
		isDir = info.IsDir()
	}
	for n := 1; ; n++ {
		candidate := filepath.Join(dir, numberedName(base, n, isDir))
		if _, err := os.Lstat(candidate); errors.Is(err, fs.ErrNotExist) {
			return candidate
		}
	}
}

// numberedName returns base for n == 0 and "stem (n).ext" otherwise. The
// extension is kept only for files whose name is not just ".something".
func numberedName(base string, n int, isDir bool) string {
	if n == 0 {
		return base
	}
	stem, ext := base, ""
	if i := strings.LastIndex(base, "."); i > 0 && !isDir {
		stem, ext = base[:i], base[i:]
	}
	return fmt.Sprintf("%s (%d)%s", stem, n, ext)
}
