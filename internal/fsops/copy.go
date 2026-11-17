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

// ErrCopyIntoItself is returned when a directory would be copied inside its
// own tree.
var ErrCopyIntoItself = errors.New("cannot copy a directory into itself")

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
		dst := filepath.Join(dstDir, filepath.Base(src))

		if dstInfo, err := os.Lstat(dst); err == nil {
			if os.SameFile(srcInfo, dstInfo) {
				dst = UniqueName(dst)
			} else {
				action := ActionSkip
				if opts.Resolve != nil {
					if action, err = opts.Resolve(src, dst); err != nil {
						return res, err
					}
				}
				switch action {
				case ActionSkip:
					if err := c.skip(src); err != nil {
						return res, err
					}
					res.Skipped++
					continue
				case ActionRename:
					dst = UniqueName(dst)
				case ActionOverwrite:
					// Handled by copyEntry: files are replaced atomically,
					// directories merged, other types removed first.
				}
			}
		} else if !errors.Is(err, fs.ErrNotExist) {
			return res, err
		}

		if err := c.copyEntry(src, dst, srcInfo); err != nil {
			return res, err
		}
		res.Copied++
	}
	return res, nil
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
		return fmt.Errorf("%w: %s", ErrCopyIntoItself, src)
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
	stem, ext := base, ""
	if i := strings.LastIndex(base, "."); i > 0 {
		if info, err := os.Lstat(path); err == nil && !info.IsDir() {
			stem, ext = base[:i], base[i:]
		}
	}
	for n := 1; ; n++ {
		candidate := filepath.Join(dir, fmt.Sprintf("%s (%d)%s", stem, n, ext))
		if _, err := os.Lstat(candidate); errors.Is(err, fs.ErrNotExist) {
			return candidate
		}
	}
}
