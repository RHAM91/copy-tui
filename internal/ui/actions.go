package ui

import (
	"context"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/RHAM91/copy-tui/internal/fsops"
)

// selectedPaths returns the paths the next operation applies to.
func (m Model) selectedPaths() []string {
	var paths []string
	for _, e := range m.panels[m.active].Selection() {
		paths = append(paths, e.Path)
	}
	return paths
}

// otherDir is the directory of the inactive panel: the destination of copy
// and move.
func (m Model) otherDir() string { return m.panels[1-m.active].dir }

// copySelection copies the selection of the active panel into the other one.
func (m *Model) copySelection() tea.Cmd {
	paths := m.selectedPaths()
	if len(paths) == 0 {
		return nil
	}
	dstDir := m.otherDir()
	return m.startOp("Copying", func(ctx context.Context, resolve fsops.ConflictResolver, progress func(fsops.Progress)) (string, error) {
		res, err := fsops.Copy(ctx, paths, dstDir, fsops.CopyOptions{Resolve: resolve, OnProgress: progress})
		return resultSummary("Copied", res.Copied, res.Skipped, dstDir), err
	})
}

// askMove opens the confirmation to move the selection of the active panel
// into the other one, so a stray "x" never moves anything by itself.
func (m *Model) askMove() {
	paths, names := m.selectedPaths(), m.selectedNames()
	if len(paths) == 0 {
		return
	}
	dstDir := m.otherDir()
	m.confirm = &confirmPrompt{
		title:    fmt.Sprintf("Move %s to the other panel?", plural(len(paths), "item", "items")),
		items:    names,
		note:     "To " + displayPath(dstDir),
		yesLabel: "Move",
		onYes:    func(m *Model) tea.Cmd { return m.moveSelection(paths, dstDir) },
	}
}

// moveSelection moves paths into dstDir.
func (m *Model) moveSelection(paths []string, dstDir string) tea.Cmd {
	return m.startOp("Moving", func(ctx context.Context, resolve fsops.ConflictResolver, progress func(fsops.Progress)) (string, error) {
		res, err := fsops.Move(ctx, paths, dstDir, fsops.CopyOptions{Resolve: resolve, OnProgress: progress})
		return resultSummary("Moved", res.Copied, res.Skipped, dstDir), err
	})
}

func resultSummary(verb string, done, skipped int, dstDir string) string {
	s := fmt.Sprintf("%s %s to %s", verb, plural(done, "item", "items"), dstDir)
	if skipped > 0 {
		s += fmt.Sprintf(" (%d skipped)", skipped)
	}
	return s
}
