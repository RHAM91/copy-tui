package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// syncModel opens the UI with the left panel in root and the right panel in
// root/right, the left cursor on "b.txt".
func syncModel(t *testing.T) (Model, string) {
	t.Helper()
	root := tree(t, "a.txt", "b.txt", "right/c.txt")
	m := sized(New(root, Options{}))
	if err := m.panels[1].cd(filepath.Join(root, "right"), ""); err != nil {
		t.Fatal(err)
	}
	if !m.panels[0].focus("b.txt") {
		t.Fatal("b.txt not listed")
	}
	return m, root
}

func TestEqualsDefaultsToActivePanelPathAndMirrorsCursor(t *testing.T) {
	m, root := syncModel(t)
	m.panels[1].toggleMark()

	m = press(t, m, "=")
	if !m.SyncPending() {
		t.Fatal("= should ask which panel to change")
	}
	view := m.View()
	for _, want := range []string{"Left → Right", "Right → Left", "Cancel"} {
		if !strings.Contains(view, want) {
			t.Fatalf("dialog should offer %q, got:\n%s", want, view)
		}
	}
	m = press(t, m, "enter")

	if m.SyncPending() {
		t.Fatal("enter should close the dialog")
	}
	if got := m.Panel(1).Dir(); got != root {
		t.Fatalf("right dir = %q, want the left path %q", got, root)
	}
	if got := m.Panel(0).Dir(); got != root {
		t.Fatalf("left dir changed to %q", got)
	}
	if m.Active() != 0 {
		t.Fatalf("focus moved to panel %d, want it to stay on 0", m.Active())
	}
	if e, _ := m.Panel(1).Current(); e.Name != "b.txt" {
		t.Fatalf("right cursor on %q, want b.txt like the left panel", e.Name)
	}
	if len(m.Panel(1).marks) != 0 {
		t.Fatal("the panel that changed should drop its marks")
	}
	if !strings.Contains(m.Status(), "Right panel") || m.StatusIsError() {
		t.Fatalf("status = %q (err=%v), want an info message about the right panel", m.Status(), m.StatusIsError())
	}
}

func TestEqualsFromRightPanelDefaultsToRightPath(t *testing.T) {
	m, root := syncModel(t)
	right := filepath.Join(root, "right")

	m = press(t, m, "tab", "=", "enter")

	if got := m.Panel(0).Dir(); got != right {
		t.Fatalf("left dir = %q, want the right path %q", got, right)
	}
	if got := m.Panel(1).Dir(); got != right {
		t.Fatalf("right dir changed to %q", got)
	}
	if m.Active() != 1 {
		t.Fatalf("focus moved to panel %d, want it to stay on 1", m.Active())
	}
	if e, _ := m.Panel(0).Current(); e.Name != "c.txt" {
		t.Fatalf("left cursor on %q, want c.txt like the right panel", e.Name)
	}
}

func TestEqualsCanChooseTheOtherDirection(t *testing.T) {
	m, root := syncModel(t)
	right := filepath.Join(root, "right")

	m = press(t, m, "=", "l", "enter")

	if got := m.Panel(0).Dir(); got != right {
		t.Fatalf("left dir = %q, want the right path %q", got, right)
	}
	if got := m.Panel(1).Dir(); got != right {
		t.Fatalf("right dir changed to %q", got)
	}
	if m.Active() != 0 {
		t.Fatalf("focus moved to panel %d, want it to stay on 0", m.Active())
	}
}

func TestEqualsCancelLeavesPanelsAlone(t *testing.T) {
	for _, keys := range [][]string{{"esc"}, {"l", "l", "enter"}} {
		m, root := syncModel(t)
		m = press(t, m, "=")
		m = press(t, m, keys...)

		if m.SyncPending() {
			t.Fatalf("%v should close the dialog", keys)
		}
		if m.Panel(0).Dir() != root || m.Panel(1).Dir() != filepath.Join(root, "right") {
			t.Fatalf("%v changed a panel: %q, %q", keys, m.Panel(0).Dir(), m.Panel(1).Dir())
		}
	}
}

func TestEqualsWithSamePathOnlyTellsSo(t *testing.T) {
	root := tree(t, "a.txt")
	m := sized(New(root, Options{}))

	m = press(t, m, "=")

	if m.SyncPending() {
		t.Fatal("no dialog is needed when both panels show the same folder")
	}
	if !strings.Contains(m.Status(), "already") || m.StatusIsError() {
		t.Fatalf("status = %q (err=%v), want an info message", m.Status(), m.StatusIsError())
	}
}

func TestEqualsReportsAFolderThatIsGone(t *testing.T) {
	m, root := syncModel(t)
	right := filepath.Join(root, "right")

	m = press(t, m, "=", "l")
	if err := os.RemoveAll(right); err != nil {
		t.Fatal(err)
	}
	m = press(t, m, "enter")

	if !m.StatusIsError() {
		t.Fatalf("status = %q, want an error", m.Status())
	}
	if got := m.Panel(0).Dir(); got != root {
		t.Fatalf("left dir = %q, want it untouched", got)
	}
}
