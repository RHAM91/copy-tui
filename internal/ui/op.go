package ui

import (
	"context"
	"errors"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/RHAM91/copy-tui/internal/fsops"
)

// opRunner performs a long file operation. It must honour ctx, ask resolve
// about conflicts and report progress; it returns the summary shown in the
// status bar.
type opRunner func(ctx context.Context, resolve fsops.ConflictResolver, progress func(fsops.Progress)) (string, error)

// operation is a long-running file operation shown in a modal dialog.
type operation struct {
	title      string
	cancel     context.CancelFunc
	events     chan tea.Msg
	progress   fsops.Progress
	conflict   *conflictPrompt
	cancelling bool
}

// conflictPrompt is the "file already exists" question.
type conflictPrompt struct {
	src, dst string
	srcSize  int64
	dstSize  int64
	srcIsDir bool
	dstIsDir bool
	choice   int
	all      bool
	reply    chan conflictReply
}

type conflictReply struct {
	action fsops.ConflictAction
	all    bool
	cancel bool
}

// conflictChoices are the options of the conflict dialog, in display order.
var conflictChoices = []struct {
	label  string
	key    string
	action fsops.ConflictAction
}{
	{label: "Overwrite", key: "o", action: fsops.ActionOverwrite},
	{label: "Skip", key: "s", action: fsops.ActionSkip},
	{label: "Rename", key: "r", action: fsops.ActionRename},
}

type opProgressMsg struct {
	op *operation
	p  fsops.Progress
}

type opConflictMsg struct {
	op       *operation
	src, dst string
	reply    chan conflictReply
}

type opDoneMsg struct {
	op      *operation
	summary string
	err     error
}

// progressInterval limits how often progress messages reach the UI.
const progressInterval = 30 * time.Millisecond

// Busy reports whether a file operation is running.
func (m Model) Busy() bool { return m.op != nil }

// ConflictPending reports whether the conflict dialog waits for an answer.
func (m Model) ConflictPending() bool { return m.op != nil && m.op.conflict != nil }

// startOp runs fn in the background and returns the command that delivers
// its events to Update.
func (m *Model) startOp(title string, fn opRunner) tea.Cmd {
	ctx, cancel := context.WithCancel(context.Background())
	op := &operation{title: title, cancel: cancel, events: make(chan tea.Msg, 64)}
	m.op = op

	go func() {
		defer cancel()
		var applyAll *fsops.ConflictAction
		resolve := func(src, dst string) (fsops.ConflictAction, error) {
			if applyAll != nil {
				return *applyAll, nil
			}
			reply := make(chan conflictReply, 1)
			select {
			case op.events <- opConflictMsg{op: op, src: src, dst: dst, reply: reply}:
			case <-ctx.Done():
				return fsops.ActionSkip, ctx.Err()
			}
			select {
			case r := <-reply:
				if r.cancel {
					return fsops.ActionSkip, context.Canceled
				}
				if r.all {
					a := r.action
					applyAll = &a
				}
				return r.action, nil
			case <-ctx.Done():
				return fsops.ActionSkip, ctx.Err()
			}
		}
		var last time.Time
		progress := func(p fsops.Progress) {
			if now := time.Now(); now.Sub(last) >= progressInterval || p.BytesDone >= p.BytesTotal {
				last = now
				select {
				case op.events <- opProgressMsg{op: op, p: p}:
				default: // the UI is behind; a newer update will follow
				}
			}
		}
		summary, err := fn(ctx, resolve, progress)
		op.events <- opDoneMsg{op: op, summary: summary, err: err}
	}()
	return waitOp(op)
}

func waitOp(op *operation) tea.Cmd {
	return func() tea.Msg { return <-op.events }
}

// handleOpMsg applies an event of the running operation.
func (m Model) handleOpMsg(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case opProgressMsg:
		if msg.op != m.op {
			return m, nil
		}
		m.op.progress = msg.p
		return m, waitOp(m.op)
	case opConflictMsg:
		if msg.op != m.op {
			return m, nil
		}
		prompt := &conflictPrompt{src: msg.src, dst: msg.dst, reply: msg.reply}
		if info, err := os.Lstat(msg.src); err == nil {
			prompt.srcSize, prompt.srcIsDir = info.Size(), info.IsDir()
		}
		if info, err := os.Lstat(msg.dst); err == nil {
			prompt.dstSize, prompt.dstIsDir = info.Size(), info.IsDir()
		}
		m.op.conflict = prompt
		// No wait: the worker is blocked until the user answers.
		return m, nil
	case opDoneMsg:
		if msg.op != m.op {
			return m, nil
		}
		title := m.op.title
		m.op = nil
		switch {
		case errors.Is(msg.err, context.Canceled):
			m.setInfo(title + " cancelled")
		case msg.err != nil:
			m.setError(msg.err)
		default:
			m.setInfo(msg.summary)
		}
		m.activePanel().clearSelection()
		m.reloadPanels()
		return m, nil
	}
	return m, nil
}

// handleBusyKey handles keys while an operation runs: only the dialogs
// react, everything else is ignored.
func (m Model) handleBusyKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	op := m.op
	if c := op.conflict; c != nil {
		answer := func(r conflictReply) (tea.Model, tea.Cmd) {
			op.conflict = nil
			c.reply <- r
			return m, waitOp(op)
		}
		switch k := msg.String(); k {
		case "esc", "ctrl+c", "q":
			return answer(conflictReply{cancel: true})
		case "a", " ", "space":
			c.all = !c.all
		case "h", "left", "shift+tab":
			c.choice = (c.choice + len(conflictChoices) - 1) % len(conflictChoices)
		case "l", "right", "tab":
			c.choice = (c.choice + 1) % len(conflictChoices)
		case "enter":
			return answer(conflictReply{action: conflictChoices[c.choice].action, all: c.all})
		default:
			for _, choice := range conflictChoices {
				if k == choice.key {
					return answer(conflictReply{action: choice.action, all: c.all})
				}
			}
		}
		return m, nil
	}
	switch msg.String() {
	case "esc", "ctrl+c":
		if !op.cancelling {
			op.cancelling = true
			op.cancel()
		}
	}
	return m, nil
}

func (m *Model) reloadPanels() {
	for i := range m.panels {
		if err := m.panels[i].reload(); err != nil {
			m.setError(err)
		}
	}
}
