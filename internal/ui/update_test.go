package ui

import (
	"context"
	"errors"
	"strings"
	"testing"
)

// startWithCheck runs Init and feeds its result back like Bubble Tea does.
func startWithCheck(t *testing.T, check func(context.Context) (string, error)) Model {
	t.Helper()
	m := sized(New(tree(t, "a"), Options{CheckUpdate: check}))
	cmd := m.Init()
	if cmd == nil {
		t.Fatal("Init returned no command although CheckUpdate is set")
	}
	return drive(t, m, cmd)
}

func TestStartupShowsAvailableUpdate(t *testing.T) {
	m := startWithCheck(t, func(context.Context) (string, error) { return "0.2.0", nil })
	view := m.View()
	if !strings.Contains(view, "v0.2.0 available") || !strings.Contains(view, "copiar update") {
		t.Fatalf("update notice missing:\n%s", view)
	}
	// The notice survives other status messages.
	m = press(t, m, ".")
	if !strings.Contains(m.View(), "v0.2.0 available") {
		t.Fatal("notice disappeared after another status message")
	}
}

func TestStartupShowsNothingWhenUpToDateOrOffline(t *testing.T) {
	cases := map[string]func(context.Context) (string, error){
		"up to date": func(context.Context) (string, error) { return "", nil },
		"offline":    func(context.Context) (string, error) { return "", errors.New("no network") },
	}
	for name, check := range cases {
		t.Run(name, func(t *testing.T) {
			m := startWithCheck(t, check)
			if strings.Contains(m.View(), "available") {
				t.Fatalf("unexpected notice:\n%s", m.View())
			}
			if m.StatusIsError() {
				t.Fatalf("failed check surfaced as error: %q", m.Status())
			}
		})
	}
}

func TestNoStartupCheckWithoutChecker(t *testing.T) {
	if cmd := New(tree(t), Options{}).Init(); cmd != nil {
		t.Fatal("Init returned a command without CheckUpdate")
	}
}
