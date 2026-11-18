package ui

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/RHAM91/copy-tui/internal/bookmarks"
)

var errBookmarksUnavailable = errors.New("bookmarks are unavailable: no config directory")

// bookmarkList is the ctrl+b table. broken, when set, is the dialog shown
// for a bookmark whose folders are gone.
type bookmarkList struct {
	items  []bookmarks.Bookmark
	cursor int
	offset int
	broken *brokenPrompt
}

// brokenPrompt warns that one or both folders of a bookmark are missing.
type brokenPrompt struct {
	bm      bookmarks.Bookmark
	leftOK  bool
	rightOK bool
	choice  int
	choices []brokenChoice
}

type brokenChoice int

const (
	brokenGo brokenChoice = iota
	brokenDelete
	brokenCancel
)

var brokenLabels = map[brokenChoice]string{
	brokenGo:     "Go anyway",
	brokenDelete: "Delete bookmark",
	brokenCancel: "Cancel",
}

// BookmarksOpen reports whether the bookmark table is shown.
func (m Model) BookmarksOpen() bool { return m.bookmarks != nil }

// askSaveBookmark asks for a name and saves both panel paths under it.
func (m *Model) askSaveBookmark() {
	store := m.opts.Bookmarks
	if store == nil {
		m.setError(errBookmarksUnavailable)
		return
	}
	b := bookmarks.Bookmark{Left: m.panels[0].dir, Right: m.panels[1].dir}
	m.input = newInputPrompt("Save bookmark for both panels", "", func(m *Model, value string) error {
		name, err := bookmarks.CleanName(value)
		if err != nil {
			return err
		}
		b.Name = name
		old, exists, err := store.Get(name)
		if err != nil {
			return err
		}
		if !exists {
			m.putBookmark(store, b)
			return nil
		}
		m.confirm = &confirmPrompt{
			title:    fmt.Sprintf("Replace bookmark %q?", old.Name),
			items:    []string{"left:  " + displayPath(old.Left), "right: " + displayPath(old.Right)},
			note:     "It will point to the current panel paths instead.",
			yesLabel: "Replace",
			onYes: func(m *Model) tea.Cmd {
				m.putBookmark(store, b)
				return nil
			},
		}
		return nil
	})
}

func (m *Model) putBookmark(store *bookmarks.Store, b bookmarks.Bookmark) {
	if err := store.Put(b); err != nil {
		m.setError(err)
		return
	}
	m.setInfo(fmt.Sprintf("Saved bookmark %q", b.Name))
}

// openBookmarks shows the bookmark table.
func (m *Model) openBookmarks() {
	if m.opts.Bookmarks == nil {
		m.setError(errBookmarksUnavailable)
		return
	}
	items, err := m.opts.Bookmarks.List()
	if err != nil {
		m.setError(err)
		return
	}
	m.bookmarks = &bookmarkList{items: items}
}

// reloadBookmarks re-reads the table after a change, keeping the cursor on
// focus when given, else on the same row.
func (m *Model) reloadBookmarks(focus string) {
	l := m.bookmarks
	if l == nil {
		return
	}
	items, err := m.opts.Bookmarks.List()
	if err != nil {
		m.setError(err)
		return
	}
	l.items = items
	for i, b := range items {
		if strings.EqualFold(b.Name, focus) {
			l.cursor = i
		}
	}
	l.cursor = min(max(l.cursor, 0), max(len(items)-1, 0))
}

func (l *bookmarkList) current() (bookmarks.Bookmark, bool) {
	if l.cursor < 0 || l.cursor >= len(l.items) {
		return bookmarks.Bookmark{}, false
	}
	return l.items[l.cursor], true
}

func (m Model) handleBookmarksKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	l := m.bookmarks
	if l.broken != nil {
		return m.handleBrokenKey(msg.String())
	}
	switch msg.String() {
	case "esc", "q", "ctrl+b", "ctrl+c":
		m.bookmarks = nil
	case "j", "down":
		l.cursor = min(l.cursor+1, max(len(l.items)-1, 0))
	case "k", "up":
		l.cursor = max(l.cursor-1, 0)
	case "g", "home":
		l.cursor = 0
	case "G", "end":
		l.cursor = max(len(l.items)-1, 0)
	case "enter", "l":
		if b, ok := l.current(); ok {
			m.openBookmark(b)
		}
	case "d":
		if b, ok := l.current(); ok {
			m.askDeleteBookmark(b)
		}
	case "r":
		if b, ok := l.current(); ok {
			m.askRenameBookmark(b)
		}
	}
	return m, nil
}

