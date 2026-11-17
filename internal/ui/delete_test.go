package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func isolatedTrash(t *testing.T) string {
	t.Helper()
	data := t.TempDir()
	t.Setenv("XDG_DATA_HOME", data)
	return filepath.Join(data, "Trash")
}

func TestShiftDDeletesPermanentlyAfterExplicitConfirmation(t *testing.T) {
	trash := isolatedTrash(t)
	root := tree(t, "a.txt", "dir/inner")
	m := sized(New(root, Options{}))

	m = act(t, m, "D")
	if !m.ConfirmPending() {
		t.Fatal("D should ask for confirmation")
	}
	if view := m.View(); !strings.Contains(view, "cannot be undone") {
		t.Fatalf("danger dialog not rendered:\n%s", view)
	}
	m = act(t, m, "enter")
	if _, err := os.Stat(filepath.Join(root, "dir")); err != nil {
		t.Fatal("enter on the default option must not delete")
	}

	m = act(t, m, "D", "y")
	if _, err := os.Stat(filepath.Join(root, "dir")); err == nil {
		t.Fatal("directory still present")
	}
	if _, err := os.Stat(filepath.Join(trash, "files", "dir")); err == nil {
		t.Fatal("permanent delete went to the trash")
	}
	if !strings.Contains(m.Status(), "Deleted 1 item") {
		t.Fatalf("status = %q", m.Status())
	}
}

func TestEscClosesConfirmation(t *testing.T) {
	root := tree(t, "a.txt")
	m := act(t, sized(New(root, Options{})), "D", "esc")
	if m.ConfirmPending() || m.Busy() {
		t.Fatal("esc should close the dialog without running")
	}
	if _, err := os.Stat(filepath.Join(root, "a.txt")); err != nil {
		t.Fatal("file deleted after esc")
	}
}

func TestOtherKeysAreIgnoredWhileConfirming(t *testing.T) {
	root := tree(t, "a", "b")
	m := act(t, sized(New(root, Options{})), "d", "j", "G")
	if !m.ConfirmPending() {
		t.Fatal("dialog closed by an unrelated key")
	}
	if got := cursorName(m); got != "a" {
		t.Fatalf("cursor moved to %q behind the dialog", got)
	}
}
