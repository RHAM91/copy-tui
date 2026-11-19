package ui

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/RHAM91/copy-tui/internal/fsops"
)

// drive runs cmd and feeds the resulting messages back into the model until
// no command is left or the model waits for the user (conflict prompt).
func drive(t *testing.T, m Model, cmd tea.Cmd) Model {
	t.Helper()
	for cmd != nil {
		done := make(chan tea.Msg, 1)
		go func(c tea.Cmd) { done <- c() }(cmd)
		var msg tea.Msg
		select {
		case msg = <-done:
		case <-time.After(5 * time.Second):
			t.Fatal("command did not finish in time")
		}
		if msg == nil {
			return m
		}
		next, nextCmd := m.Update(msg)
		m, cmd = next.(Model), nextCmd
		if m.ConflictPending() {
			return m
		}
	}
	return m
}

// act presses keys and drives every command they produce.
func act(t *testing.T, m Model, keys ...string) Model {
	t.Helper()
	for _, k := range keys {
		next, cmd := m.Update(key(k))
		m = drive(t, next.(Model), cmd)
	}
	return m
}

// twoPanels returns a model whose left panel shows root and whose right
// panel shows root/dst.
func twoPanels(t *testing.T, paths ...string) (Model, string) {
	t.Helper()
	root := tree(t, append([]string{"dst/"}, paths...)...)
	m := sized(New(root, Options{}))
	m = press(t, m, "tab", "l", "tab")
	if m.Panel(1).Dir() != filepath.Join(root, "dst") {
		t.Fatalf("right panel in %q", m.Panel(1).Dir())
	}
	return m, root
}

