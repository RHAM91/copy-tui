package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// tree creates files (or directories when the path ends with "/") under a
// fresh temporary directory and returns it.
func tree(t *testing.T, paths ...string) string {
	t.Helper()
	root := t.TempDir()
	for _, p := range paths {
		full := filepath.Join(root, p)
		if strings.HasSuffix(p, "/") {
			if err := os.MkdirAll(full, 0o755); err != nil {
				t.Fatal(err)
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func cursorName(m Model) string {
	e, ok := m.Panel(m.Active()).Current()
	if !ok {
		return ""
	}
	return e.Name
}

func TestJKMoveCursorWithinBounds(t *testing.T) {
	m := sized(New(tree(t, "a.txt", "b.txt", "c.txt"), Options{}))

	if got := cursorName(m); got != "a.txt" {
		t.Fatalf("initial cursor on %q, want a.txt", got)
	}
	m = press(t, m, "j", "j")
	if got := cursorName(m); got != "c.txt" {
		t.Fatalf("after jj cursor on %q, want c.txt", got)
	}
	m = press(t, m, "j")
	if got := cursorName(m); got != "c.txt" {
		t.Fatalf("j past the end moved cursor to %q", got)
	}
	m = press(t, m, "k", "k", "k")
	if got := cursorName(m); got != "a.txt" {
		t.Fatalf("k past the start moved cursor to %q", got)
	}
}

func TestGGAndGJumpToEdges(t *testing.T) {
	m := sized(New(tree(t, "a", "b", "c", "d"), Options{}))

	m = press(t, m, "G")
	if got := cursorName(m); got != "d" {
		t.Fatalf("G cursor on %q, want d", got)
	}
	m = press(t, m, "g")
	if got := cursorName(m); got != "d" {
		t.Fatalf("single g moved cursor to %q", got)
	}
	m = press(t, m, "g")
	if got := cursorName(m); got != "a" {
		t.Fatalf("gg cursor on %q, want a", got)
	}
}

func TestCtrlDAndCtrlUMoveHalfPage(t *testing.T) {
	var files []string
	for i := 0; i < 60; i++ {
		files = append(files, fmt.Sprintf("f%02d", i))
	}
	m := sized(New(tree(t, files...), Options{}))
	half := m.Panel(0).listHeight() / 2
	if half < 1 {
		t.Fatalf("list height too small: %d", m.Panel(0).listHeight())
	}

	m = press(t, m, "ctrl+d")
	if got, want := cursorName(m), fmt.Sprintf("f%02d", half); got != want {
		t.Fatalf("ctrl+d cursor on %q, want %q", got, want)
	}
	m = press(t, m, "ctrl+u")
	if got := cursorName(m); got != "f00" {
		t.Fatalf("ctrl+u cursor on %q, want f00", got)
	}
}

func TestLEntersDirectoryAndHReturnsToParentOnSameEntry(t *testing.T) {
	root := tree(t, "aaa/", "sub/inner.txt", "zzz.txt")
	m := sized(New(root, Options{}))

	m = press(t, m, "j", "l")
	if got, want := m.Panel(0).Dir(), filepath.Join(root, "sub"); got != want {
		t.Fatalf("l entered %q, want %q", got, want)
	}
	if got := cursorName(m); got != "inner.txt" {
		t.Fatalf("cursor after entering on %q, want inner.txt", got)
	}

	m = press(t, m, "h")
	if got := m.Panel(0).Dir(); got != root {
		t.Fatalf("h went to %q, want %q", got, root)
	}
	if got := cursorName(m); got != "sub" {
		t.Fatalf("cursor after h on %q, want sub", got)
	}
}

func TestEnterAndBackspaceNavigateLikeLAndH(t *testing.T) {
	root := tree(t, "sub/inner.txt")
	m := sized(New(root, Options{}))

	m = press(t, m, "enter")
	if got := m.Panel(0).Dir(); got != filepath.Join(root, "sub") {
		t.Fatalf("enter went to %q", got)
	}
	m = press(t, m, "backspace")
	if got := m.Panel(0).Dir(); got != root {
		t.Fatalf("backspace went to %q", got)
	}
}

func TestLOnFileDoesNotChangeDirectory(t *testing.T) {
	root := tree(t, "file.txt")
	m := sized(New(root, Options{}))

	m = press(t, m, "l")
	if got := m.Panel(0).Dir(); got != root {
		t.Fatalf("l on a file changed dir to %q", got)
	}
}

func TestPanelsNavigateIndependently(t *testing.T) {
	root := tree(t, "sub/inner.txt")
	m := sized(New(root, Options{}))

	m = press(t, m, "l", "tab")
	if got := m.Panel(0).Dir(); got != filepath.Join(root, "sub") {
		t.Fatalf("left panel dir = %q", got)
	}
	if got := m.Panel(1).Dir(); got != root {
		t.Fatalf("right panel moved to %q, want %q", got, root)
	}
}

func TestDotTogglesHiddenFiles(t *testing.T) {
	m := sized(New(tree(t, ".hidden", "shown"), Options{}))

	if n := len(m.Panel(0).Entries()); n != 1 {
		t.Fatalf("visible entries = %d, want 1", n)
	}
	m = press(t, m, ".")
	if n := len(m.Panel(0).Entries()); n != 2 {
		t.Fatalf("entries after . = %d, want 2", n)
	}
	if n := len(m.Panel(1).Entries()); n != 2 {
		t.Fatalf("right panel entries after . = %d, want 2", n)
	}
	m = press(t, m, ".")
	if n := len(m.Panel(0).Entries()); n != 1 {
		t.Fatalf("entries after second . = %d, want 1", n)
	}
}

func TestUnreadableDirectoryShowsErrorAndStays(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root can read any directory")
	}
	root := tree(t, "locked/inner.txt")
	locked := filepath.Join(root, "locked")
	if err := os.Chmod(locked, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(locked, 0o755) })

	m := sized(New(root, Options{}))
	m = press(t, m, "l")

	if got := m.Panel(0).Dir(); got != root {
		t.Fatalf("entered unreadable dir %q", got)
	}
	if !m.StatusIsError() || !strings.Contains(m.Status(), "locked") {
		t.Fatalf("status = %q (error=%v), want error mentioning locked", m.Status(), m.StatusIsError())
	}
}

func TestViewListsEntriesWithAndWithoutIcons(t *testing.T) {
	root := tree(t, "docs/", "main.go")

	withIcons := sized(New(root, Options{Icons: true})).View()
	if !strings.Contains(withIcons, iconFor("main.go", false, false)+" main.go") {
		t.Fatalf("view with icons lacks icon for main.go:\n%s", withIcons)
	}

	plain := sized(New(root, Options{Icons: false})).View()
	if !strings.Contains(plain, "docs/") || !strings.Contains(plain, "main.go") {
		t.Fatalf("plain view lacks entries:\n%s", plain)
	}
	if strings.Contains(plain, iconFor("main.go", false, false)) {
		t.Fatalf("plain view contains icons:\n%s", plain)
	}
}
