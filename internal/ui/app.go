// Package ui implements the dual-pane terminal interface of copiar.
package ui

import (
	"context"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/RHAM91/copy-tui/internal/bookmarks"
)

// Options configures the UI.
type Options struct {
	// Icons enables Nerd Font icons next to entry names.
	Icons bool
	// CheckUpdate, when set, runs in the background at startup and returns
	// a newer version ("" when up to date) to show in the status bar.
	CheckUpdate func(context.Context) (string, error)
	// Bookmarks stores the ctrl+s / ctrl+b bookmarks; nil disables them.
	Bookmarks *bookmarks.Store
}

// Model is the root Bubble Tea model: two panels and the active one.
type Model struct {
	panels     [2]Panel
	active     int
	width      int
	height     int
	opts       Options
	showHidden bool

	// pendingG is set after a first "g" while waiting for "gg".
	pendingG bool
	search   search

	// op is the running file operation, nil when idle.
	op *operation
	// confirm is the open confirmation dialog, nil when none.
	confirm *confirmPrompt
	// input is the open text prompt (rename, new folder), nil when none.
	input *inputPrompt
	// sync is the open "=" dialog, nil when closed.
	sync *syncPrompt
	// bookmarks is the open ctrl+b table, nil when closed.
	bookmarks *bookmarkList
	// helpOpen is true while the full "?" help is shown.
	helpOpen bool
	// updateVersion is a newer release found at startup, "" when none.
	updateVersion string

	status    string
	statusErr bool
}

// New creates the UI with both panels opened in startDir.
func New(startDir string, opts Options) Model {
	return Model{
		panels: [2]Panel{newPanel(startDir, false), newPanel(startDir, false)},
		opts:   opts,
	}
}

// Panel returns the panel at index i (0 = left, 1 = right).
func (m Model) Panel(i int) Panel { return m.panels[i] }

// Active returns the index of the focused panel.
func (m Model) Active() int { return m.active }

// Status returns the message shown in the status bar.
func (m Model) Status() string { return m.status }

// StatusIsError reports whether the status message is an error.
func (m Model) StatusIsError() bool { return m.statusErr }

// modalKind is the layer on top of the panels that owns the keyboard.
type modalKind int

const (
	modalNone modalKind = iota
	// modalConflict is the "file already exists" dialog of a running operation.
	modalConflict
	// modalOp is the progress dialog of a running operation.
	modalOp
	modalConfirm
	modalInput
	// modalSync is the "=" dialog that shows one folder in both panels.
	modalSync
	// modalBookmarks is the ctrl+b table and its broken-bookmark dialog. It
	// sits under confirm and input, which it opens for delete and rename.
	modalBookmarks
	modalHelp
	// modalSearch is the "/" prompt. It has no overlay: it takes over the
	// status bar.
	modalSearch
)

// activeModal returns the topmost layer. This order is the only place that
// decides which layer receives keys and which overlay is drawn.
func (m Model) activeModal() modalKind {
	switch {
	case m.op != nil && m.op.conflict != nil:
		return modalConflict
	case m.op != nil:
		return modalOp
	case m.confirm != nil:
		return modalConfirm
	case m.input != nil:
		return modalInput
	case m.sync != nil:
		return modalSync
	case m.bookmarks != nil:
		return modalBookmarks
	case m.helpOpen:
		return modalHelp
	case m.search.active:
		return modalSearch
	}
	return modalNone
}

func (m *Model) setInfo(msg string) { m.status, m.statusErr = msg, false }
func (m *Model) setError(err error) { m.status, m.statusErr = err.Error(), true }

