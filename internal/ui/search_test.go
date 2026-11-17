package ui

import (
	"strings"
	"testing"
)

func TestSlashSearchJumpsToFirstMatchCaseInsensitive(t *testing.T) {
	m := sized(New(tree(t, "alpha", "Beta", "gamma", "beta2"), Options{}))

	m = press(t, m, "/", "b", "e")
	if !m.Searching() {
		t.Fatal("expected search mode after /")
	}
	if got := cursorName(m); got != "Beta" {
		t.Fatalf("incremental search cursor on %q, want Beta", got)
	}
	m = press(t, m, "enter")
	if m.Searching() {
		t.Fatal("enter should leave search mode")
	}
	if got := cursorName(m); got != "Beta" {
		t.Fatalf("cursor after enter on %q, want Beta", got)
	}
}

func TestNAndShiftNCycleThroughMatches(t *testing.T) {
	m := sized(New(tree(t, "a-match", "b", "c-match", "d"), Options{}))

	m = press(t, m, "/", "m", "a", "t", "enter")
	if got := cursorName(m); got != "a-match" {
		t.Fatalf("first match %q, want a-match", got)
	}
	m = press(t, m, "n")
	if got := cursorName(m); got != "c-match" {
		t.Fatalf("n moved to %q, want c-match", got)
	}
	m = press(t, m, "n")
	if got := cursorName(m); got != "a-match" {
		t.Fatalf("n should wrap to a-match, got %q", got)
	}
	m = press(t, m, "N")
	if got := cursorName(m); got != "c-match" {
		t.Fatalf("N should wrap back to c-match, got %q", got)
	}
}

func TestEscCancelsSearchAndRestoresCursor(t *testing.T) {
	m := sized(New(tree(t, "one", "two", "three"), Options{}))

	// Sorted: one, three, two. Start on "three", search jumps to "two".
	m = press(t, m, "j", "/", "t", "w")
	if got := cursorName(m); got != "two" {
		t.Fatalf("search cursor on %q, want two", got)
	}
	m = press(t, m, "esc")
	if m.Searching() {
		t.Fatal("esc should leave search mode")
	}
	if got := cursorName(m); got != "three" {
		t.Fatalf("cursor after esc on %q, want three", got)
	}
}

func TestSearchWithoutMatchReportsError(t *testing.T) {
	m := sized(New(tree(t, "one"), Options{}))

	m = press(t, m, "/", "z", "z", "enter")
	if !m.StatusIsError() || !strings.Contains(m.Status(), "zz") {
		t.Fatalf("status = %q (error=%v), want not-found error for zz", m.Status(), m.StatusIsError())
	}
}

func TestKeysAreTypedWhileSearching(t *testing.T) {
	m := sized(New(tree(t, "quiz", "other"), Options{}))

	next, cmd := m.Update(key("/"))
	m = next.(Model)
	next, cmd = m.Update(key("q"))
	m = next.(Model)
	if cmd != nil {
		t.Fatal("q while searching should not quit")
	}
	if got := m.SearchQuery(); got != "q" {
		t.Fatalf("query = %q, want q", got)
	}
	m = press(t, m, "backspace")
	if got := m.SearchQuery(); got != "" {
		t.Fatalf("query after backspace = %q, want empty", got)
	}
}
