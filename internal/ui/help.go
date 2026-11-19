package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

type helpGroup struct {
	title string
	keys  [][2]string
}

// helpGroups is the full keymap shown with "?".
var helpGroups = []helpGroup{
	{title: "Navigation", keys: [][2]string{
		{"j / k", "down / up"},
		{"h / l", "parent / open folder"},
		{"gg / G", "first / last entry"},
		{"ctrl+d / ctrl+u", "half page down / up"},
		{"tab", "switch panel"},
		{"=", "same folder in both panels"},
		{"/", "search in panel"},
		{"n / N", "next / previous match"},
		{".", "show / hide hidden files"},
	}},
	{title: "Selection", keys: [][2]string{
		{"m", "mark / unmark and move down"},
		{"v", "visual range (v again keeps it)"},
		{"esc", "clear selection"},
	}},
	{title: "Files", keys: [][2]string{
		{"space", "copy to the other panel"},
		{"x", "move to the other panel"},
		{"d", "move to trash"},
		{"D", "delete permanently"},
		{"r", "rename"},
		{"a", "new folder"},
	}},
	{title: "Other", keys: [][2]string{
		{"ctrl+s", "save both paths as a bookmark"},
		{"ctrl+b", "bookmarks (enter open, r rename, d delete)"},
		{"?", "toggle this help"},
		{"q", "quit"},
		{"--no-icons", "start without Nerd Font icons"},
	}},
}

// HelpOpen reports whether the full help is shown.
func (m Model) HelpOpen() bool { return m.helpOpen }

func (m Model) handleHelpKey(k string) Model {
	switch k {
	case "?", "esc", "q", "enter":
		m.helpOpen = false
	}
	return m
}

func helpGroupView(g helpGroup) string {
	lines := []string{styleDialogTitle.Render(g.title)}
	for _, kv := range g.keys {
		lines = append(lines, styleHelpKey.Render(padRight(kv[0], 16))+styleHelpDesc.Render(kv[1]))
	}
	return strings.Join(lines, "\n")
}

func padRight(s string, w int) string {
	if n := lipgloss.Width(s); n < w {
		return s + strings.Repeat(" ", w-n)
	}
	return s + " "
}

// fullHelpView renders the "?" overlay in two columns.
func fullHelpView() string {
	left := helpGroupView(helpGroups[0]) + "\n\n" + helpGroupView(helpGroups[1])
	right := helpGroupView(helpGroups[2]) + "\n\n" + helpGroupView(helpGroups[3])
	body := lipgloss.JoinHorizontal(lipgloss.Top, left, "    ", right)
	return styleDialog.Render(body + "\n\n" + hints("?/esc", "close"))
}

// helpBarView renders the one-line command bar for the current mode.
func (m Model) helpBarView() string {
	var left string
	switch {
	case m.search.active:
		left = hints("type", "to search", "enter", "confirm", "esc", "cancel")
	case m.panels[m.active].visual:
		left = hints("j/k", "extend", "v", "keep selection", "esc", "clear", "space", "copy", "x", "move", "d", "trash")
	default:
		left = hints("space", "copy", "x", "move", "d", "trash", "r", "rename",
			"a", "folder", "m", "mark", "/", "search", "tab", "panel")
	}
	right := hints("?", "help", "q", "quit")
	room := m.width - lipgloss.Width(right) - 2
	left = ansi.Truncate(" "+left, max(room, 0), "…")
	gap := max(m.width-lipgloss.Width(left)-lipgloss.Width(right)-1, 1)
	return left + strings.Repeat(" ", gap) + right
}