// openBookmark checks that both folders still exist before going there.
func (m *Model) openBookmark(b bookmarks.Bookmark) {
	leftOK, rightOK := isDir(b.Left), isDir(b.Right)
	if leftOK && rightOK {
		m.goToBookmark(b, true, true)
		return
	}
	p := &brokenPrompt{bm: b, leftOK: leftOK, rightOK: rightOK}
	if leftOK || rightOK {
		p.choices = []brokenChoice{brokenGo, brokenDelete, brokenCancel}
	} else {
		// Nothing to open: Cancel is the default so enter alone never deletes.
		p.choices = []brokenChoice{brokenDelete, brokenCancel}
		p.choice = 1
	}
	m.bookmarks.broken = p
}

func isDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// goToBookmark opens the bookmark's folders in the panels selected by left
// and right, then closes the table.
func (m *Model) goToBookmark(b bookmarks.Bookmark, left, right bool) {
	m.bookmarks = nil
	var errs []error
	if left {
		errs = append(errs, m.panels[0].cd(b.Left, ""))
	}
	if right {
		errs = append(errs, m.panels[1].cd(b.Right, ""))
	}
	if err := errors.Join(errs...); err != nil {
		m.setError(err)
		return
	}
	switch {
	case !left:
		m.setInfo(fmt.Sprintf("Opened bookmark %q; left folder not found: %s", b.Name, displayPath(b.Left)))
	case !right:
		m.setInfo(fmt.Sprintf("Opened bookmark %q; right folder not found: %s", b.Name, displayPath(b.Right)))
	default:
		m.setInfo(fmt.Sprintf("Opened bookmark %q", b.Name))
	}
}

func (m Model) handleBrokenKey(k string) (tea.Model, tea.Cmd) {
	p := m.bookmarks.broken
	apply := func(c brokenChoice) {
		switch c {
		case brokenGo:
			m.goToBookmark(p.bm, p.leftOK, p.rightOK)
		case brokenDelete:
			m.bookmarks.broken = nil
			m.deleteBookmark(p.bm.Name, "Deleted broken bookmark %q")
		case brokenCancel:
			m.bookmarks.broken = nil
		}
	}
	switch k {
	case "esc", "q", "n", "ctrl+c":
		apply(brokenCancel)
	case "d":
		apply(brokenDelete)
	case "g":
		if p.leftOK || p.rightOK {
			apply(brokenGo)
		}
	case "h", "left", "shift+tab":
		p.choice = (p.choice + len(p.choices) - 1) % len(p.choices)
	case "l", "right", "tab":
		p.choice = (p.choice + 1) % len(p.choices)
	case "enter":
		apply(p.choices[p.choice])
	}
	return m, nil
}

func (m *Model) deleteBookmark(name, done string) {
	if err := m.opts.Bookmarks.Delete(name); err != nil {
		m.setError(err)
		return
	}
	m.reloadBookmarks("")
	m.setInfo(fmt.Sprintf(done, name))
}

// askDeleteBookmark confirms before removing a bookmark from the table.
func (m *Model) askDeleteBookmark(b bookmarks.Bookmark) {
	m.confirm = &confirmPrompt{
		title:    fmt.Sprintf("Delete bookmark %q?", b.Name),
		items:    []string{"left:  " + displayPath(b.Left), "right: " + displayPath(b.Right)},
		note:     "Only the bookmark is removed; the folders are not touched.",
		yesLabel: "Delete",
		danger:   true,
		choice:   1,
		onYes: func(m *Model) tea.Cmd {
			m.deleteBookmark(b.Name, "Deleted bookmark %q")
			return nil
		},
	}
}

// askRenameBookmark asks for a new name, then confirms the change.
func (m *Model) askRenameBookmark(b bookmarks.Bookmark) {
	store := m.opts.Bookmarks
	m.input = newInputPrompt("Rename bookmark", b.Name, func(m *Model, value string) error {
		name, err := bookmarks.CleanName(value)
		if err != nil {
			return err
		}
		if name == b.Name {
			return nil
		}
		if other, exists, err := store.Get(name); err != nil {
			return err
		} else if exists && !strings.EqualFold(other.Name, b.Name) {
			return bookmarks.ErrExists
		}
		m.confirm = &confirmPrompt{
			title:    "Rename bookmark?",
			items:    []string{b.Name + " → " + name},
			note:     "The saved paths stay the same.",
			yesLabel: "Rename",
			onYes: func(m *Model) tea.Cmd {
				if err := store.Rename(b.Name, name); err != nil {
					m.setError(err)
					return nil
				}
				m.reloadBookmarks(name)
				m.setInfo(fmt.Sprintf("Renamed bookmark %q → %q", b.Name, name))
				return nil
			},
		}
		return nil
	})
}

