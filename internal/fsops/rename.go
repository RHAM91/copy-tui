package fsops

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// ErrInvalidName is returned for names that cannot be a single path element.
var ErrInvalidName = errors.New("invalid name")

// ValidateName checks that name can be used as a file or folder name on
// this operating system.
func ValidateName(name string) error { return validateName(name, runtime.GOOS) }

// windowsReserved are device names Windows refuses as file names, with or
// without an extension.
var windowsReserved = map[string]bool{
	"CON": true, "PRN": true, "AUX": true, "NUL": true,
	"COM1": true, "COM2": true, "COM3": true, "COM4": true, "COM5": true, "COM6": true, "COM7": true, "COM8": true, "COM9": true,
	"LPT1": true, "LPT2": true, "LPT3": true, "LPT4": true, "LPT5": true, "LPT6": true, "LPT7": true, "LPT8": true, "LPT9": true,
}

func validateName(name, goos string) error {
	switch {
	case strings.TrimSpace(name) == "":
		return fmt.Errorf("%w: name is empty", ErrInvalidName)
	case name == "." || name == "..":
		return fmt.Errorf("%w: %q", ErrInvalidName, name)
	case strings.ContainsAny(name, "/\x00"):
		return fmt.Errorf("%w: %q contains '/'", ErrInvalidName, name)
	}
	if goos != "windows" {
		return nil
	}
	if i := strings.IndexAny(name, `\:*?"<>|`); i >= 0 {
		return fmt.Errorf("%w: %q contains %q, not allowed on Windows", ErrInvalidName, name, name[i])
	}
	if strings.HasSuffix(name, ".") || strings.HasSuffix(name, " ") {
		return fmt.Errorf("%w: %q ends with a dot or space, not allowed on Windows", ErrInvalidName, name)
	}
	stem, _, _ := strings.Cut(name, ".")
	if windowsReserved[strings.ToUpper(stem)] {
		return fmt.Errorf("%w: %q is a reserved name on Windows", ErrInvalidName, name)
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
	if dstInfo, err := os.Lstat(dst); err == nil {
		// On case-insensitive file systems (macOS, Windows) a case-only
		// rename finds the source itself; that is not a conflict.
		srcInfo, srcErr := os.Lstat(path)
		if srcErr != nil || !os.SameFile(srcInfo, dstInfo) {
			return fmt.Errorf("%q already exists: %w", newName, fs.ErrExist)
		}
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