func writeTo(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func contentOf(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestSpaceCopiesCursorEntryToOtherPanel(t *testing.T) {
	m, root := twoPanels(t, "file.txt")

	m = act(t, m, "j", "space")
	if m.Busy() {
		t.Fatal("operation still running")
	}
	if got := contentOf(t, filepath.Join(root, "dst", "file.txt")); got != "x" {
		t.Fatalf("copied content = %q", got)
	}
	if n := len(m.Panel(1).Entries()); n != 1 {
		t.Fatalf("right panel not refreshed: %d entries", n)
	}
	if m.StatusIsError() || !strings.Contains(m.Status(), "Copied 1 item") {
		t.Fatalf("status = %q", m.Status())
	}
}

func TestCopyWorksFromRightToLeft(t *testing.T) {
	m, root := twoPanels(t)
	writeTo(t, filepath.Join(root, "dst", "back.txt"), "back")

	m = press(t, m, "tab")
	if err := m.panels[1].reload(); err != nil {
		t.Fatal(err)
	}
	m = act(t, m, "space")
	if got := contentOf(t, filepath.Join(root, "back.txt")); got != "back" {
		t.Fatalf("copied content = %q", got)
	}
}

func TestSpaceCopiesAllMarkedEntriesAndClearsSelection(t *testing.T) {
	m, root := twoPanels(t, "a.txt", "b.txt", "c.txt")

	// Entries: dst, a.txt, b.txt, c.txt.
	m = act(t, m, "j", "m", "j", "m", "space")
	for _, name := range []string{"a.txt", "c.txt"} {
		if _, err := os.Stat(filepath.Join(root, "dst", name)); err != nil {
			t.Fatalf("%s not copied: %v", name, err)
		}
	}
	if _, err := os.Stat(filepath.Join(root, "dst", "b.txt")); err == nil {
		t.Fatal("unmarked b.txt was copied")
	}
	if n := m.Panel(0).selectedCount(); n != 0 {
		t.Fatalf("selection not cleared: %d", n)
	}
}

func TestSpaceInEmptyDirectoryDoesNothing(t *testing.T) {
	m := sized(New(tree(t), Options{}))
	next, cmd := m.Update(key("space"))
	if cmd != nil || next.(Model).Busy() {
		t.Fatal("space in an empty directory started an operation")
	}
}

func TestConflictDialogChoices(t *testing.T) {
	tests := []struct {
		name   string
		choice string
		want   map[string]string
	}{
		{name: "o overwrites", choice: "o", want: map[string]string{"file.txt": "x"}},
		{name: "s skips", choice: "s", want: map[string]string{"file.txt": "old"}},
		{name: "r renames", choice: "r", want: map[string]string{"file.txt": "old", "file (1).txt": "x"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, root := twoPanels(t, "file.txt")
			writeTo(t, filepath.Join(root, "dst", "file.txt"), "old")

			m = act(t, m, "j", "space")
			if !m.ConflictPending() {
				t.Fatal("expected conflict dialog")
			}
			if view := m.View(); !strings.Contains(view, "already exists") {
				t.Fatalf("conflict dialog not rendered:\n%s", view)
			}
			m = act(t, m, tt.choice)
			if m.Busy() || m.ConflictPending() {
				t.Fatal("operation did not finish")
			}
			for name, want := range tt.want {
				if got := contentOf(t, filepath.Join(root, "dst", name)); got != want {
					t.Fatalf("%s = %q, want %q", name, got, want)
				}
			}
		})
	}
}

func TestConflictApplyToAllAsksOnce(t *testing.T) {
	m, root := twoPanels(t, "a.txt", "b.txt")
	writeTo(t, filepath.Join(root, "dst", "a.txt"), "old")
	writeTo(t, filepath.Join(root, "dst", "b.txt"), "old")

	m = act(t, m, "j", "m", "m", "space")
	if !m.ConflictPending() {
		t.Fatal("expected conflict dialog")
	}
	m = act(t, m, "a", "o")
	if m.ConflictPending() {
		t.Fatal("apply to all asked again")
	}
	for _, name := range []string{"a.txt", "b.txt"} {
		if got := contentOf(t, filepath.Join(root, "dst", name)); got != "x" {
			t.Fatalf("%s = %q, want overwritten", name, got)
		}
	}
}

func TestConflictDialogEnterConfirmsHighlightedOption(t *testing.T) {
	m, root := twoPanels(t, "file.txt")
	writeTo(t, filepath.Join(root, "dst", "file.txt"), "old")

	// Options: overwrite, skip, rename. Move right twice to rename.
	m = act(t, m, "j", "space", "l", "l", "enter")
	if got := contentOf(t, filepath.Join(root, "dst", "file (1).txt")); got != "x" {
		t.Fatalf("renamed copy = %q", got)
	}
}

func TestEscInConflictDialogCancelsOperation(t *testing.T) {
	m, root := twoPanels(t, "file.txt")
	writeTo(t, filepath.Join(root, "dst", "file.txt"), "old")

	m = act(t, m, "j", "space", "esc")
	if m.Busy() || m.ConflictPending() {
		t.Fatal("operation still running after esc")
	}
	if got := contentOf(t, filepath.Join(root, "dst", "file.txt")); got != "old" {
		t.Fatalf("file.txt = %q, want old", got)
	}
	if !strings.Contains(m.Status(), "cancelled") {
		t.Fatalf("status = %q, want cancelled", m.Status())
	}
}

func TestEscCancelsRunningOperation(t *testing.T) {
	m := sized(New(tree(t, "f"), Options{}))
	started := make(chan struct{})
	cmd := m.startOp("Copying", func(ctx context.Context, _ fsops.ConflictResolver, _ func(fsops.Progress)) (string, error) {
		close(started)
		<-ctx.Done()
		return "", ctx.Err()
	})
	<-started
	if !m.Busy() {
		t.Fatal("expected busy model")
	}
	if view := m.View(); !strings.Contains(view, "Copying") {
		t.Fatalf("progress modal not rendered:\n%s", view)
	}

	next, _ := m.Update(key("esc"))
	m = drive(t, next.(Model), cmd)
	if m.Busy() {
		t.Fatal("still busy after esc")
	}
	if !strings.Contains(m.Status(), "cancelled") {
		t.Fatalf("status = %q, want cancelled", m.Status())
	}
}

func TestKeysAreIgnoredWhileBusy(t *testing.T) {
	m := sized(New(tree(t, "a", "b"), Options{}))
	release := make(chan struct{})
	cmd := m.startOp("Copying", func(ctx context.Context, _ fsops.ConflictResolver, _ func(fsops.Progress)) (string, error) {
		<-release
		return "done", nil
	})

	next, quit := m.Update(key("q"))
	m = next.(Model)
	if quit != nil {
		if _, ok := quit().(tea.QuitMsg); ok {
			t.Fatal("q quit while busy")
		}
	}
	m = press(t, m, "j")
	if got := cursorName(m); got != "a" {
		t.Fatalf("j moved cursor to %q while busy", got)
	}
	close(release)
	m = drive(t, m, cmd)
	if m.Busy() {
		t.Fatal("operation did not finish")
	}
}

func TestXMovesSelectionToOtherPanel(t *testing.T) {
	m, root := twoPanels(t, "a.txt", "b.txt")

	m = act(t, m, "j", "m", "m", "x", "enter")
	if m.Busy() {
		t.Fatal("operation still running")
	}
	for _, name := range []string{"a.txt", "b.txt"} {
		if got := contentOf(t, filepath.Join(root, "dst", name)); got != "x" {
			t.Fatalf("%s = %q", name, got)
		}
		if _, err := os.Stat(filepath.Join(root, name)); err == nil {
			t.Fatalf("%s still in source", name)
		}
	}
	if n := len(m.Panel(0).Entries()); n != 1 {
		t.Fatalf("left panel not refreshed: %d entries", n)
	}
	if !strings.Contains(m.Status(), "Moved 2 items") {
		t.Fatalf("status = %q", m.Status())
	}
}

func TestXAsksBeforeMovingAndCancelKeepsEverything(t *testing.T) {
	for _, cancel := range []string{"esc", "n"} {
		m, root := twoPanels(t, "work/", "work/inner.txt")

		m = act(t, m, "j", "x")
		if !m.ConfirmPending() {
			t.Fatal("x should ask before moving")
		}
		view := m.View()
		for _, want := range []string{"Move 1 item", "work", "dst", "Move"} {
			if !strings.Contains(view, want) {
				t.Fatalf("confirmation should show %q, got:\n%s", want, view)
			}
		}
		m = act(t, m, cancel)

		if m.ConfirmPending() || m.Busy() {
			t.Fatalf("%s should close the confirmation without moving", cancel)
		}
		if _, err := os.Stat(filepath.Join(root, "work", "inner.txt")); err != nil {
			t.Fatalf("%s: source was moved: %v", cancel, err)
		}
		if _, err := os.Stat(filepath.Join(root, "dst", "work")); err == nil {
			t.Fatalf("%s: folder reached the destination", cancel)
		}
	}
}

func TestXAsksOnConflict(t *testing.T) {
	m, root := twoPanels(t, "a.txt")
	writeTo(t, filepath.Join(root, "dst", "a.txt"), "old")

	m = act(t, m, "j", "x", "y")
	if !m.ConflictPending() {
		t.Fatal("expected conflict dialog")
	}
	m = act(t, m, "s")
	if got := contentOf(t, filepath.Join(root, "dst", "a.txt")); got != "old" {
		t.Fatalf("dst overwritten: %q", got)
	}
	if _, err := os.Stat(filepath.Join(root, "a.txt")); err != nil {
		t.Fatal("skipped source was removed")
	}
}
