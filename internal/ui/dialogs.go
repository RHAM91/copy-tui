package ui

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

const dialogWidth = 60

var progressBar = progress.New(
	progress.WithGradient(string(colMauve), string(colBlue)),
	progress.WithWidth(dialogWidth-4),
)

// progressView renders the modal shown while an operation runs.
func (op *operation) progressView() string {
	p := op.progress
	pct := 0.0
	if p.BytesTotal > 0 {
		pct = float64(p.BytesDone) / float64(p.BytesTotal)
	} else if p.FilesTotal > 0 {
		pct = float64(p.FilesDone) / float64(p.FilesTotal)
	}
	current := "Preparing…"
	if p.Current != "" {
		current = filepath.Base(p.Current)
	}
	hint := "esc cancel"
	if op.cancelling {
		hint = "cancelling…"
	}
	lines := []string{
		styleDialogTitle.Render(op.title + "…"),
		"",
		styleDialogText.Render(ansi.Truncate(current, dialogWidth-4, "…")),
		progressBar.ViewAs(min(pct, 1)),
		styleDialogHint.Render(fmt.Sprintf("%s / %s · %d/%d files",
			humanSize(p.BytesDone), humanSize(p.BytesTotal), p.FilesDone, p.FilesTotal)),
		"",
		styleHelpKey.Render(hint),
	}
	return styleDialog.Width(dialogWidth).Render(strings.Join(lines, "\n"))
}

// conflictView renders the "already exists" question.
func (c *conflictPrompt) view() string {
	kind := "File"
	if c.dstIsDir {
		kind = "Folder"
	}
	name := filepath.Base(c.dst)
	describe := func(isDir bool, size int64) string {
		if isDir {
			return "folder"
		}
		return humanSize(size)
	}

	var opts []string
	for i, choice := range conflictChoices {
		style := styleOption
		if i == c.choice {
			style = styleOptionActive
		}
		opts = append(opts, style.Render(choice.label))
	}
	check := "[ ]"
	if c.all {
		check = "[x]"
	}

	lines := []string{
		styleDialogTitle.Render(kind + " already exists"),
		"",
		styleDialogText.Render(ansi.Truncate(name, dialogWidth-4, "…")),
		styleDialogHint.Render(ansi.Truncate("in "+filepath.Dir(c.dst), dialogWidth-4, "…")),
		styleDialogHint.Render(fmt.Sprintf("existing: %s · new: %s",
			describe(c.dstIsDir, c.dstSize), describe(c.srcIsDir, c.srcSize))),
		"",
		lipgloss.JoinHorizontal(lipgloss.Top, opts...),
		"",
		styleDialogText.Render(check + " Apply to all"),
		"",
		hints("o", "overwrite", "s", "skip", "r", "rename", "a", "all", "esc", "cancel"),
	}
	return styleDialog.Width(dialogWidth).Render(strings.Join(lines, "\n"))
}

// hints renders "key desc · key desc" pairs.
func hints(pairs ...string) string {
	var parts []string
	for i := 0; i+1 < len(pairs); i += 2 {
		parts = append(parts, styleHelpKey.Render(pairs[i])+" "+styleHelpDesc.Render(pairs[i+1]))
	}
	return strings.Join(parts, styleHelpSep.Render(" · "))
}

// overlay draws fg centered on top of bg.
func overlay(bg, fg string, width, height int) string {
	bgLines := strings.Split(bg, "\n")
	for len(bgLines) < height {
		bgLines = append(bgLines, "")
	}
	fgLines := strings.Split(fg, "\n")
	fgW := lipgloss.Width(fg)
	x := max((width-fgW)/2, 0)
	y := max((height-len(fgLines))/2, 0)

	for i, line := range fgLines {
		row := y + i
		if row >= len(bgLines) {
			break
		}
		bgLine := bgLines[row]
		left := ansi.Truncate(bgLine, x, "")
		if w := lipgloss.Width(left); w < x {
			left += strings.Repeat(" ", x-w)
		}
		right := ansi.TruncateLeft(bgLine, x+lipgloss.Width(line), "")
		bgLines[row] = left + "\x1b[0m" + line + "\x1b[0m" + right
	}
	return strings.Join(bgLines, "\n")
}
