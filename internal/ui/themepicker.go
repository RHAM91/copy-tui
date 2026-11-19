package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// themePicker is the ctrl+t list. Moving the cursor previews a theme; enter
// keeps and saves it, esc goes back to the theme it was opened with.
type themePicker struct {
	cursor   int
	original Theme
}

// ThemePickerOpen reports whether the ctrl+t theme list is shown.
func (m Model) ThemePickerOpen() bool { return m.themes != nil }

// loadTheme applies the theme saved in the config, or the default one with
// an error in the status bar when it cannot be read or is unknown.
func (m *Model) loadTheme() {
	applyTheme(themes[0])
	if m.opts.Config == nil {
		return
	}
	c, err := m.opts.Config.Load()
	if err != nil {
		m.setError(err)
		return
	}
	if c.Theme == "" {
		return
	}
	th, ok := ThemeByID(c.Theme)
	if !ok {
		m.setError(fmt.Errorf("unknown theme %q in %s; using %s", c.Theme, m.opts.Config.Path(), themes[0].Name))
		return
	}
	applyTheme(th)
}

func (m *Model) openThemePicker() {
	p := &themePicker{original: theme}
	for i, th := range themes {
		if th.ID == theme.ID {
			p.cursor = i
		}
	}
	m.themes = p
}

func (m Model) handleThemeKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	p := m.themes
	move := func(i int) {
		p.cursor = min(max(i, 0), len(themes)-1)
		applyTheme(themes[p.cursor])
	}
	switch msg.String() {
	case "j", "down", "tab":
		move(p.cursor + 1)
	case "k", "up", "shift+tab":
		move(p.cursor - 1)
	case "g", "home":
		move(0)
	case "G", "end":
		move(len(themes) - 1)
	case "esc", "q", "ctrl+t", "ctrl+c":
		applyTheme(p.original)
		m.themes = nil
	case "enter":
		m.themes = nil
		m.saveTheme(themes[p.cursor])
	}
	return m, nil
}

// saveTheme keeps th and writes it to the config. When the write fails the
// theme still applies until copiar exits.
func (m *Model) saveTheme(th Theme) {
	applyTheme(th)
	if m.opts.Config == nil {
		m.setInfo(fmt.Sprintf("Theme %s (not saved: no config file)", th.Name))
		return
	}
	if err := m.opts.Config.SetTheme(th.ID); err != nil {
		m.setError(fmt.Errorf("theme %s applied but not saved: %w", th.Name, err))
		return
	}
	m.setInfo(fmt.Sprintf("Theme %s saved", th.Name))
}

// swatch shows a few colors of th, drawn in that theme's own colors.
func swatch(th Theme) string {
	var b strings.Builder
	for _, c := range []lipgloss.Color{th.Accent, th.Dialog, th.Dir, th.Exec, th.Marked, th.Update, th.Error} {
		b.WriteString(lipgloss.NewStyle().Foreground(c).Render("●"))
	}
	return b.String()
}

func (p *themePicker) view() string {
	lines := []string{styleDialogTitle.Render("Theme"), ""}
	for i, th := range themes {
		name := fmt.Sprintf("  %-18s", th.Name)
		if i == p.cursor {
			name = styleOptionActive.Padding(0).Render(fmt.Sprintf("▸ %-18s", th.Name))
		} else {
			name = styleDialogText.Render(name)
		}
		lines = append(lines, name+" "+swatch(th))
	}
	lines = append(lines,
		"",
		styleDialogHint.Render("The panels preview the theme under the cursor."),
		"",
		hints("j/k", "preview", "enter", "save", "esc", "cancel"),
	)
	return styleDialog.Width(dialogWidth).Render(strings.Join(lines, "\n"))
}
