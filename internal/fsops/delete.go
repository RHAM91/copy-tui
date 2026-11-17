package fsops

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"time"
)

// TrashDir returns the home trash of the freedesktop.org Trash spec:
// $XDG_DATA_HOME/Trash, or ~/.local/share/Trash when it is unset.
func TrashDir() string {
	if data := os.Getenv("XDG_DATA_HOME"); filepath.IsAbs(data) {
		return filepath.Join(data, "Trash")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "share", "Trash")
}

// Trash moves each path to the home trash, writing the .trashinfo file that
// lets file managers restore it. Paths on another file system are copied
// into the trash and then deleted. It returns how many entries were trashed.
func Trash(ctx context.Context, paths []string, onProgress func(Progress)) (int, error) {
	trash := TrashDir()
	filesDir, infoDir := filepath.Join(trash, "files"), filepath.Join(trash, "info")
	for _, dir := range []string{filesDir, infoDir} {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return 0, err
		}
	}

	c := &copier{ctx: ctx, opts: CopyOptions{OnProgress: onProgress}}
	shares := make([]Progress, len(paths))
	for i, p := range paths {
		before := c.prog
		if err := c.measure(p); err != nil {
			return 0, err
		}
		shares[i] = Progress{
			BytesTotal: c.prog.BytesTotal - before.BytesTotal,
			FilesTotal: c.prog.FilesTotal - before.FilesTotal,
		}
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
		name, infoPath, err := reserveTrashName(filesDir, infoDir, filepath.Base(abs), info.IsDir(), abs)
		if err != nil {
			return done, err
		}
		if err := c.moveEntry(abs, filepath.Join(filesDir, name), info, shares[i]); err != nil {
			os.Remove(infoPath)
			return done, err
		}
		done++
	}
	return done, nil
}

// reserveTrashName picks a name free in both files/ and info/ and claims it
// by creating its .trashinfo file exclusively, as the spec requires.
func reserveTrashName(filesDir, infoDir, base string, isDir bool, original string) (name, infoPath string, err error) {
	content := fmt.Sprintf("[Trash Info]\nPath=%s\nDeletionDate=%s\n",
		(&url.URL{Path: original}).EscapedPath(),
		time.Now().Format("2006-01-02T15:04:05"))

	for n := 0; ; n++ {
		name = numberedName(base, n, isDir)
		if _, err := os.Lstat(filepath.Join(filesDir, name)); err == nil {
			continue
		}
		infoPath = filepath.Join(infoDir, name+".trashinfo")
		f, err := os.OpenFile(infoPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		if errors.Is(err, fs.ErrExist) {
			continue
		}
		if err != nil {
			return "", "", err
		}
		_, werr := f.WriteString(content)
		cerr := f.Close()
		if werr != nil || cerr != nil {
			os.Remove(infoPath)
			return "", "", errors.Join(werr, cerr)
		}
		return name, infoPath, nil
	}
}

// Remove permanently deletes each path (directories recursively, symlinks
// without touching their target) and returns how many were removed.
func Remove(ctx context.Context, paths []string, onProgress func(Progress)) (int, error) {
	prog := Progress{FilesTotal: len(paths)}
	report := func() {
		if onProgress != nil {
			onProgress(prog)
		}
	}
	report()
	for _, p := range paths {
		if err := ctx.Err(); err != nil {
			return prog.FilesDone, err
		}
		prog.Current = p
		if err := os.RemoveAll(p); err != nil {
			return prog.FilesDone, err
		}
		prog.FilesDone++
		report()
	}
	return prog.FilesDone, nil
}
