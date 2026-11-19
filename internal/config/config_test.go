package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	return NewStore(filepath.Join(t.TempDir(), "copiar", "config.toml"))
}

func TestLoadWithoutFileIsEmpty(t *testing.T) {
	s := newTestStore(t)
	c, err := s.Load()
	if err != nil {
		t.Fatal(err)
	}
	if c.Theme != "" {
		t.Fatalf("Theme = %q, want empty", c.Theme)
	}
}

func TestSetThemeCreatesTheFileAndLoadReadsIt(t *testing.T) {
	s := newTestStore(t)
	if err := s.SetTheme("kanagawa-wave"); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(s.Path())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `theme = "kanagawa-wave"`) {
		t.Fatalf("file content:\n%s", data)
	}
	c, err := s.Load()
	if err != nil {
		t.Fatal(err)
	}
	if c.Theme != "kanagawa-wave" {
		t.Fatalf("Theme = %q", c.Theme)
	}
}

func TestSetThemeKeepsOtherKeys(t *testing.T) {
	s := newTestStore(t)
	if err := os.MkdirAll(filepath.Dir(s.Path()), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(s.Path(), []byte("theme = \"tokyo-night\"\nfuture = 3\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := s.SetTheme("kanagawa-dragon"); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(s.Path())
	for _, want := range []string{`theme = "kanagawa-dragon"`, "future = 3"} {
		if !strings.Contains(string(data), want) {
			t.Fatalf("missing %q in:\n%s", want, data)
		}
	}
}

func TestCorruptFileIsReportedAndNeverOverwritten(t *testing.T) {
	s := newTestStore(t)
	if err := os.MkdirAll(filepath.Dir(s.Path()), 0o755); err != nil {
		t.Fatal(err)
	}
	bad := []byte("theme = \n[[[")
	if err := os.WriteFile(s.Path(), bad, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Load(); err == nil || !strings.Contains(err.Error(), s.Path()) {
		t.Fatalf("Load error = %v, want one naming the file", err)
	}
	if err := s.SetTheme("tokyo-night"); err == nil {
		t.Fatal("SetTheme should refuse to overwrite a corrupt file")
	}
	if data, _ := os.ReadFile(s.Path()); string(data) != string(bad) {
		t.Fatalf("corrupt file was changed to:\n%s", data)
	}
}

func TestSetThemeLeavesNoTempFiles(t *testing.T) {
	s := newTestStore(t)
	if err := s.SetTheme("tokyo-night"); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(filepath.Dir(s.Path()))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != "config.toml" {
		var names []string
		for _, e := range entries {
			names = append(names, e.Name())
		}
		t.Fatalf("directory holds %v, want only config.toml", names)
	}
}

func TestDefaultPathIsUnderUserConfigDir(t *testing.T) {
	dir, err := os.UserConfigDir()
	if err != nil {
		t.Skip("no user config dir:", err)
	}
	got, err := DefaultPath()
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(dir, "copiar", "config.toml"); got != want {
		t.Fatalf("DefaultPath() = %q, want %q", got, want)
	}
}
