// Package config reads and writes the user preferences in config.toml.
package config

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// Config holds the preferences copiar reads at startup.
type Config struct {
	// Theme is the id of the color theme, "" when none was chosen.
	Theme string `toml:"theme"`
}

// Store reads and writes config.toml at a file path.
type Store struct {
	path string
}

// NewStore returns a store backed by the file at path. The file and its
// directory are created on the first write.
func NewStore(path string) *Store { return &Store{path: path} }

// DefaultPath is <user config dir>/copiar/config.toml, for example
// ~/.config/copiar/config.toml on Linux.
func DefaultPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "copiar", "config.toml"), nil
}

// Path returns the file the store uses.
func (s *Store) Path() string { return s.path }

// Load reads the preferences. A missing file gives an empty Config.
func (s *Store) Load() (Config, error) {
	var c Config
	data, err := s.read()
	if err != nil || data == nil {
		return c, err
	}
	if _, err := toml.Decode(string(data), &c); err != nil {
		return Config{}, s.corrupt(err)
	}
	return c, nil
}

// SetTheme saves id as the theme, keeping every other key in the file. A
// corrupt file is reported and left untouched.
func (s *Store) SetTheme(id string) error {
	keys := map[string]any{}
	data, err := s.read()
	if err != nil {
		return err
	}
	if _, err := toml.Decode(string(data), &keys); err != nil {
		return s.corrupt(err)
	}
	keys["theme"] = id
	var buf bytes.Buffer
	if err := toml.NewEncoder(&buf).Encode(keys); err != nil {
		return err
	}
	return s.write(buf.Bytes())
}

// read returns the file content, or nil when it does not exist.
func (s *Store) read() ([]byte, error) {
	data, err := os.ReadFile(s.path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	return data, nil
}

func (s *Store) corrupt(err error) error {
	return fmt.Errorf("config file %s is corrupt: %w", s.path, err)
}

// write saves data to a temporary file and renames it over the real one, so
// a crash never leaves a half-written file.
func (s *Store) write(data []byte) error {
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("save config: %w", err)
	}
	tmp, err := os.CreateTemp(dir, ".config-*.toml")
	if err != nil {
		return fmt.Errorf("save config: %w", err)
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("save config: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("save config: %w", err)
	}
	if err := os.Rename(tmp.Name(), s.path); err != nil {
		return fmt.Errorf("save config: %w", err)
	}
	return nil
}
