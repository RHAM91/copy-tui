// Package fsops holds the file system operations of copiar, independent of
// the user interface: listing, copying, moving, deleting and trashing.
package fsops

import (
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Entry describes one item of a directory listing.
type Entry struct {
	Name string
	Path string
	// IsDir is true for directories and for symlinks pointing to directories,
	// so the UI can navigate into both.
	IsDir bool
	// IsLink is true when the entry itself is a symbolic link.
	IsLink     bool
	LinkTarget string
	Size       int64
	Mode       fs.FileMode
}

// ListDir returns the entries of dir sorted with directories first and then
// by case-insensitive name. Names starting with "." are skipped unless
// showHidden is true.
func ListDir(dir string, showHidden bool) ([]Entry, error) {
	des, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	entries := make([]Entry, 0, len(des))
	for _, de := range des {
		name := de.Name()
		if !showHidden && strings.HasPrefix(name, ".") {
			continue
		}
		e := Entry{Name: name, Path: filepath.Join(dir, name)}
		info, err := de.Info()
		if err != nil {
			// The entry vanished between ReadDir and Info; skip it.
			continue
		}
		e.Mode = info.Mode()
		e.Size = info.Size()
		e.IsDir = info.IsDir()
		if info.Mode()&fs.ModeSymlink != 0 {
			e.IsLink = true
			e.LinkTarget, _ = os.Readlink(e.Path)
			if target, err := os.Stat(e.Path); err == nil {
				e.IsDir = target.IsDir()
			}
		}
		entries = append(entries, e)
	}
	sort.SliceStable(entries, func(i, j int) bool {
		if entries[i].IsDir != entries[j].IsDir {
			return entries[i].IsDir
		}
		a, b := strings.ToLower(entries[i].Name), strings.ToLower(entries[j].Name)
		if a != b {
			return a < b
		}
		return entries[i].Name < entries[j].Name
	})
	return entries, nil
}
