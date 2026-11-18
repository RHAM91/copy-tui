package ui

import (
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/muesli/termenv"

	"github.com/RHAM91/copy-tui/internal/bookmarks"
)

// bookmarkModel opens the UI in root with an empty bookmark store and puts
// the right panel in root/right when that folder exists.
func bookmarkModel(t *testing.T, root string) (Model, *bookmarks.Store) {
	t.Helper()
	store := bookmarks.NewStore(filepath.Join(t.TempDir(), "bookmarks.json"))
	m := sized(New(root, Options{Bookmarks: store}))
	if _, err := os.Stat(filepath.Join(root, "right")); err == nil {
		if err := m.panels[1].cd(filepath.Join(root, "right"), ""); err != nil {
			t.Fatal(err)
		}
	}
	return m, store
}

func bookmarkNames(t *testing.T, s *bookmarks.Store) []string {
	t.Helper()
	list, err := s.List()
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, b := range list {
		out = append(out, b.Name)
	}
	return out
}

func mustPut(t *testing.T, s *bookmarks.Store, b bookmarks.Bookmark) {
	t.Helper()
	if err := s.Put(b); err != nil {
		t.Fatal(err)
	}
}

func TestCtrlSSavesBothPanelPaths(t *testing.T) {
	root := tree(t, "right/")
	m, store := bookmarkModel(t, root)

	m = press(t, m, "ctrl+s")
	if !m.InputPending() {
		t.Fatal("ctrl+s should ask for the bookmark name")
	}
	m = typeText(t, m, "work")
	m = press(t, m, "enter")

	if m.InputPending() || m.ConfirmPending() {
		t.Fatal("dialog still open after saving")
	}
	got, ok, err := store.Get("work")
	want := bookmarks.Bookmark{Name: "work", Left: root, Right: filepath.Join(root, "right")}
	if err != nil || !ok || got != want {
		t.Fatalf("stored %v, %v, %v; want %v", got, ok, err, want)
	}
	if !strings.Contains(m.Status(), "work") || m.StatusIsError() {
		t.Fatalf("status = %q (error %v)", m.Status(), m.StatusIsError())
	}
}

func TestCtrlSWithBlankNameKeepsPromptWithError(t *testing.T) {
	m, store := bookmarkModel(t, tree(t))
	m = press(t, m, "ctrl+s")
	m = typeText(t, m, "  ")
	m = press(t, m, "enter")
	if !m.InputPending() {
		t.Fatal("prompt closed on a blank name")
	}
	if !strings.Contains(m.View(), "empty") {
		t.Fatalf("error not shown:\n%s", m.View())
	}
	if names := bookmarkNames(t, store); len(names) != 0 {
		t.Fatalf("saved %v", names)
	}
}

func TestCtrlSWithExistingNameAsksBeforeReplacing(t *testing.T) {
	root := tree(t, "right/")
	for _, tt := range []struct {
		answer    string
		wantRight string
	}{
		{"y", filepath.Join(root, "right")},
		{"n", "/old/right"},
	} {
		t.Run(tt.answer, func(t *testing.T) {
			m, store := bookmarkModel(t, root)
			mustPut(t, store, bookmarks.Bookmark{Name: "Work", Left: "/old/left", Right: "/old/right"})

			m = press(t, m, "ctrl+s")
			m = typeText(t, m, "work")
			m = press(t, m, "enter")
			if !m.ConfirmPending() {
				t.Fatal("existing name should ask for confirmation")
			}
			if view := m.View(); !strings.Contains(view, "Replace") || !strings.Contains(view, "/old/left") {
				t.Fatalf("confirmation does not describe the replacement:\n%s", view)
			}
			m = press(t, m, tt.answer)

			got, _, _ := store.Get("work")
			if got.Right != tt.wantRight {
				t.Fatalf("after %q right = %q, want %q", tt.answer, got.Right, tt.wantRight)
			}
			if names := bookmarkNames(t, store); len(names) != 1 {
				t.Fatalf("bookmarks = %v, want one", names)
			}
		})
	}
}

func TestCtrlBListsBookmarksSortedAndEnterOpensBoth(t *testing.T) {
	root := tree(t, "a/", "b/", "c/", "d/")
	m, store := bookmarkModel(t, root)
	mustPut(t, store, bookmarks.Bookmark{Name: "zeta", Left: filepath.Join(root, "a"), Right: filepath.Join(root, "b")})
	mustPut(t, store, bookmarks.Bookmark{Name: "alpha", Left: filepath.Join(root, "c"), Right: filepath.Join(root, "d")})

	m = press(t, m, "j") // move the panel cursor so we can see ctrl+b works from anywhere
	m = press(t, m, "ctrl+b")
	if !m.BookmarksOpen() {
		t.Fatal("ctrl+b should open the bookmark table")
	}
	view := m.View()
	if a, z := strings.Index(view, "alpha"), strings.Index(view, "zeta"); a < 0 || z < 0 || a > z {
		t.Fatalf("table not sorted by name:\n%s", view)
	}

	m = press(t, m, "j", "enter")
	if m.BookmarksOpen() {
		t.Fatal("table still open after enter")
	}
	if got := []string{m.Panel(0).Dir(), m.Panel(1).Dir()}; !reflect.DeepEqual(got, []string{filepath.Join(root, "a"), filepath.Join(root, "b")}) {
		t.Fatalf("panels = %v", got)
	}
	if !strings.Contains(m.Status(), "zeta") || m.StatusIsError() {
		t.Fatalf("status = %q", m.Status())
	}
}

