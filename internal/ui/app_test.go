package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func key(s string) tea.KeyMsg {
	switch s {
	case "tab":
		return tea.KeyMsg{Type: tea.KeyTab}
	case "esc":
		return tea.KeyMsg{Type: tea.KeyEsc}
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	case "space":
		return tea.KeyMsg{Type: tea.KeySpace, Runes: []rune{' '}}
	case "ctrl+d":
		return tea.KeyMsg{Type: tea.KeyCtrlD}
	case "ctrl+u":
		return tea.KeyMsg{Type: tea.KeyCtrlU}
	case "backspace":
		return tea.KeyMsg{Type: tea.KeyBackspace}
	}
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}

// press sends each key to the model and returns the final model.
func press(t *testing.T, m Model, keys ...string) Model {
	t.Helper()
	for _, k := range keys {
		next, _ := m.Update(key(k))
		m = next.(Model)
	}
	return m
}

func sized(m Model) Model {
	next, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 30})
	return next.(Model)
}

func TestNewStartsBothPanelsInStartDir(t *testing.T) {
	dir := t.TempDir()
	m := New(dir, Options{})

	if got := m.Panel(0).Dir(); got != dir {
		t.Fatalf("left panel dir = %q, want %q", got, dir)
	}
	if got := m.Panel(1).Dir(); got != dir {
		t.Fatalf("right panel dir = %q, want %q", got, dir)
	}
	if m.Active() != 0 {
		t.Fatalf("active panel = %d, want 0", m.Active())
	}
}

func TestTabSwitchesActivePanel(t *testing.T) {
	m := New(t.TempDir(), Options{})

	m = press(t, m, "tab")
	if m.Active() != 1 {
		t.Fatalf("after tab active = %d, want 1", m.Active())
	}
	m = press(t, m, "tab")
	if m.Active() != 0 {
		t.Fatalf("after second tab active = %d, want 0", m.Active())
	}
}

func TestQuitKeyReturnsQuitCommand(t *testing.T) {
	m := New(t.TempDir(), Options{})

	_, cmd := m.Update(key("q"))
	if cmd == nil {
		t.Fatal("q returned nil command, want tea.Quit")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Fatalf("q command produced %T, want tea.QuitMsg", cmd())
	}
}

func TestViewShowsCurrentPathInBothPanels(t *testing.T) {
	dir := t.TempDir()
	m := sized(New(dir, Options{}))

	view := m.View()
	if n := strings.Count(view, dir); n < 2 {
		t.Fatalf("view shows path %d times, want at least 2:\n%s", n, view)
	}
}