// displayPath shortens the home directory to "~".
func displayPath(path string) string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return path
	}
	if path == home {
		return "~"
	}
	if rel, ok := strings.CutPrefix(path, home+string(filepath.Separator)); ok {
		return "~" + string(filepath.Separator) + rel
	}
	return path
}

// view renders the table, sized to the window.
func (l *bookmarkList) view(width, height int) string {
	if l.broken != nil {
		return l.broken.view()
	}
	boxW := min(max(width-4, dialogWidth), 120)
	inner := boxW - 4

	lines := []string{styleDialogTitle.Render("Bookmarks"), ""}
	if len(l.items) == 0 {
		lines = append(lines,
			styleEmpty.Render("No bookmarks yet."),
			styleDialogHint.Render("Press ctrl+s in the panels to save both paths."),
			"", hints("esc", "close"))
		return styleDialog.Width(boxW).Render(strings.Join(lines, "\n"))
	}

	nameW := 4
	for _, b := range l.items {
		nameW = max(nameW, lipgloss.Width(b.Name))
	}
	nameW = min(nameW, inner/4)
	pathW := max((inner-nameW-4)/2, 8)
	row := func(name, left, right string) string {
		return padRight(ansi.Truncate(name, nameW, "…"), nameW+2) +
			padRight(ansi.Truncate(left, pathW, "…"), pathW+2) +
			ansi.Truncate(right, pathW, "…")
	}

	rows := max(height-12, 3)
	if l.cursor < l.offset {
		l.offset = l.cursor
	}
	if l.cursor >= l.offset+rows {
		l.offset = l.cursor - rows + 1
	}

	lines = append(lines, styleDialogHint.Render(row("Name", "Left", "Right")))
	for i := l.offset; i < min(l.offset+rows, len(l.items)); i++ {
		b := l.items[i]
		text := row(b.Name, displayPath(b.Left), displayPath(b.Right))
		// Pad to exactly inner (padRight always adds a space) so the
		// highlighted row spans the box without wrapping.
		text += strings.Repeat(" ", max(inner-lipgloss.Width(text), 0))
		if i == l.cursor {
			lines = append(lines, styleCursor.Foreground(colText).Bold(true).Render(text))
		} else {
			lines = append(lines, styleDialogText.Render(text))
		}
	}
	if len(l.items) > rows {
		lines = append(lines, styleDialogHint.Render(fmt.Sprintf("%d/%d", l.cursor+1, len(l.items))))
	}
	lines = append(lines, "", hints("enter", "open", "j/k", "move", "r", "rename", "d", "delete", "esc", "close"))
	return styleDialog.Width(boxW).Render(strings.Join(lines, "\n"))
}

func (p *brokenPrompt) view() string {
	width := dialogWidth - 4
	// The full path is shown, wrapped, so the user can tell what is missing.
	status := func(side, path string, ok bool) string {
		head := styleStatusError.Render("✗ " + side + " folder not found:")
		if ok {
			head = styleStatusInfo.Render("✓ " + side + " folder exists:")
		}
		return head + "\n" + styleDialogText.PaddingLeft(2).Render(ansi.Hardwrap(displayPath(path), width-2, false))
	}
	note := "Both folders are gone. Delete this bookmark?"
	if p.leftOK || p.rightOK {
		note = "Go anyway opens only the folder that still exists."
	}

	var opts []string
	for i, c := range p.choices {
		style := styleOption
		if i == p.choice {
			style = styleOptionActive
			if c == brokenDelete {
				style = style.Background(colRed)
			}
		}
		opts = append(opts, style.Render(brokenLabels[c]))
	}
	keys := []string{"d", "delete", "esc", "cancel", "enter", "confirm"}
	if p.leftOK || p.rightOK {
		keys = append([]string{"g", "go anyway"}, keys...)
	}
	lines := []string{
		styleDialogDangerTitle.Render(fmt.Sprintf("Bookmark %q is broken", p.bm.Name)),
		"",
		status("Left", p.bm.Left, p.leftOK),
		status("Right", p.bm.Right, p.rightOK),
		"",
		styleDialogHint.Render(note),
		"",
		lipgloss.JoinHorizontal(lipgloss.Top, opts...),
		"",
		hints(keys...),
	}
	return styleDialogDanger.Width(dialogWidth).Render(strings.Join(lines, "\n"))
}
