package bookmarks

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func newStore(t *testing.T) (*Store, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "nested", "bookmarks.json")
	return NewStore(path), path
}

func names(t *testing.T, s *Store) []string {
	t.Helper()
	list, err := s.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	var out []string
	for _, b := range list {
		out = append(out, b.Name)
	}
	return out
}

func TestListWithoutFileIsEmpty(t *testing.T) {
	s, path := newStore(t)
	list, err := s.List()
	if err != nil || len(list) != 0 {
		t.Fatalf("List() = %v, %v; want empty, nil", list, err)
	}
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("List created the file: %v", err)
	}
}

func TestPutPersistsAndListsSortedByName(t *testing.T) {
	s, path := newStore(t)
	for _, b := range []Bookmark{
		{Name: "work", Left: "/w/l", Right: "/w/r"},
		{Name: "Backup", Left: "/b/l", Right: "/b/r"},
		{Name: "music", Left: "/m/l", Right: "/m/r"},
	} {
		if err := s.Put(b); err != nil {
			t.Fatalf("Put(%v): %v", b, err)
		}
	}
	if got, want := names(t, NewStore(path)), []string{"Backup", "music", "work"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("names = %v, want %v", got, want)
	}
	b, ok, err := NewStore(path).Get("WORK")
	if err != nil || !ok || b != (Bookmark{Name: "work", Left: "/w/l", Right: "/w/r"}) {
		t.Fatalf("Get(WORK) = %v, %v, %v", b, ok, err)
	}
}

func TestPutReplacesSameNameIgnoringCase(t *testing.T) {
	s, _ := newStore(t)
	if err := s.Put(Bookmark{Name: "Work", Left: "/old", Right: "/old"}); err != nil {
		t.Fatal(err)
	}
	if err := s.Put(Bookmark{Name: "work", Left: "/new/l", Right: "/new/r"}); err != nil {
		t.Fatal(err)
	}
	list, err := s.List()
	if err != nil {
		t.Fatal(err)
	}
	want := []Bookmark{{Name: "work", Left: "/new/l", Right: "/new/r"}}
	if !reflect.DeepEqual(list, want) {
		t.Fatalf("List = %v, want %v", list, want)
	}
}

func TestPutTrimsAndRejectsEmptyName(t *testing.T) {
	s, _ := newStore(t)
	if err := s.Put(Bookmark{Name: "   ", Left: "/l", Right: "/r"}); !errors.Is(err, ErrEmptyName) {
		t.Fatalf("Put(blank) = %v, want ErrEmptyName", err)
	}
	if err := s.Put(Bookmark{Name: "  docs  ", Left: "/l", Right: "/r"}); err != nil {
		t.Fatal(err)
	}
	if got := names(t, s); !reflect.DeepEqual(got, []string{"docs"}) {
		t.Fatalf("names = %v", got)
	}
}

func TestRename(t *testing.T) {
	s, _ := newStore(t)
	_ = s.Put(Bookmark{Name: "a", Left: "/a", Right: "/a"})
	_ = s.Put(Bookmark{Name: "b", Left: "/b", Right: "/b"})

	if err := s.Rename("a", "B"); !errors.Is(err, ErrExists) {
		t.Fatalf("Rename onto other bookmark = %v, want ErrExists", err)
	}
	if err := s.Rename("missing", "x"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Rename missing = %v, want ErrNotFound", err)
	}
	if err := s.Rename("a", " "); !errors.Is(err, ErrEmptyName) {
		t.Fatalf("Rename to blank = %v, want ErrEmptyName", err)
	}
	if err := s.Rename("a", "A"); err != nil {
		t.Fatalf("Rename changing only case: %v", err)
	}
	if err := s.Rename("A", "zeta"); err != nil {
		t.Fatal(err)
	}
	b, ok, _ := s.Get("zeta")
	if !ok || b.Left != "/a" {
		t.Fatalf("renamed bookmark = %v, %v", b, ok)
	}
	if got := names(t, s); !reflect.DeepEqual(got, []string{"b", "zeta"}) {
		t.Fatalf("names = %v", got)
	}
}

func TestDelete(t *testing.T) {
	s, _ := newStore(t)
	_ = s.Put(Bookmark{Name: "a", Left: "/a", Right: "/a"})
	_ = s.Put(Bookmark{Name: "b", Left: "/b", Right: "/b"})
	if err := s.Delete("A"); err != nil {
		t.Fatal(err)
	}
	if err := s.Delete("a"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("second Delete = %v, want ErrNotFound", err)
	}
	if got := names(t, s); !reflect.DeepEqual(got, []string{"b"}) {
		t.Fatalf("names = %v", got)
	}
}

func TestCorruptFileIsReportedAndNeverOverwritten(t *testing.T) {
	s, path := newStore(t)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := s.List(); err == nil {
		t.Fatal("List of corrupt file returned no error")
	}
	if err := s.Put(Bookmark{Name: "x", Left: "/x", Right: "/x"}); err == nil {
		t.Fatal("Put over corrupt file returned no error")
	}
	data, _ := os.ReadFile(path)
	if string(data) != "{not json" {
		t.Fatalf("corrupt file was overwritten: %q", data)
	}
}

func TestSaveLeavesNoTempFiles(t *testing.T) {
	s, path := newStore(t)
	_ = s.Put(Bookmark{Name: "a", Left: "/a", Right: "/a"})
	entries, err := os.ReadDir(filepath.Dir(path))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != "bookmarks.json" {
		t.Fatalf("dir holds %v, want only bookmarks.json", entries)
	}
}

func TestDefaultPathIsUnderUserConfigDir(t *testing.T) {
	cfg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", cfg)
	t.Setenv("HOME", cfg)
	t.Setenv("AppData", cfg)
	base, err := os.UserConfigDir()
	if err != nil {
		t.Skip(err)
	}
	got, err := DefaultPath()
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(base, "copiar", "bookmarks.json"); got != want {
		t.Fatalf("DefaultPath() = %q, want %q", got, want)
	}
}