func TestBookmarkTableRowsWithLongPathsDoNotWrap(t *testing.T) {
	root := tree(t)
	m, store := bookmarkModel(t, root)
	// The 50th character is a hyphen, so the truncated column ends in "-…"
	// and lipgloss can break the row there when it is one column too wide.
	long := "/tmp/claude-1000/-home-abcd-documentos-roma-repos-copiar-tui/0f1e2d3c/scratch/play"
	mustPut(t, store, bookmarks.Bookmark{Name: "api", Left: long, Right: long})
	mustPut(t, store, bookmarks.Bookmark{Name: "deploy", Left: long, Right: long})

	// Render with colors as a real terminal does: the overflow only shows
	// once the highlighted row carries ANSI styles.
	prev := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.TrueColor)
	t.Cleanup(func() { lipgloss.SetColorProfile(prev) })

	m = press(t, m, "ctrl+b")
	// Between the header and the blank line before the hints there must be
	// exactly one line per bookmark; a wrapped row adds a line.
	lines := strings.Split(ansi.Strip(m.View()), "\n")
	start := slices.IndexFunc(lines, func(l string) bool { return strings.Contains(l, "Name") && strings.Contains(l, "Right") })
	end := slices.IndexFunc(lines, func(l string) bool { return strings.Contains(l, "esc close") })
	if start < 0 || end < 0 || end-start-2 != 2 {
		t.Fatalf("want 2 table rows, got lines %d..%d:\n%s", start, end, m.View())
	}
}

func TestBookmarkTableEscClosesAndEmptyTableExplainsHowToSave(t *testing.T) {
	m, _ := bookmarkModel(t, tree(t))
	m = press(t, m, "ctrl+b")
	if !strings.Contains(m.View(), "ctrl+s") {
		t.Fatalf("empty table should explain ctrl+s:\n%s", m.View())
	}
	m = press(t, m, "enter")
	if !m.BookmarksOpen() {
		t.Fatal("enter on an empty table closed it")
	}
	m = press(t, m, "esc")
	if m.BookmarksOpen() {
		t.Fatal("esc should close the table")
	}
}

func TestBrokenBookmarkWarnsAndCanBeDeleted(t *testing.T) {
	root := tree(t)
	m, store := bookmarkModel(t, root)
	gone := filepath.Join(root, "gone")
	mustPut(t, store, bookmarks.Bookmark{Name: "old", Left: gone + "-left", Right: gone + "-right"})

	m = press(t, m, "ctrl+b", "enter")
	view := m.View()
	// Paths are wrapped in the dialog, so look for their last element.
	for _, want := range []string{"broken", "Left folder not found", "Right folder not found", "gone-left", "gone-right"} {
		if !strings.Contains(view, want) {
			t.Fatalf("dialog lacks %q:\n%s", want, view)
		}
	}
	if strings.Contains(view, "Go anyway") {
		t.Fatalf("go anyway offered with no folder left:\n%s", view)
	}
	if m.Panel(0).Dir() != root || m.Panel(1).Dir() != root {
		t.Fatal("panels moved for a broken bookmark")
	}

	m = press(t, m, "esc")
	if !m.BookmarksOpen() || len(bookmarkNames(t, store)) != 1 {
		t.Fatal("cancel should return to the table and keep the bookmark")
	}

	m = press(t, m, "enter", "d")
	if names := bookmarkNames(t, store); len(names) != 0 {
		t.Fatalf("bookmarks after delete = %v", names)
	}
	if !m.BookmarksOpen() || strings.Contains(m.View(), "is broken") {
		t.Fatalf("should be back in the table:\n%s", m.View())
	}
}

