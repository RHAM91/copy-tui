package ui

import (
	"strings"
	"testing"
)

func lastLine(view string) string {
	lines := strings.Split(view, "\n")
	return lines[len(lines)-1]
}

func TestHelpBarShowsMainCommands(t *testing.T) {
	m := sized(New(tree(t, "a"), Options{}))
	bar := lastLine(m.View())
	for _, want := range []string{"space", "copy", "x", "move", "d", "trash", "?", "help", "q", "quit"} {
		if !strings.Contains(bar, want) {
			t.Fatalf("help bar lacks %q: %q", want, bar)
		}
	}
}

func TestHelpBarAdaptsToMode(t *testing.T) {
	m := sized(New(tree(t, "a", "b"), Options{}))

	if bar := lastLine(press(t, m, "/").View()); !strings.Contains(bar, "enter") || !strings.Contains(bar, "esc") {
		t.Fatalf("search help bar = %q", bar)
	}
	if bar := lastLine(press(t, m, "v").View()); !strings.Contains(bar, "keep selection") {
		t.Fatalf("visual help bar = %q", bar)
	}
}

func TestQuestionMarkTogglesFullHelp(t *testing.T) {
	m := sized(New(tree(t, "a", "b"), Options{}))

	m = press(t, m, "?")
	if !m.HelpOpen() {
		t.Fatal("? should open the help")
	}
	view := m.View()
	for _, want := range []string{"Navigation", "Selection", "Files", "delete permanently", "--no-icons"} {
		if !strings.Contains(view, want) {
			t.Fatalf("help lacks %q:\n%s", want, view)
		}
	}
	m = press(t, m, "j")
	if got := cursorName(m); got != "a" {
		t.Fatalf("j moved the cursor behind the help to %q", got)
	}
	m = press(t, m, "esc")
	if m.HelpOpen() {
		t.Fatal("esc should close the help")
	}
	m = press(t, m, "?", "?")
	if m.HelpOpen() {
		t.Fatal("? should toggle the help closed")
	}
}

func TestStatusShowsErrorsInRed(t *testing.T) {
	m := sized(New(tree(t, "a"), Options{}))
	m = press(t, m, "/", "z", "enter")
	if !m.StatusIsError() {
		t.Fatal("expected error status")
	}
	if view := m.View(); !strings.Contains(view, "pattern not found: z") {
		t.Fatalf("error not rendered:\n%s", view)
	}
}
