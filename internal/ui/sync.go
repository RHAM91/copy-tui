package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// syncPrompt is the "=" dialog that opens one panel's folder in the other.
// choice indexes syncLabels: 0 copies the left path to the right panel, 1
// the right path to the left panel, 2 cancels.
type syncPrompt struct {
	choice int
}

var syncLabels = []string{"Left → Right", "Right → Left", "Cancel"}

const syncCancel = 2

// SyncPending reports whether the "=" dialog is open.
func (m Model) SyncPending() bool { return m.sync != nil }

// askSync opens the "=" dialog with the active panel's path preselected as
// the one to copy. When both panels already show the same folder it only
// says so.
func (m *Model) askSync() {
	if m.panels[0].dir == m.panels[1].dir {
		m.setInfo("Both panels already show " + displayPath(m.panels[0].dir))
		return
	}
	m.sync = &syncPrompt{choice: m.active}
}

func (m Model) handleSyncKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	p := m.sync
	switch msg.String() {
	case "esc", "q", "n", "ctrl+c":
		m.sync = nil
	case "h", "left", "shift+tab":
		p.choice = (p.choice + len(syncLabels) - 1) % len(syncLabels)
	case "l", "right", "tab":
		p.choice = (p.choice + 1) % len(syncLabels)
	case "enter":
		m.sync = nil
		if p.choice != syncCancel {
			m.syncPanels(p.choice)
		}
	}
	return m, nil
}

// syncPanels opens the folder of panel from in the other panel, with the
// cursor on the same entry. Focus stays where it was.
func (m *Model) syncPanels(from int) {
	src, dst := m.panels[from], &m.panels[1-from]
	focus := ""
	if e, ok := src.Current(); ok {
		focus = e.Name
	}
	if err := dst.cd(src.dir, focus); err != nil {
		m.setError(err)
		return
	}
	side := "Right"
	if from == 1 {
		side = "Left"
	}
	m.setInfo(fmt.Sprintf("%s panel now shows %s", side, displayPath(src.dir)))
}

func (p *syncPrompt) view(left, right string) string {
	width := dialogWidth - 4
	path := func(side, dir string) string {
		return styleDialogTitle.Render(side+":") + "\n" +
			styleDialogText.PaddingLeft(2).Render(ansi.Hardwrap(displayPath(dir), width-2, false))
	}
	var opts []string
	for i, label := range syncLabels {
		style := styleOption
		if i == p.choice {
			style = styleOptionActive
		}
		opts = append(opts, style.Render(label))
	}
	lines := []string{
		styleDialogTitle.Render("Show the same folder in both panels"),
		"",
		path("Left", left),
		path("Right", right),
		"",
		styleDialogHint.Render("Left → Right opens the left folder in the right panel."),
		"",
		lipgloss.JoinHorizontal(lipgloss.Top, opts...),
		"",
		hints("←/→", "choose", "enter", "confirm", "esc", "cancel"),
	}
	return styleDialog.Width(dialogWidth).Render(strings.Join(lines, "\n"))
}
