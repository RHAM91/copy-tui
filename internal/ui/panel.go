package ui

import (
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/RHAM91/copy-tui/internal/fsops"
)

// Panel is one side of the dual-pane view: a directory being browsed.
type Panel struct {
	dir        string
	entries    []fsops.Entry
	cursor     int
	offset     int
	showHidden bool
	// height is the outer height (borders included) given by the layout.
	height int
	// loadErr is set when the directory could not be listed at all.
	loadErr error
}

func newPanel(dir string, showHidden bool) Panel {
	p := Panel{dir: dir, showHidden: showHidden}
	p.loadErr = p.load()
	return p
}

// Dir returns the directory the panel is showing.
func (p Panel) Dir() string { return p.dir }

// Entries returns the listed entries in display order.
func (p Panel) Entries() []fsops.Entry { return p.entries }

// Current returns the entry under the cursor.
func (p Panel) Current() (fsops.Entry, bool) {
	if p.cursor < 0 || p.cursor >= len(p.entries) {
		return fsops.Entry{}, false
	}
	return p.entries[p.cursor], true
}

// listHeight is the number of rows available for entries: the outer height
// minus the borders, the path header and the footer.
func (p Panel) listHeight() int {
	return max(p.height-4, 1)
}

func (p *Panel) load() error {
	entries, err := fsops.ListDir(p.dir, p.showHidden)
	if err != nil {
		return err
	}
	p.entries = entries
	return nil
}

// cd switches to dir and puts the cursor on focus when it exists. On error
// the panel is left untouched.
func (p *Panel) cd(dir, focus string) error {
	entries, err := fsops.ListDir(dir, p.showHidden)
	if err != nil {
		return err
	}
	p.dir, p.entries, p.loadErr = dir, entries, nil
	p.cursor, p.offset = 0, 0
	p.focus(focus)
	return nil
}

// reload re-reads the directory keeping the cursor on the same entry name
// when possible.
func (p *Panel) reload() error {
	name := ""
	if e, ok := p.Current(); ok {
		name = e.Name
	}
	if err := p.load(); err != nil {
		return err
	}
	p.loadErr = nil
	if !p.focus(name) {
		p.setCursor(p.cursor)
	}
	return nil
}

// focus moves the cursor to the entry called name and reports whether it
// was found.
func (p *Panel) focus(name string) bool {
	if name == "" {
		return false
	}
	for i, e := range p.entries {
		if e.Name == name {
			p.setCursor(i)
			return true
		}
	}
	return false
}

func (p *Panel) setCursor(i int) {
	if len(p.entries) == 0 {
		p.cursor, p.offset = 0, 0
		return
	}
	p.cursor = min(max(i, 0), len(p.entries)-1)
	h := p.listHeight()
	if p.cursor < p.offset {
		p.offset = p.cursor
	}
	if p.cursor >= p.offset+h {
		p.offset = p.cursor - h + 1
	}
	p.offset = max(min(p.offset, len(p.entries)-h), 0)
}

func (p *Panel) move(delta int) { p.setCursor(p.cursor + delta) }

// enter opens the directory under the cursor. Files are ignored.
func (p *Panel) enter() error {
	e, ok := p.Current()
	if !ok || !e.IsDir {
		return nil
	}
	return p.cd(e.Path, "")
}

// parent goes to the parent directory, keeping the cursor on the directory
// we came from.
func (p *Panel) parent() error {
	parent := filepath.Dir(p.dir)
	if parent == p.dir {
		return nil
	}
	return p.cd(parent, filepath.Base(p.dir))
}

func (p *Panel) setHeight(h int) {
	p.height = h
	p.setCursor(p.cursor)
}

// view renders the panel inside a box of the given outer size.
func (p Panel) view(width int, active bool, opts Options) string {
	box, pathStyle := stylePanelInactive, stylePathInactive
	if active {
		box, pathStyle = stylePanelActive, stylePathActive
	}
	innerW := max(width-2, 1)
	innerH := max(p.height-2, 1)

	lines := []string{pathStyle.Render(truncateLeft(p.dir, innerW))}
	listH := p.listHeight()
	switch {
	case p.loadErr != nil:
		lines = append(lines, styleStatusError.Render(ansi.Truncate(" "+p.loadErr.Error(), innerW, "…")))
	case len(p.entries) == 0:
		lines = append(lines, styleEmpty.Render(" (empty)"))
	default:
		end := min(p.offset+listH, len(p.entries))
		for i := p.offset; i < end; i++ {
			lines = append(lines, p.row(p.entries[i], innerW, i == p.cursor, active, opts))
		}
	}
	for len(lines) < innerH-1 {
		lines = append(lines, "")
	}
	lines = append(lines, p.footer(innerW))
	return box.Width(innerW).Height(innerH).Render(strings.Join(lines, "\n"))
}

func (p Panel) row(e fsops.Entry, width int, isCursor, active bool, opts Options) string {
	nameStyle := styleFile
	switch {
	case e.IsLink:
		nameStyle = styleLink
	case e.IsDir:
		nameStyle = styleDir
	case e.Mode&0o111 != 0:
		nameStyle = styleExec
	}
	sizeStyle := styleSize

	if isCursor {
		bg := styleCursorI.GetBackground()
		if active {
			bg = styleCursor.GetBackground()
		}
		nameStyle = nameStyle.Background(bg)
		sizeStyle = sizeStyle.Background(bg)
	}

	name := e.Name
	if opts.Icons {
		name = iconFor(e.Name, e.IsDir, e.IsLink) + " " + name
	} else if e.IsDir {
		name += "/"
	}
	if e.IsLink && e.LinkTarget != "" {
		name += " → " + e.LinkTarget
	}

	size := ""
	if !e.IsDir {
		size = humanSize(e.Size)
	}
	// Layout: " " + name + padding + size + " ".
	nameW := max(width-lipgloss.Width(size)-3, 1)
	name = ansi.Truncate(name, nameW, "…")
	pad := width - 2 - lipgloss.Width(name) - lipgloss.Width(size)

	return nameStyle.Render(" "+name+strings.Repeat(" ", max(pad, 1))) + sizeStyle.Render(size+" ")
}

func (p Panel) footer(width int) string {
	dirs, files := 0, 0
	for _, e := range p.entries {
		if e.IsDir {
			dirs++
		} else {
			files++
		}
	}
	text := " " + plural(dirs, "dir", "dirs") + " · " + plural(files, "file", "files")
	if p.showHidden {
		text += " · hidden shown"
	}
	return styleStatusMuted.Render(ansi.Truncate(text, width, "…"))
}

// truncateLeft keeps the end of s (the most specific part of a path) when it
// does not fit in width cells.
func truncateLeft(s string, width int) string {
	if lipgloss.Width(s) <= width {
		return s
	}
	runes := []rune(s)
	for len(runes) > 0 && lipgloss.Width(string(runes))+1 > width {
		runes = runes[1:]
	}
	return "…" + string(runes)
}
