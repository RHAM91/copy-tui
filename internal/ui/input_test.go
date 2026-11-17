package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// typeText sends each rune of s as a key press.
func typeText(t *testing.T, m Model, s string) Model {
	t.Helper()
	for _, r := range s {
		m = press(t, m, string(r))
	}
	return m
}

func TestRRenamesEntryUnderCursor(t *testing.T) {
	root := tree(t, "old.txt")
	m := sized(New(root, Options{}))

	m = press(t, m, "r")
	if !m.InputPending() {
		t.Fatal("r should open the rename prompt")
	}
	if got := m.InputValue(); got != "old.txt" {
		t.Fatalf("prompt prefilled with %q, want old.txt", got)
	}
	m = press(t, m, "ctrl+u")
	m = typeText(t, m, "new.txt")
	m = press(t, m, "enter")

	if m.InputPending() {
		t.Fatal("prompt still open after enter")
	}
	if _, err := os.Stat(filepath.Join(root, "new.txt")); err != nil {
		t.Fatalf("new.txt missing: %v", err)
	}
	if got := cursorName(m); got != "new.txt" {
		t.Fatalf("cursor on %q, want new.txt", got)
	}
	if !strings.Contains(m.Status(), "new.txt") {
		t.Fatalf("status = %q", m.Status())
	}
}

func TestRenameToExistingNameKeepsPromptWithError(t *testing.T) {
	root := tree(t, "a.txt", "b.txt")
	m := sized(New(root, Options{}))

	m = press(t, m, "r", "ctrl+u")
	m = typeText(t, m, "b.txt")
	m = press(t, m, "enter")
	if !m.InputPending() {
		t.Fatal("prompt closed on error")
	}
	if view := m.View(); !strings.Contains(view, "exists") {
		t.Fatalf("error not shown in prompt:\n%s", view)
	}
	if got := contentOf(t, filepath.Join(root, "b.txt")); got != "x" {
		t.Fatalf("b.txt changed: %q", got)
	}
	m = press(t, m, "esc")
	if m.InputPending() {
		t.Fatal("esc should close the prompt")
	}
	if _, err := os.Stat(filepath.Join(root, "a.txt")); err != nil {
		t.Fatal("a.txt renamed after esc")
	}
}

func TestRInEmptyDirectoryDoesNothing(t *testing.T) {
	m := press(t, sized(New(tree(t), Options{})), "r")
	if m.InputPending() {
		t.Fatal("r opened a prompt without an entry")
	}
}

func TestACreatesFolderAndFocusesIt(t *testing.T) {
	root := tree(t, "zzz.txt")
	m := sized(New(root, Options{}))

	m = press(t, m, "a")
	if !m.InputPending() || m.InputValue() != "" {
		t.Fatalf("a should open an empty prompt, got pending=%v value=%q", m.InputPending(), m.InputValue())
	}
	m = typeText(t, m, "docs")
	m = press(t, m, "enter")

	if info, err := os.Stat(filepath.Join(root, "docs")); err != nil || !info.IsDir() {
		t.Fatalf("docs not created: %v", err)
	}
	if got := cursorName(m); got != "docs" {
		t.Fatalf("cursor on %q, want docs", got)
	}
}

func TestACreatesFolderInActivePanelOnly(t *testing.T) {
	m, root := twoPanels(t)
	m = press(t, m, "tab", "a")
	m = typeText(t, m, "inside")
	m = press(t, m, "enter")
	if _, err := os.Stat(filepath.Join(root, "dst", "inside")); err != nil {
		t.Fatalf("folder not created in right panel dir: %v", err)
	}
}

func TestInvalidFolderNameShowsError(t *testing.T) {
	m := sized(New(tree(t), Options{}))
	m = press(t, m, "a")
	m = typeText(t, m, "a/b")
	m = press(t, m, "enter")
	if !m.InputPending() {
		t.Fatal("prompt closed on invalid name")
	}
	if view := m.View(); !strings.Contains(view, "invalid") {
		t.Fatalf("error not shown:\n%s", view)
	}
}

func TestKeysGoToPromptNotToPanels(t *testing.T) {
	m := sized(New(tree(t, "a", "b"), Options{}))
	// If "q" were handled as quit it would not reach the prompt value.
	m = press(t, m, "a")
	m = typeText(t, m, "qj")
	if got := m.InputValue(); got != "qj" {
		t.Fatalf("prompt value = %q, want qj", got)
	}
	if got := cursorName(m); got != "a" {
		t.Fatalf("cursor moved to %q while typing", got)
	}
}