func TestPartlyBrokenBookmarkCanGoAnyway(t *testing.T) {
	root := tree(t, "kept/", "start/")
	kept, gone := filepath.Join(root, "kept"), filepath.Join(root, "gone")
	start := filepath.Join(root, "start")

	t.Run("go anyway opens the existing folder only", func(t *testing.T) {
		m, store := bookmarkModel(t, root)
		_ = m.panels[0].cd(start, "")
		_ = m.panels[1].cd(start, "")
		mustPut(t, store, bookmarks.Bookmark{Name: "half", Left: gone, Right: kept})

		m = press(t, m, "ctrl+b", "enter")
		view := m.View()
		if !strings.Contains(view, "Go anyway") || !strings.Contains(view, "Left folder not found") || !strings.Contains(view, "Right folder exists") {
			t.Fatalf("dialog should name the missing folder and offer go anyway:\n%s", view)
		}
		m = press(t, m, "enter") // go anyway is the default
		if m.BookmarksOpen() {
			t.Fatal("table still open")
		}
		if m.Panel(0).Dir() != start || m.Panel(1).Dir() != kept {
			t.Fatalf("panels = %q, %q", m.Panel(0).Dir(), m.Panel(1).Dir())
		}
		if !strings.Contains(m.Status(), "not found") {
			t.Fatalf("status should mention the missing folder: %q", m.Status())
		}
		if len(bookmarkNames(t, store)) != 1 {
			t.Fatal("go anyway deleted the bookmark")
		}
	})

	t.Run("delete removes it", func(t *testing.T) {
		m, store := bookmarkModel(t, root)
		mustPut(t, store, bookmarks.Bookmark{Name: "half", Left: kept, Right: gone})
		m = press(t, m, "ctrl+b", "enter", "d")
		if names := bookmarkNames(t, store); len(names) != 0 {
			t.Fatalf("bookmarks = %v", names)
		}
		if m.Panel(0).Dir() != root {
			t.Fatal("delete should not navigate")
		}
	})
}

func TestBookmarkTableDeleteAsksFirst(t *testing.T) {
	root := tree(t)
	for _, tt := range []struct {
		answer string
		want   []string
	}{
		{"n", []string{"a", "b"}},
		{"y", []string{"b"}},
	} {
		t.Run(tt.answer, func(t *testing.T) {
			m, store := bookmarkModel(t, root)
			mustPut(t, store, bookmarks.Bookmark{Name: "a", Left: root, Right: root})
			mustPut(t, store, bookmarks.Bookmark{Name: "b", Left: root, Right: root})

			m = press(t, m, "ctrl+b", "d")
			if !m.ConfirmPending() {
				t.Fatal("d should ask for confirmation")
			}
			m = press(t, m, tt.answer)
			if got := bookmarkNames(t, store); !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("bookmarks = %v, want %v", got, tt.want)
			}
			if !m.BookmarksOpen() {
				t.Fatal("table should stay open")
			}
		})
	}
}

func TestBookmarkTableRenameAsksFirst(t *testing.T) {
	root := tree(t)
	for _, tt := range []struct {
		answer string
		want   []string
	}{
		{"n", []string{"a", "b"}},
		{"y", []string{"b", "c"}},
	} {
		t.Run(tt.answer, func(t *testing.T) {
			m, store := bookmarkModel(t, root)
			mustPut(t, store, bookmarks.Bookmark{Name: "a", Left: root, Right: root})
			mustPut(t, store, bookmarks.Bookmark{Name: "b", Left: root, Right: root})

			m = press(t, m, "ctrl+b", "r")
			if got := m.InputValue(); got != "a" {
				t.Fatalf("rename prompt prefilled with %q", got)
			}
			m = press(t, m, "ctrl+u")
			m = typeText(t, m, "c")
			m = press(t, m, "enter")
			if !m.ConfirmPending() {
				t.Fatal("rename should ask for confirmation")
			}
			m = press(t, m, tt.answer)
			if got := bookmarkNames(t, store); !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("bookmarks = %v, want %v", got, tt.want)
			}
			if !m.BookmarksOpen() {
				t.Fatal("table should stay open")
			}
		})
	}
}

func TestBookmarkRenameOntoOtherNameIsRefused(t *testing.T) {
	root := tree(t)
	m, store := bookmarkModel(t, root)
	mustPut(t, store, bookmarks.Bookmark{Name: "a", Left: root, Right: root})
	mustPut(t, store, bookmarks.Bookmark{Name: "b", Left: root, Right: root})

	m = press(t, m, "ctrl+b", "r", "ctrl+u")
	m = typeText(t, m, "B")
	m = press(t, m, "enter")
	if !m.InputPending() || m.ConfirmPending() {
		t.Fatal("prompt should stay open with an error")
	}
	if !strings.Contains(m.View(), "already exists") {
		t.Fatalf("error not shown:\n%s", m.View())
	}
}

func TestBookmarksReportStoreErrors(t *testing.T) {
	t.Run("no store", func(t *testing.T) {
		m := sized(New(tree(t), Options{}))
		for _, k := range []string{"ctrl+s", "ctrl+b"} {
			m = press(t, m, k)
			if m.BookmarksOpen() || m.InputPending() || !m.StatusIsError() {
				t.Fatalf("%s without a store: open=%v input=%v status=%q", k, m.BookmarksOpen(), m.InputPending(), m.Status())
			}
		}
	})
	t.Run("corrupt file", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "bookmarks.json")
		if err := os.WriteFile(path, []byte("{oops"), 0o600); err != nil {
			t.Fatal(err)
		}
		m := sized(New(tree(t), Options{Bookmarks: bookmarks.NewStore(path)}))
		m = press(t, m, "ctrl+b")
		if m.BookmarksOpen() || !m.StatusIsError() || !strings.Contains(m.Status(), "corrupt") {
			t.Fatalf("open=%v status=%q", m.BookmarksOpen(), m.Status())
		}
	})
}
