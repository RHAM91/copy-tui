package ui

import "github.com/charmbracelet/lipgloss"

// Theme is a named color palette. Only foregrounds, borders and highlights
// are colored: the terminal's own background is kept.
type Theme struct {
	ID   string
	Name string

	// Base is the text drawn on top of highlighted options and badges.
	Base lipgloss.Color
	// Mantle highlights the cursor row of the inactive panel.
	Mantle lipgloss.Color
	// Surface0 highlights the cursor row of the active panel.
	Surface0 lipgloss.Color
	// Surface1 draws the inactive borders and separators.
	Surface1 lipgloss.Color
	// Overlay0 draws sizes and empty-folder hints.
	Overlay0 lipgloss.Color
	// Overlay1 draws inactive paths and muted hints.
	Overlay1 lipgloss.Color
	Text     lipgloss.Color
	Subtext0 lipgloss.Color

	// Accent marks the active panel, the help keys and the mode badge.
	Accent lipgloss.Color
	// Dialog colors dialog borders, titles and the selected option.
	Dialog lipgloss.Color
	Dir    lipgloss.Color
	Link   lipgloss.Color
	Exec   lipgloss.Color
	Marked lipgloss.Color
	Update lipgloss.Color
	Error  lipgloss.Color
	Info   lipgloss.Color
}

// DefaultThemeID is the theme used when none is configured.
const DefaultThemeID = "catppuccin-mocha"

// themes lists the built-in themes; the first one is the default.
var themes = []Theme{
	// https://catppuccin.com/palette (Mocha)
	{
		ID: "catppuccin-mocha", Name: "Catppuccin Mocha",
		Base: "#1e1e2e", Mantle: "#181825", Surface0: "#313244", Surface1: "#45475a",
		Overlay0: "#6c7086", Overlay1: "#7f849c", Text: "#cdd6f4", Subtext0: "#a6adc8",
		Accent: "#cba6f7", Dialog: "#b4befe", Dir: "#89b4fa", Link: "#94e2d5",
		Exec: "#a6e3a1", Marked: "#f9e2af", Update: "#fab387", Error: "#f38ba8", Info: "#a6e3a1",
	},
	// https://github.com/rebelot/kanagawa.nvim (wave)
	{
		ID: "kanagawa-wave", Name: "Kanagawa Wave",
		Base: "#1F1F28", Mantle: "#2A2A37", Surface0: "#363646", Surface1: "#54546D",
		Overlay0: "#727169", Overlay1: "#938AA9", Text: "#DCD7BA", Subtext0: "#C8C093",
		Accent: "#957FB8", Dialog: "#7FB4CA", Dir: "#7E9CD8", Link: "#7AA89F",
		Exec: "#98BB6C", Marked: "#E6C384", Update: "#FFA066", Error: "#E46876", Info: "#98BB6C",
	},
	// https://github.com/rebelot/kanagawa.nvim (dragon)
	{
		ID: "kanagawa-dragon", Name: "Kanagawa Dragon",
		Base: "#181616", Mantle: "#282727", Surface0: "#393836", Surface1: "#625e5a",
		Overlay0: "#737c73", Overlay1: "#7a8382", Text: "#c5c9c5", Subtext0: "#a6a69c",
		Accent: "#8992a7", Dialog: "#8ba4b0", Dir: "#8ba4b0", Link: "#8ea4a2",
		Exec: "#87a987", Marked: "#c4b28a", Update: "#b6927b", Error: "#c4746e", Info: "#87a987",
	},
	// https://github.com/folke/tokyonight.nvim (night)
	{
		ID: "tokyo-night", Name: "Tokyo Night",
		Base: "#1a1b26", Mantle: "#16161e", Surface0: "#292e42", Surface1: "#414868",
		Overlay0: "#565f89", Overlay1: "#737aa2", Text: "#c0caf5", Subtext0: "#a9b1d6",
		Accent: "#bb9af7", Dialog: "#7dcfff", Dir: "#7aa2f7", Link: "#73daca",
		Exec: "#9ece6a", Marked: "#e0af68", Update: "#ff9e64", Error: "#f7768e", Info: "#9ece6a",
	},
}

