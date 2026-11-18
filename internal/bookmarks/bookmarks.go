// Package bookmarks stores named pairs of panel directories in a JSON file.
package bookmarks

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// Bookmark is a named pair of directories, one per panel.
type Bookmark struct {
	Name  string `json:"name"`
	Left  string `json:"left"`
	Right string `json:"right"`
}

var (
	// ErrEmptyName is returned for a blank bookmark name.
	ErrEmptyName = errors.New("bookmark name cannot be empty")
	// ErrExists is returned when renaming onto another bookmark's name.
	ErrExists = errors.New("a bookmark with that name already exists")
	// ErrNotFound is returned when the named bookmark does not exist.
	ErrNotFound = errors.New("bookmark not found")
)

// fileFormat is the on-disk layout. Version lets later releases migrate it.
type fileFormat struct {
	Version   int        `json:"version"`
	Bookmarks []Bookmark `json:"bookmarks"`
}

const formatVersion = 1

// Store reads and writes bookmarks at a file path. Every call re-reads the
// file, so two running instances never overwrite each other's bookmarks
// with a stale copy.
type Store struct {
	path string
}

// NewStore returns a store backed by the file at path. The file and its
// directory are created on the first write.
func NewStore(path string) *Store { return &Store{path: path} }

// DefaultPath is <user config dir>/copiar/bookmarks.json, for example
// ~/.config/copiar/bookmarks.json on Linux.
func DefaultPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "copiar", "bookmarks.json"), nil
}

// Path returns the file the store uses.
func (s *Store) Path() string { return s.path }

// CleanName trims name and rejects a blank one.
func CleanName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", ErrEmptyName
	}
	return name, nil
}

// List returns every bookmark sorted by name, ignoring case.
func (s *Store) List() ([]Bookmark, error) {
	list, err := s.load()
	if err != nil {
		return nil, err
	}
	slices.SortStableFunc(list, func(a, b Bookmark) int {
		return strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name))
	})
	return list, nil
}

// Get returns the bookmark whose name matches, ignoring case.
func (s *Store) Get(name string) (Bookmark, bool, error) {
	list, err := s.load()
	if err != nil {
		return Bookmark{}, false, err
	}
	if i := index(list, name); i >= 0 {
		return list[i], true, nil
	}
	return Bookmark{}, false, nil
}

// Put adds b, replacing a bookmark with the same name (ignoring case).
func (s *Store) Put(b Bookmark) error {
	name, err := CleanName(b.Name)
	if err != nil {
		return err
	}
	b.Name = name
	list, err := s.load()
	if err != nil {
		return err
	}
	if i := index(list, name); i >= 0 {
		list[i] = b
	} else {
		list = append(list, b)
	}
	return s.save(list)
}

// Rename changes the name of a bookmark. Changing only the case is allowed;
// taking another bookmark's name is not.
func (s *Store) Rename(oldName, newName string) error {
	newName, err := CleanName(newName)
	if err != nil {
		return err
	}
	list, err := s.load()
	if err != nil {
		return err
	}
	i := index(list, oldName)
	if i < 0 {
		return ErrNotFound
	}
	if j := index(list, newName); j >= 0 && j != i {
		return ErrExists
	}
	list[i].Name = newName
	return s.save(list)
}

// Delete removes the named bookmark.
func (s *Store) Delete(name string) error {
	list, err := s.load()
	if err != nil {
		return err
	}
	i := index(list, name)
	if i < 0 {
		return ErrNotFound
	}
	return s.save(slices.Delete(list, i, i+1))
}

func index(list []Bookmark, name string) int {
	name = strings.TrimSpace(name)
	return slices.IndexFunc(list, func(b Bookmark) bool { return strings.EqualFold(b.Name, name) })
}

// load reads the file. A missing file is an empty list; an unreadable or
// corrupt one is an error, so callers never save over it.
func (s *Store) load() ([]Bookmark, error) {
	data, err := os.ReadFile(s.path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read bookmarks: %w", err)
	}
	var f fileFormat
	if err := json.Unmarshal(data, &f); err != nil {
		return nil, fmt.Errorf("bookmarks file %s is corrupt: %w", s.path, err)
	}
	return f.Bookmarks, nil
}

// save writes list to a temporary file and renames it over the real one, so
// a crash never leaves a half-written file.
func (s *Store) save(list []Bookmark) error {
	if list == nil {
		list = []Bookmark{}
	}
	data, err := json.MarshalIndent(fileFormat{Version: formatVersion, Bookmarks: list}, "", "  ")
	if err != nil {
		return err
	}
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("save bookmarks: %w", err)
	}
	tmp, err := os.CreateTemp(dir, ".bookmarks-*.json")
	if err != nil {
		return fmt.Errorf("save bookmarks: %w", err)
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(append(data, '\n')); err != nil {
		tmp.Close()
		return fmt.Errorf("save bookmarks: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("save bookmarks: %w", err)
	}
	if err := os.Rename(tmp.Name(), s.path); err != nil {
		return fmt.Errorf("save bookmarks: %w", err)
	}
	return nil
}
