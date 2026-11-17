package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Panel is one side of the dual-pane view: a directory being browsed.
type Panel struct {
	dir string
}

func newPanel(dir string) Panel {
	return Panel{dir: dir}
}

// Dir returns the directory the panel is showing.
func (p Panel) Dir() string { return p.dir }

// view renders the panel inside a box of the given outer size.
func (p Panel) view(width, height int, active bool, opts Options) string {
	box, pathStyle := stylePanelInactive, stylePathInactive
	if active {
		box, pathStyle = stylePanelActive, stylePathActive
	}
	innerW := max(width-2, 1)
	innerH := max(height-2, 1)

	lines := []string{pathStyle.Render(truncateLeft(p.dir, innerW))}
	for len(lines) < innerH {
		lines = append(lines, "")
	}
	return box.Width(innerW).Height(innerH).Render(strings.Join(lines, "\n"))
}

// truncateLeft keeps the end of s (the most specific part of a path) when it
// does not fit in width cells.
func truncateLeft(s string, width int) string {
	if lipgloss.Width(s) <= width {
		return s
	}
	runes := []rune(s)
	for len(runes) > 0 && lipgloss.Width(string(runes))+1 > width {
		runes = runes[1:]
	}
	return "…" + string(runes)
}
