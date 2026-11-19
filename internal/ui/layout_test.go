package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

func resized(t *testing.T, m Model, w, h int) Model {
	t.Helper()
	next, _ := m.Update(tea.WindowSizeMsg{Width: w, Height: h})
	return next.(Model)
}

// topBorders returns the row index of every line holding a panel's top
// border, and how many panel tops each of those lines holds.
func topBorders(view string) (rows []int, perRow []int) {
	for i, line := range strings.Split(ansi.Strip(view), "\n") {
		if n := strings.Count(line, "╭"); n > 0 && strings.HasPrefix(line, "╭") {
			rows = append(rows, i)
			perRow = append(perRow, n)
		}
	}
	return rows, perRow
}

func TestNarrowTerminalStacksPanels(t *testing.T) {
	root := tree(t, "a.txt", "right/")
	m := resized(t, New(root, Options{}), 99, 30)

	rows, perRow := topBorders(m.View())
	if len(rows) != 2 || perRow[0] != 1 || perRow[1] != 1 {
		t.Fatalf("want two panels one above the other, got tops at rows %v with %v boxes:\n%s", rows, perRow, m.View())
	}
	body := m.bodyHeight()
	top, bottom := m.Panel(0).height, m.Panel(1).height
	if top+bottom != body || top != body/2 {
		t.Fatalf("panel heights %d + %d, want halves of %d", top, bottom, body)
	}
	if rows[1] != top {
		t.Fatalf("bottom panel starts at row %d, want %d", rows[1], top)
	}
	for i, line := range strings.Split(ansi.Strip(m.View()), "\n") {
		if w := ansi.StringWidth(line); w > 99 {
			t.Fatalf("line %d is %d columns wide, want at most 99", i, w)
		}
	}
	if n := strings.Count(m.View(), "\n") + 1; n != 30 {
		t.Fatalf("view has %d lines, want 30", n)
	}
}

func TestWideTerminalKeepsPanelsSideBySide(t *testing.T) {
	root := tree(t, "a.txt")
	m := resized(t, New(root, Options{}), 100, 30)

	rows, perRow := topBorders(m.View())
	if len(rows) != 1 || perRow[0] != 2 {
		t.Fatalf("want both panels on one row, got tops at rows %v with %v boxes", rows, perRow)
	}
	if m.Panel(0).height != m.bodyHeight() || m.Panel(1).height != m.bodyHeight() {
		t.Fatalf("panel heights %d, %d, want %d", m.Panel(0).height, m.Panel(1).height, m.bodyHeight())
	}
}

func TestResizingAcrossTheBreakpointSwitchesLayout(t *testing.T) {
	root := tree(t, "a.txt")
	m := resized(t, New(root, Options{}), 80, 30)
	m = resized(t, m, 120, 30)

	if rows, _ := topBorders(m.View()); len(rows) != 1 {
		t.Fatalf("after widening, panel tops at rows %v, want one row", rows)
	}
	m = resized(t, m, 80, 30)
	if rows, _ := topBorders(m.View()); len(rows) != 2 {
		t.Fatalf("after narrowing, panel tops at rows %v, want two rows", rows)
	}
}
