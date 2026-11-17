package ui

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/RHAM91/copy-tui/internal/fsops"
)

// inputPrompt asks for a single line of text, e.g. a new name.
type inputPrompt struct {
	title string
	input textinput.Model
	err   error
	// onSubmit applies the value; on error the prompt stays open showing it.
	onSubmit func(m *Model, value string) error
}

// InputPending reports whether a text prompt is open.
func (m Model) InputPending() bool { return m.input != nil }

// InputValue returns the text typed in the open prompt.
func (m Model) InputValue() string {
	if m.input == nil {
		return ""
	}
	return m.input.input.Value()
}

func newInputPrompt(title, value string, onSubmit func(m *Model, value string) error) *inputPrompt {
	ti := textinput.New()
	ti.Prompt = "› "
	ti.PromptStyle = styleHelpKey
	ti.TextStyle = styleDialogText
	ti.Cursor.Style = styleHelpKey
	ti.CharLimit = 255
	ti.Width = dialogWidth - 8
	ti.SetValue(value)
	ti.CursorEnd()
	ti.Focus()
	return &inputPrompt{title: title, input: ti, onSubmit: onSubmit}
}

func (m Model) handleInputKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	in := m.input
	switch msg.Type {
	case tea.KeyEsc, tea.KeyCtrlC:
		m.input = nil
		return m, nil
	case tea.KeyEnter:
		if err := in.onSubmit(&m, in.input.Value()); err != nil {
			in.err = err
			return m, nil
		}
		m.input = nil
		return m, nil
	}
	var cmd tea.Cmd
	in.input, cmd = in.input.Update(msg)
	in.err = nil
	return m, cmd
}

func (in *inputPrompt) view() string {
	lines := []string{styleDialogTitle.Render(in.title), "", in.input.View(), ""}
	if in.err != nil {
		lines = append(lines, styleStatusError.Render(in.err.Error()), "")
	}
	lines = append(lines, hints("enter", "confirm", "esc", "cancel"))
	return styleDialog.Width(dialogWidth).Render(strings.Join(lines, "\n"))
}

// askRename opens the prompt to rename the entry under the cursor.
func (m *Model) askRename() {
	e, ok := m.activePanel().Current()
	if !ok {
		return
	}
	m.input = newInputPrompt("Rename", e.Name, func(m *Model, value string) error {
		if err := fsops.Rename(e.Path, value); err != nil {
			return err
		}
		m.afterNameChange(value)
		m.setInfo(fmt.Sprintf("Renamed %s → %s", e.Name, value))
		return nil
	})
}

// askMkdir opens the prompt to create a folder in the active panel.
func (m *Model) askMkdir() {
	dir := m.activePanel().dir
	m.input = newInputPrompt("New folder in "+filepath.Base(dir), "", func(m *Model, value string) error {
		if _, err := fsops.Mkdir(dir, value); err != nil {
			return err
		}
		m.afterNameChange(value)
		m.setInfo("Created folder " + value)
		return nil
	})
}

// afterNameChange refreshes both panels and puts the active cursor on name.
func (m *Model) afterNameChange(name string) {
	m.reloadPanels()
	m.activePanel().focus(name)
}
