package ui

import (
	"context"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// updateCheckTimeout bounds the background update check at startup.
const updateCheckTimeout = 5 * time.Second

// updateAvailableMsg carries the newer version found at startup ("" when
// there is none or the check failed).
type updateAvailableMsg struct{ version string }

// checkUpdateCmd runs the injected checker in the background. Failures are
// silent: being offline must never get in the way of copying files.
func checkUpdateCmd(check func(context.Context) (string, error)) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), updateCheckTimeout)
		defer cancel()
		version, err := check(ctx)
		if err != nil {
			return updateAvailableMsg{}
		}
		return updateAvailableMsg{version: version}
	}
}

func (m Model) updateNoticeView() string {
	if m.updateVersion == "" {
		return ""
	}
	return styleUpdate.Render("v"+m.updateVersion+" available") +
		styleStatusMuted.Render(" — run ") + styleHelpKey.Render("copiar update") + " "
}