func (m *Model) activePanel() *Panel { return &m.panels[m.active] }

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	if m.opts.CheckUpdate == nil {
		return nil
	}
	return checkUpdateCmd(m.opts.CheckUpdate)
}

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		for i, size := range m.panelSizes() {
			m.panels[i].setHeight(size.h)
		}
		return m, nil
	case tea.KeyMsg:
		switch m.activeModal() {
		case modalConflict, modalOp:
			return m.handleBusyKey(msg)
		case modalConfirm:
			return m.handleConfirmKey(msg)
		case modalInput:
			return m.handleInputKey(msg)
		case modalSync:
			return m.handleSyncKey(msg)
		case modalBookmarks:
			return m.handleBookmarksKey(msg)
		case modalHelp:
			return m.handleHelpKey(msg.String()), nil
		case modalSearch:
			return m.handleSearchKey(msg)
		}
		return m.handleKey(msg)
	case opProgressMsg, opConflictMsg, opDoneMsg:
		return m.handleOpMsg(msg)
	case updateAvailableMsg:
		m.updateVersion = msg.version
		return m, nil
	}
	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	k := msg.String()

	if m.pendingG {
		m.pendingG = false
		if k == "g" {
			m.activePanel().setCursor(0)
			return m, nil
		}
	}

	p := m.activePanel()
	switch k {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "tab":
		m.active = 1 - m.active
	case "j", "down":
		p.move(1)
	case "k", "up":
		p.move(-1)
	case "g":
		m.pendingG = true
	case "G", "end":
		p.setCursor(len(p.entries) - 1)
	case "home":
		p.setCursor(0)
	case "ctrl+d", "pgdown":
		p.move(p.listHeight() / 2)
	case "ctrl+u", "pgup":
		p.move(-p.listHeight() / 2)
	case "l", "right", "enter":
		if err := p.enter(); err != nil {
			m.setError(err)
		}
	case "h", "left", "backspace":
		if err := p.parent(); err != nil {
			m.setError(err)
		}
	case ".":
		m.toggleHidden()
	case "/":
		m.startSearch()
	case "n":
		m.repeatSearch(1)
	case "N":
		m.repeatSearch(-1)
	case "m":
		p.toggleMark()
	case "v":
		p.toggleVisual()
	case "esc":
		p.clearSelection()
		m.setInfo("")
	case " ":
		return m, m.copySelection()
	case "x":
		m.askMove()
	case "d":
		m.askTrash()
	case "D":
		m.askDelete()
	case "?":
		m.helpOpen = true
	case "r":
		m.askRename()
	case "a":
		m.askMkdir()
	case "ctrl+s":
		m.askSaveBookmark()
	case "ctrl+b":
		m.openBookmarks()
	case "=":
		m.askSync()
	}
	return m, nil
}

func (m *Model) toggleHidden() {
	m.showHidden = !m.showHidden
	for i := range m.panels {
		m.panels[i].showHidden = m.showHidden
		if err := m.panels[i].reload(); err != nil {
			m.setError(err)
		}
	}
	if m.showHidden {
		m.setInfo("Showing hidden files")
	} else {
		m.setInfo("Hiding hidden files")
	}
}

// bodyHeight is the height of the panels: the window minus the status and
// help bars.
func (m Model) bodyHeight() int { return max(m.height-2, 3) }

// stackedBelow is the terminal width under which the panels are shown one
// above the other instead of side by side.
const stackedBelow = 100

// stacked reports whether the panels are laid out one above the other.
func (m Model) stacked() bool { return m.width < stackedBelow }

type panelSize struct{ w, h int }

// panelSizes returns the outer size of the left and right panels. Side by
// side they split the width; stacked, the left panel takes the top half.
func (m Model) panelSizes() [2]panelSize {
	body := m.bodyHeight()
	if m.stacked() {
		top := body / 2
		return [2]panelSize{{m.width, top}, {m.width, body - top}}
	}
	left := m.width / 2
	return [2]panelSize{{left, body}, {m.width - left, body}}
}

// View implements tea.Model.
func (m Model) View() string {
	if m.width == 0 || m.height == 0 {
		return ""
	}
	sizes := m.panelSizes()
	left := m.panels[0].view(sizes[0].w, m.active == 0, m.opts)
	right := m.panels[1].view(sizes[1].w, m.active == 1, m.opts)
	body := lipgloss.JoinHorizontal(lipgloss.Top, left, right)
	if m.stacked() {
		body = lipgloss.JoinVertical(lipgloss.Left, left, right)
	}
	view := lipgloss.JoinVertical(lipgloss.Left, body, m.statusView(), m.helpBarView())
	switch m.activeModal() {
	case modalConflict:
		view = overlay(view, m.op.conflict.view(), m.width, m.height)
	case modalOp:
		view = overlay(view, m.op.progressView(), m.width, m.height)
	case modalConfirm:
		view = overlay(view, m.confirm.view(), m.width, m.height)
	case modalInput:
		view = overlay(view, m.input.view(), m.width, m.height)
	case modalSync:
		view = overlay(view, m.sync.view(m.panels[0].dir, m.panels[1].dir), m.width, m.height)
	case modalBookmarks:
		view = overlay(view, m.bookmarks.view(m.width, m.height), m.width, m.height)
	case modalHelp:
		view = overlay(view, fullHelpView(), m.width, m.height)
	}
	return view
}

func (m Model) statusView() string {
	if m.search.active {
		return styleStatusMode.Render("SEARCH") + " " +
			styleDialogText.Render("/"+m.search.query) + styleHelpKey.Render("▏")
	}
	prefix := ""
	if m.panels[m.active].visual {
		prefix = styleStatusMode.Render("VISUAL") + " "
	}
	style := styleStatusInfo
	if m.statusErr {
		style = styleStatusError
	}
	notice := m.updateNoticeView()
	if notice == "" {
		return ansi.Truncate(prefix+style.Render(" "+m.status), m.width, "…")
	}
	room := max(m.width-lipgloss.Width(notice)-1, 0)
	left := ansi.Truncate(prefix+style.Render(" "+m.status), room, "…")
	gap := max(m.width-lipgloss.Width(left)-lipgloss.Width(notice), 1)
	return left + strings.Repeat(" ", gap) + notice
}