// Themes returns the built-in themes, the default first.
func Themes() []Theme { return append([]Theme(nil), themes...) }

// ThemeByID returns the built-in theme with id.
func ThemeByID(id string) (Theme, bool) {
	for _, th := range themes {
		if th.ID == id {
			return th, true
		}
	}
	return Theme{}, false
}

// theme is the palette in use; applyTheme changes it.
var theme Theme

// Styles shared by the whole UI, rebuilt by applyTheme.
var (
	stylePanelActive, stylePanelInactive      lipgloss.Style
	stylePathActive, stylePathInactive        lipgloss.Style
	styleDir, styleFile, styleLink            lipgloss.Style
	styleExec, styleSize, styleMarked         lipgloss.Style
	styleEmpty, styleCursor, styleCursorI     lipgloss.Style
	styleHelpKey, styleHelpDesc, styleHelpSep lipgloss.Style

	styleStatusInfo, styleStatusError, styleStatusMode lipgloss.Style
	styleStatusMuted, styleUpdate                      lipgloss.Style

	styleDialog, styleDialogDanger           lipgloss.Style
	styleDialogTitle, styleDialogDangerTitle lipgloss.Style
	styleDialogText, styleDialogHint         lipgloss.Style
	styleOption, styleOptionActive           lipgloss.Style
)

func init() { applyTheme(themes[0]) }

// applyTheme makes th the palette in use and rebuilds every style from it.
func applyTheme(th Theme) {
	theme = th
	box := lipgloss.NewStyle().Border(lipgloss.RoundedBorder())

	stylePanelActive = box.BorderForeground(th.Accent)
	stylePanelInactive = box.BorderForeground(th.Surface1)

	stylePathActive = lipgloss.NewStyle().Foreground(th.Accent).Bold(true)
	stylePathInactive = lipgloss.NewStyle().Foreground(th.Overlay1)

	styleDir = lipgloss.NewStyle().Foreground(th.Dir).Bold(true)
	styleFile = lipgloss.NewStyle().Foreground(th.Text)
	styleLink = lipgloss.NewStyle().Foreground(th.Link).Italic(true)
	styleExec = lipgloss.NewStyle().Foreground(th.Exec)
	styleSize = lipgloss.NewStyle().Foreground(th.Overlay0)
	styleMarked = lipgloss.NewStyle().Foreground(th.Marked).Bold(true)
	styleEmpty = lipgloss.NewStyle().Foreground(th.Overlay0).Italic(true)
	styleCursor = lipgloss.NewStyle().Background(th.Surface0)
	styleCursorI = lipgloss.NewStyle().Background(th.Mantle)

	styleHelpKey = lipgloss.NewStyle().Foreground(th.Accent).Bold(true)
	styleHelpDesc = lipgloss.NewStyle().Foreground(th.Subtext0)
	styleHelpSep = lipgloss.NewStyle().Foreground(th.Surface1)

	styleStatusInfo = lipgloss.NewStyle().Foreground(th.Info)
	styleStatusError = lipgloss.NewStyle().Foreground(th.Error).Bold(true)
	styleStatusMode = lipgloss.NewStyle().Foreground(th.Base).Background(th.Accent).Bold(true).Padding(0, 1)
	styleStatusMuted = lipgloss.NewStyle().Foreground(th.Overlay1)
	styleUpdate = lipgloss.NewStyle().Foreground(th.Update).Bold(true)

	styleDialog = box.BorderForeground(th.Dialog).Padding(1, 2)
	styleDialogDanger = styleDialog.BorderForeground(th.Error)
	styleDialogTitle = lipgloss.NewStyle().Foreground(th.Dialog).Bold(true)
	styleDialogDangerTitle = lipgloss.NewStyle().Foreground(th.Error).Bold(true)
	styleDialogText = lipgloss.NewStyle().Foreground(th.Text)
	styleDialogHint = lipgloss.NewStyle().Foreground(th.Overlay1)
	styleOption = lipgloss.NewStyle().Foreground(th.Subtext0).Padding(0, 1)
	styleOptionActive = lipgloss.NewStyle().Foreground(th.Base).Background(th.Dialog).Bold(true).Padding(0, 1)

	progressBar = newProgressBar(th)
}
