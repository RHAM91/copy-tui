package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// search holds the state of the "/" prompt.
type search struct {
	active bool
	query  string
	// origin is the cursor position before the search started, restored on Esc.
	origin int
	// last is the last confirmed query, reused by n and N.
	last string
}

// Searching reports whether the "/" prompt is open.
func (m Model) Searching() bool { return m.search.active }

// SearchQuery returns the text typed in the "/" prompt.
func (m Model) SearchQuery() string { return m.search.query }

func (m *Model) startSearch() {
	m.search.active = true
	m.search.query = ""
	m.search.origin = m.activePanel().cursor
}

func (m Model) handleSearchKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	p := m.activePanel()
	switch msg.Type {
	case tea.KeyEsc, tea.KeyCtrlC:
		m.search.active = false
		p.setCursor(m.search.origin)
		return m, nil
	case tea.KeyEnter:
		m.search.active = false
		if m.search.query == "" {
			return m, nil
		}
		m.search.last = m.search.query
		if p.findFrom(m.search.query, m.search.origin, 1) < 0 {
			m.setError(fmt.Errorf("pattern not found: %s", m.search.query))
		}
		return m, nil
	case tea.KeyBackspace:
		if r := []rune(m.search.query); len(r) > 0 {
			m.search.query = string(r[:len(r)-1])
		}
	case tea.KeyRunes, tea.KeySpace:
		m.search.query += string(msg.Runes)
	default:
		return m, nil
	}
	m.status, m.statusErr = "", false
	if m.search.query == "" {
		p.setCursor(m.search.origin)
		return m, nil
	}
	if i := p.findFrom(m.search.query, m.search.origin, 1); i >= 0 {
		p.setCursor(i)
	}
	return m, nil
}

// repeatSearch jumps to the next (dir = 1) or previous (dir = -1) match of
// the last query.
func (m *Model) repeatSearch(dir int) {
	if m.search.last == "" {
		return
	}
	p := m.activePanel()
	i := p.findFrom(m.search.last, p.cursor+dir, dir)
	if i < 0 {
		m.setError(fmt.Errorf("pattern not found: %s", m.search.last))
		return
	}
	p.setCursor(i)
}

// findFrom returns the index of the first entry whose name contains query
// (case-insensitive), scanning from start in direction dir and wrapping
// around. It returns -1 when nothing matches.
func (p Panel) findFrom(query string, start, dir int) int {
	n := len(p.entries)
	if n == 0 {
		return -1
	}
	q := strings.ToLower(query)
	for k := 0; k < n; k++ {
		i := ((start+dir*k)%n + n) % n
		if strings.Contains(strings.ToLower(p.entries[i].Name), q) {
			return i
		}
	}
	return -1
}
