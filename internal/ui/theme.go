package ui

import "github.com/charmbracelet/lipgloss"

// Catppuccin Mocha palette (https://catppuccin.com/palette).
var (
	colBase     = lipgloss.Color("#1e1e2e")
	colMantle   = lipgloss.Color("#181825")
	colSurface0 = lipgloss.Color("#313244")
	colSurface1 = lipgloss.Color("#45475a")
	colOverlay0 = lipgloss.Color("#6c7086")
	colOverlay1 = lipgloss.Color("#7f849c")
	colText     = lipgloss.Color("#cdd6f4")
	colSubtext0 = lipgloss.Color("#a6adc8")
	colLavender = lipgloss.Color("#b4befe")
	colBlue     = lipgloss.Color("#89b4fa")
	colSapphire = lipgloss.Color("#74c7ec")
	colTeal     = lipgloss.Color("#94e2d5")
	colGreen    = lipgloss.Color("#a6e3a1")
	colYellow   = lipgloss.Color("#f9e2af")
	colPeach    = lipgloss.Color("#fab387")
	colRed      = lipgloss.Color("#f38ba8")
	colMauve    = lipgloss.Color("#cba6f7")
	colPink     = lipgloss.Color("#f5c2e7")
)

// Styles shared by the whole UI.
var (
	stylePanelActive = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(colMauve)
	stylePanelInactive = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(colSurface1)

	stylePathActive   = lipgloss.NewStyle().Foreground(colMauve).Bold(true)
	stylePathInactive = lipgloss.NewStyle().Foreground(colOverlay1)

	styleDir     = lipgloss.NewStyle().Foreground(colBlue).Bold(true)
	styleFile    = lipgloss.NewStyle().Foreground(colText)
	styleLink    = lipgloss.NewStyle().Foreground(colTeal).Italic(true)
	styleExec    = lipgloss.NewStyle().Foreground(colGreen)
	styleSize    = lipgloss.NewStyle().Foreground(colOverlay0)
	styleMarked  = lipgloss.NewStyle().Foreground(colYellow).Bold(true)
	styleEmpty   = lipgloss.NewStyle().Foreground(colOverlay0).Italic(true)
	styleCursor  = lipgloss.NewStyle().Background(colSurface0)
	styleCursorI = lipgloss.NewStyle().Background(colMantle)

	styleHelpKey  = lipgloss.NewStyle().Foreground(colMauve).Bold(true)
	styleHelpDesc = lipgloss.NewStyle().Foreground(colSubtext0)
	styleHelpSep  = lipgloss.NewStyle().Foreground(colSurface1)

	styleStatusInfo  = lipgloss.NewStyle().Foreground(colGreen)
	styleStatusError = lipgloss.NewStyle().Foreground(colRed).Bold(true)
	styleStatusMode  = lipgloss.NewStyle().Foreground(colBase).Background(colMauve).Bold(true).Padding(0, 1)
	styleStatusMuted = lipgloss.NewStyle().Foreground(colOverlay1)
	styleUpdate      = lipgloss.NewStyle().Foreground(colPeach).Bold(true)

	styleDialog = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colLavender).
			Padding(1, 2)
	styleDialogDanger      = styleDialog.BorderForeground(colRed)
	styleDialogTitle       = lipgloss.NewStyle().Foreground(colLavender).Bold(true)
	styleDialogDangerTitle = lipgloss.NewStyle().Foreground(colRed).Bold(true)
	styleDialogText        = lipgloss.NewStyle().Foreground(colText)
	styleDialogHint        = lipgloss.NewStyle().Foreground(colOverlay1)
	styleOption            = lipgloss.NewStyle().Foreground(colSubtext0).Padding(0, 1)
	styleOptionActive      = lipgloss.NewStyle().Foreground(colBase).Background(colLavender).Bold(true).Padding(0, 1)
)
