package ui

import (
	"context"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/RHAM91/copy-tui/internal/fsops"
)

// confirmPrompt is a yes/cancel question shown before destructive actions.
type confirmPrompt struct {
	title    string
	items    []string
	note     string
	yesLabel string
	danger   bool
	// choice is 0 for the action and 1 for cancel.
	choice int
	onYes  func(m *Model) tea.Cmd
}

// maxListedItems is how many names the confirmation lists before "and N more".
const maxListedItems = 5

// ConfirmPending reports whether a confirmation dialog is open.
func (m Model) ConfirmPending() bool { return m.confirm != nil }

func (m Model) handleConfirmKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	c := m.confirm
	run := func() (tea.Model, tea.Cmd) {
		m.confirm = nil
		return m, c.onYes(&m)
	}
	switch msg.String() {
	case "y", "Y":
		return run()
	case "n", "N", "esc", "q", "ctrl+c":
		m.confirm = nil
	case "h", "l", "left", "right", "tab", "shift+tab":
		c.choice = 1 - c.choice
	case "enter":
		if c.choice == 0 {
			return run()
		}
		m.confirm = nil
	}
	return m, nil
}

func (c *confirmPrompt) view() string {
	box, titleStyle := styleDialog, styleDialogTitle
	if c.danger {
		box, titleStyle = styleDialogDanger, styleDialogDangerTitle
	}
	width := dialogWidth - 4

	lines := []string{titleStyle.Render(c.title), ""}
	for i, item := range c.items {
		if i == maxListedItems {
			lines = append(lines, styleDialogHint.Render(fmt.Sprintf("  … and %d more", len(c.items)-maxListedItems)))
			break
		}
		lines = append(lines, styleDialogText.Render(ansi.Truncate("  "+item, width, "…")))
	}
	noteStyle := styleDialogHint
	if c.danger {
		noteStyle = styleStatusError
	}
	lines = append(lines, "", noteStyle.Render(c.note), "")

	yes, no := styleOption, styleOption
	if c.choice == 0 {
		yes = styleOptionActive
		if c.danger {
			yes = yes.Background(colRed)
		}
	} else {
		no = styleOptionActive
	}
	lines = append(lines,
		lipgloss.JoinHorizontal(lipgloss.Top, yes.Render(c.yesLabel), " ", no.Render("Cancel")),
		"",
		hints("y", "yes", "n", "no", "←/→", "choose", "enter", "confirm"),
	)
	return box.Width(dialogWidth).Render(strings.Join(lines, "\n"))
}

func (m Model) selectedNames() []string {
	var names []string
	for _, e := range m.panels[m.active].Selection() {
		names = append(names, e.Name)
	}
	return names
}

// askTrash opens the confirmation to send the selection to the trash.
func (m *Model) askTrash() {
	paths, names := m.selectedPaths(), m.selectedNames()
	if len(paths) == 0 {
		return
	}
	m.confirm = &confirmPrompt{
		title:    fmt.Sprintf("Move %s to the trash?", plural(len(paths), "item", "items")),
		items:    names,
		note:     "You can restore them from your file manager's trash.",
		yesLabel: "Trash",
		onYes: func(m *Model) tea.Cmd {
			return m.startOp("Moving to trash", func(ctx context.Context, _ fsops.ConflictResolver, progress func(fsops.Progress)) (string, error) {
				n, err := fsops.Trash(ctx, paths, progress)
				return fmt.Sprintf("Moved %s to the trash", plural(n, "item", "items")), err
			})
		},
	}
}

// askDelete opens the confirmation to delete the selection permanently.
// Cancel is the default option so Enter alone never deletes.
func (m *Model) askDelete() {
	paths, names := m.selectedPaths(), m.selectedNames()
	if len(paths) == 0 {
		return
	}
	m.confirm = &confirmPrompt{
		title:    fmt.Sprintf("Permanently delete %s?", plural(len(paths), "item", "items")),
		items:    names,
		note:     "This cannot be undone.",
		yesLabel: "Delete",
		danger:   true,
		choice:   1,
		onYes: func(m *Model) tea.Cmd {
			return m.startOp("Deleting", func(ctx context.Context, _ fsops.ConflictResolver, progress func(fsops.Progress)) (string, error) {
				n, err := fsops.Remove(ctx, paths, progress)
				return fmt.Sprintf("Deleted %s", plural(n, "item", "items")), err
			})
		},
	}
}
