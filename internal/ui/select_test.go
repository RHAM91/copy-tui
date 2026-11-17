package ui

import (
	"strings"
	"testing"
)

func selectionNames(m Model) []string {
	var out []string
	for _, e := range m.Panel(m.Active()).Selection() {
		out = append(out, e.Name)
	}
	return out
}

func TestSelectionDefaultsToCursorEntry(t *testing.T) {
	m := sized(New(tree(t, "a", "b"), Options{}))

	m = press(t, m, "j")
	if got := strings.Join(selectionNames(m), ","); got != "b" {
		t.Fatalf("selection = %q, want b", got)
	}
}

func TestSelectionOfEmptyDirectoryIsEmpty(t *testing.T) {
	m := sized(New(tree(t), Options{}))
	if got := selectionNames(m); len(got) != 0 {
		t.Fatalf("selection = %v, want empty", got)
	}
}

func TestMTogglesMarkAndMovesDown(t *testing.T) {
	m := sized(New(tree(t, "a", "b", "c"), Options{}))

	m = press(t, m, "m", "j", "m")
	if got := strings.Join(selectionNames(m), ","); got != "a,c" {
		t.Fatalf("selection = %q, want a,c", got)
	}
	m = press(t, m, "g", "g", "m")
	if got := strings.Join(selectionNames(m), ","); got != "c" {
		t.Fatalf("selection after unmarking a = %q, want c", got)
	}
}

func TestVisualModeSelectsRange(t *testing.T) {
	m := sized(New(tree(t, "a", "b", "c", "d"), Options{}))

	m = press(t, m, "j", "v", "j", "j")
	if !m.Panel(0).Visual() {
		t.Fatal("expected visual mode")
	}
	if got := strings.Join(selectionNames(m), ","); got != "b,c,d" {
		t.Fatalf("visual selection = %q, want b,c,d", got)
	}
	m = press(t, m, "k", "k", "k")
	if got := strings.Join(selectionNames(m), ","); got != "a,b" {
		t.Fatalf("visual selection upwards = %q, want a,b", got)
	}
}

func TestLeavingVisualModeKeepsRangeMarked(t *testing.T) {
	m := sized(New(tree(t, "a", "b", "c", "d"), Options{}))

	m = press(t, m, "v", "j", "v", "G")
	if m.Panel(0).Visual() {
		t.Fatal("second v should leave visual mode")
	}
	if got := strings.Join(selectionNames(m), ","); got != "a,b" {
		t.Fatalf("selection after leaving visual = %q, want a,b", got)
	}
}

func TestEscClearsMarksAndVisual(t *testing.T) {
	m := sized(New(tree(t, "a", "b", "c"), Options{}))

	m = press(t, m, "m", "v", "j", "esc")
	if m.Panel(0).Visual() {
		t.Fatal("esc should leave visual mode")
	}
	if got := strings.Join(selectionNames(m), ","); got != "c" {
		t.Fatalf("selection after esc = %q, want cursor entry c", got)
	}
}

func TestChangingDirectoryClearsMarks(t *testing.T) {
	m := sized(New(tree(t, "sub/x", "sub/y", "file"), Options{}))

	m = press(t, m, "j", "m", "k", "l", "h")
	if got := strings.Join(selectionNames(m), ","); got != "sub" {
		t.Fatalf("selection after cd = %q, want only cursor entry sub", got)
	}
}

func TestMarksArePerPanel(t *testing.T) {
	m := sized(New(tree(t, "a", "b"), Options{}))

	m = press(t, m, "m", "m", "tab")
	if got := strings.Join(selectionNames(m), ","); got != "a" {
		t.Fatalf("right panel selection = %q, want cursor entry a", got)
	}
}

func TestFooterShowsSelectedCount(t *testing.T) {
	m := sized(New(tree(t, "a", "b", "c"), Options{}))

	m = press(t, m, "m", "m")
	if view := m.View(); !strings.Contains(view, "2 selected") {
		t.Fatalf("view lacks selected count:\n%s", view)
	}
}
