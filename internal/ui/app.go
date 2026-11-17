// Package ui implements the dual-pane terminal interface of copiar.
package ui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Options configures the UI.
type Options struct {
	// Icons enables Nerd Font icons next to entry names.
	Icons bool
}

// Model is the root Bubble Tea model: two panels and the active one.
type Model struct {
	panels [2]Panel
	active int
	width  int
	height int
	opts   Options
}

// New creates the UI with both panels opened in startDir.
func New(startDir string, opts Options) Model {
	return Model{
		panels: [2]Panel{newPanel(startDir), newPanel(startDir)},
		opts:   opts,
	}
}

// Panel returns the panel at index i (0 = left, 1 = right).
func (m Model) Panel(i int) Panel { return m.panels[i] }

// Active returns the index of the focused panel.
func (m Model) Active() int { return m.active }

// Init implements tea.Model.
func (m Model) Init() tea.Cmd { return nil }

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil
	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "tab":
		m.active = 1 - m.active
	}
	return m, nil
}

// View implements tea.Model.
func (m Model) View() string {
	if m.width == 0 || m.height == 0 {
		return ""
	}
	bodyH := max(m.height-2, 3)
	leftW := m.width / 2
	rightW := m.width - leftW

	body := lipgloss.JoinHorizontal(lipgloss.Top,
		m.panels[0].view(leftW, bodyH, m.active == 0, m.opts),
		m.panels[1].view(rightW, bodyH, m.active == 1, m.opts),
	)
	return lipgloss.JoinVertical(lipgloss.Left, body, "", "")
}
