package fsops

import (
	"os"
	"path/filepath"
	"testing"
)

func writeFile(t *testing.T, path string, size int) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, make([]byte, size), 0o644); err != nil {
		t.Fatal(err)
	}
}

func names(entries []Entry) []string {
	out := make([]string, len(entries))
	for i, e := range entries {
		out[i] = e.Name
	}
	return out
}

func equal(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestListDirSortsDirectoriesFirstThenCaseInsensitiveName(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "beta.txt"), 1)
	writeFile(t, filepath.Join(dir, "Alpha.txt"), 1)
	writeFile(t, filepath.Join(dir, "zeta", "inner.txt"), 1)
	writeFile(t, filepath.Join(dir, "Docs", "inner.txt"), 1)

	entries, err := ListDir(dir, false)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"Docs", "zeta", "Alpha.txt", "beta.txt"}
	if got := names(entries); !equal(got, want) {
		t.Fatalf("order = %v, want %v", got, want)
	}
}

func TestListDirHiddenFiles(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, ".secret"), 1)
	writeFile(t, filepath.Join(dir, "visible"), 1)

	tests := []struct {
		name       string
		showHidden bool
		want       []string
	}{
		{name: "hidden files are skipped by default", showHidden: false, want: []string{"visible"}},
		{name: "hidden files are listed when enabled", showHidden: true, want: []string{".secret", "visible"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entries, err := ListDir(dir, tt.showHidden)
			if err != nil {
				t.Fatal(err)
			}
			if got := names(entries); !equal(got, tt.want) {
				t.Fatalf("names = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestListDirReportsEntryDetails(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "file.bin"), 2048)
	if err := os.Mkdir(filepath.Join(dir, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("sub", filepath.Join(dir, "link-to-sub")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("missing", filepath.Join(dir, "broken")); err != nil {
		t.Fatal(err)
	}

	entries, err := ListDir(dir, false)
	if err != nil {
		t.Fatal(err)
	}
	byName := map[string]Entry{}
	for _, e := range entries {
		byName[e.Name] = e
	}

	if e := byName["file.bin"]; e.IsDir || e.IsLink || e.Size != 2048 || e.Path != filepath.Join(dir, "file.bin") {
		t.Fatalf("file.bin = %+v", e)
	}
	if e := byName["sub"]; !e.IsDir || e.IsLink {
		t.Fatalf("sub = %+v, want dir", e)
	}
	if e := byName["link-to-sub"]; !e.IsDir || !e.IsLink || e.LinkTarget != "sub" {
		t.Fatalf("link-to-sub = %+v, want link to dir", e)
	}
	if e := byName["broken"]; e.IsDir || !e.IsLink {
		t.Fatalf("broken = %+v, want link that is not a dir", e)
	}
}

func TestListDirMissingDirectoryIsAnError(t *testing.T) {
	if _, err := ListDir(filepath.Join(t.TempDir(), "nope"), false); err == nil {
		t.Fatal("expected error for missing directory")
	}
}
