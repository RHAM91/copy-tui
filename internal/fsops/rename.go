package fsops

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// ErrInvalidName is returned for names that cannot be a single path element.
var ErrInvalidName = errors.New("invalid name")

// ValidateName checks that name can be used as a file or folder name.
func ValidateName(name string) error {
	switch {
	case strings.TrimSpace(name) == "":
		return fmt.Errorf("%w: name is empty", ErrInvalidName)
	case name == "." || name == "..":
		return fmt.Errorf("%w: %q", ErrInvalidName, name)
	case strings.ContainsAny(name, "/\x00"):
		return fmt.Errorf("%w: %q contains '/'", ErrInvalidName, name)
	}
	return nil
}

// Rename gives path a new name in the same directory. It never replaces an
// existing entry.
func Rename(path, newName string) error {
	if err := ValidateName(newName); err != nil {
		return err
	}
	if filepath.Base(path) == newName {
		return nil
	}
	dst := filepath.Join(filepath.Dir(path), newName)
	if _, err := os.Lstat(dst); err == nil {
		return fmt.Errorf("%q already exists: %w", newName, fs.ErrExist)
	} else if !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	return os.Rename(path, dst)
}

// Mkdir creates the folder name inside dir and returns its path.
func Mkdir(dir, name string) (string, error) {
	if err := ValidateName(name); err != nil {
		return "", err
	}
	path := filepath.Join(dir, name)
	if err := os.Mkdir(path, 0o755); err != nil {
		if errors.Is(err, fs.ErrExist) {
			return "", fmt.Errorf("%q already exists: %w", name, fs.ErrExist)
		}
		return "", err
	}
	return path, nil
}
