package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/RHAM91/copy-tui/internal/config"
)

// themeModel opens the UI with a config store in a temp dir, optionally
// holding content, and restores the default theme after the test.
func themeModel(t *testing.T, content string) (Model, *config.Store) {
	t.Helper()
	store := config.NewStore(filepath.Join(t.TempDir(), "config.toml"))
	if content != "" {
		if err := os.WriteFile(store.Path(), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	t.Cleanup(func() { applyTheme(themes[0]) })
	return sized(New(tree(t, "a.txt"), Options{Config: store})), store
}

func savedTheme(t *testing.T, s *config.Store) string {
	t.Helper()
	c, err := s.Load()
	if err != nil {
		t.Fatal(err)
	}
	return c.Theme
}

func TestStartupAppliesTheConfiguredTheme(t *testing.T) {
	m, _ := themeModel(t, `theme = "tokyo-night"`)
	if theme.ID != "tokyo-night" {
		t.Fatalf("theme = %q, want tokyo-night", theme.ID)
	}
	if m.Status() != "" {
		t.Fatalf("status = %q, want none", m.Status())
	}
}

func TestStartupWithUnknownThemeFallsBackAndSaysSo(t *testing.T) {
	m, _ := themeModel(t, `theme = "solarized"`)
	if theme.ID != DefaultThemeID {
		t.Fatalf("theme = %q, want the default", theme.ID)
	}
	if !strings.Contains(m.Status(), `"solarized"`) || !m.StatusIsError() {
		t.Fatalf("status = %q (err=%v), want an error naming the theme", m.Status(), m.StatusIsError())
	}
}

func TestStartupWithCorruptConfigUsesDefaultAndReportsIt(t *testing.T) {
	m, store := themeModel(t, "theme = \n[[[")
	if theme.ID != DefaultThemeID {
		t.Fatalf("theme = %q, want the default", theme.ID)
	}
	if !strings.Contains(m.Status(), store.Path()) || !m.StatusIsError() {
		t.Fatalf("status = %q (err=%v), want an error naming the file", m.Status(), m.StatusIsError())
	}
}

func TestCtrlTPreviewsLiveAndEnterSaves(t *testing.T) {
	m, store := themeModel(t, "")

	m = press(t, m, "ctrl+t")
	if !m.ThemePickerOpen() {
		t.Fatal("ctrl+t should open the theme picker")
	}
	view := m.View()
	for _, th := range Themes() {
		if !strings.Contains(view, th.Name) {
			t.Fatalf("picker should list %q, got:\n%s", th.Name, view)
		}
	}

	m = press(t, m, "j")
	if theme.ID != "kanagawa-wave" {
		t.Fatalf("after j theme = %q, want a live preview of kanagawa-wave", theme.ID)
	}
	if got := savedTheme(t, store); got != "" {
		t.Fatalf("preview saved %q before enter", got)
	}
	m = press(t, m, "j", "j", "k", "enter")

	if m.ThemePickerOpen() {
		t.Fatal("enter should close the picker")
	}
	if theme.ID != "kanagawa-dragon" {
		t.Fatalf("theme = %q, want kanagawa-dragon", theme.ID)
	}
	if got := savedTheme(t, store); got != "kanagawa-dragon" {
		t.Fatalf("saved theme = %q, want kanagawa-dragon", got)
	}
	if !strings.Contains(m.Status(), "Kanagawa Dragon") || m.StatusIsError() {
		t.Fatalf("status = %q (err=%v)", m.Status(), m.StatusIsError())
	}
}

func TestCtrlTEscRestoresThePreviousTheme(t *testing.T) {
	m, store := themeModel(t, `theme = "tokyo-night"`)

	m = press(t, m, "ctrl+t")
	if !strings.Contains(m.View(), "▸ Tokyo Night") {
		t.Fatalf("cursor should start on the current theme:\n%s", m.View())
	}
	m = press(t, m, "k")
	if theme.ID != "kanagawa-dragon" {
		t.Fatalf("after k theme = %q", theme.ID)
	}
	m = press(t, m, "esc")

	if m.ThemePickerOpen() {
		t.Fatal("esc should close the picker")
	}
	if theme.ID != "tokyo-night" {
		t.Fatalf("theme = %q, want tokyo-night restored", theme.ID)
	}
	if got := savedTheme(t, store); got != "tokyo-night" {
		t.Fatalf("saved theme changed to %q", got)
	}
}

func TestThemeSaveErrorKeepsThemeAndReportsIt(t *testing.T) {
	m, store := themeModel(t, "")
	m = press(t, m, "ctrl+t", "j")
	if err := os.WriteFile(store.Path(), []byte("theme = \n[[["), 0o644); err != nil {
		t.Fatal(err)
	}
	m = press(t, m, "enter")

	if theme.ID != "kanagawa-wave" {
		t.Fatalf("theme = %q, want the chosen one kept for this session", theme.ID)
	}
	if !m.StatusIsError() || !strings.Contains(m.Status(), "not saved") {
		t.Fatalf("status = %q (err=%v), want a not-saved error", m.Status(), m.StatusIsError())
	}
}
