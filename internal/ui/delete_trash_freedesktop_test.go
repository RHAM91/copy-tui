//go:build unix && !darwin

package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDAsksBeforeTrashing(t *testing.T) {
	trash := isolatedTrash(t)
	root := tree(t, "a.txt")
	m := sized(New(root, Options{}))

	m = act(t, m, "d")
	if !m.ConfirmPending() {
		t.Fatal("d should ask for confirmation")
	}
	if view := m.View(); !strings.Contains(view, "trash") || !strings.Contains(view, "a.txt") {
		t.Fatalf("confirmation dialog not rendered:\n%s", view)
	}
	m = act(t, m, "n")
	if m.ConfirmPending() {
		t.Fatal("n should close the dialog")
	}
	if _, err := os.Stat(filepath.Join(root, "a.txt")); err != nil {
		t.Fatal("file removed after answering no")
	}

	m = act(t, m, "d", "y")
	if _, err := os.Stat(filepath.Join(root, "a.txt")); err == nil {
		t.Fatal("file still present after trashing")
	}
	if _, err := os.Stat(filepath.Join(trash, "files", "a.txt")); err != nil {
		t.Fatalf("file not in trash: %v", err)
	}
	if !strings.Contains(m.Status(), "trash") || len(m.Panel(0).Entries()) != 0 {
		t.Fatalf("status = %q, entries = %d", m.Status(), len(m.Panel(0).Entries()))
	}
}

func TestDEnterConfirmsTrashByDefault(t *testing.T) {
	isolatedTrash(t)
	root := tree(t, "a.txt")
	act(t, sized(New(root, Options{})), "d", "enter")
	if _, err := os.Stat(filepath.Join(root, "a.txt")); err == nil {
		t.Fatal("enter did not trash the file")
	}
}

func TestDTrashesAllMarkedEntries(t *testing.T) {
	isolatedTrash(t)
	root := tree(t, "a", "b", "c")
	m := act(t, sized(New(root, Options{})), "m", "j", "m", "d", "y")
	for name, want := range map[string]bool{"a": false, "b": true, "c": false} {
		_, err := os.Stat(filepath.Join(root, name))
		if (err == nil) != want {
			t.Fatalf("%s exists = %v, want %v", name, err == nil, want)
		}
	}
	if m.Busy() {
		t.Fatal("still busy")
	}
}
