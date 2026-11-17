package update

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/creativeprojects/go-selfupdate"
)

// fakeAsset and fakeRelease implement the selfupdate source interfaces.
type fakeAsset struct {
	id   int64
	name string
	data []byte
}

func (a fakeAsset) GetID() int64                  { return a.id }
func (a fakeAsset) GetName() string               { return a.name }
func (a fakeAsset) GetSize() int                  { return len(a.data) }
func (a fakeAsset) GetBrowserDownloadURL() string { return "https://example.invalid/" + a.name }

type fakeRelease struct {
	tag    string
	assets []fakeAsset
}

func (r fakeRelease) GetID() int64              { return 1 }
func (r fakeRelease) GetTagName() string        { return r.tag }
func (r fakeRelease) GetDraft() bool            { return false }
func (r fakeRelease) GetPrerelease() bool       { return false }
func (r fakeRelease) GetPublishedAt() time.Time { return time.Now() }
func (r fakeRelease) GetReleaseNotes() string   { return "" }
func (r fakeRelease) GetName() string           { return r.tag }
func (r fakeRelease) GetURL() string            { return "https://example.invalid/" + r.tag }
func (r fakeRelease) GetAssets() []selfupdate.SourceAsset {
	out := make([]selfupdate.SourceAsset, len(r.assets))
	for i, a := range r.assets {
		out[i] = a
	}
	return out
}

type fakeSource struct {
	releases []fakeRelease
	err      error
}

func (s fakeSource) ListReleases(context.Context, selfupdate.Repository) ([]selfupdate.SourceRelease, error) {
	if s.err != nil {
		return nil, s.err
	}
	out := make([]selfupdate.SourceRelease, len(s.releases))
	for i, r := range s.releases {
		out[i] = r
	}
	return out, nil
}

func (s fakeSource) DownloadReleaseAsset(_ context.Context, _ *selfupdate.Release, id int64) (io.ReadCloser, error) {
	for _, r := range s.releases {
		for _, a := range r.assets {
			if a.id == id {
				return io.NopCloser(bytes.NewReader(a.data)), nil
			}
		}
	}
	return nil, fmt.Errorf("asset %d not found", id)
}

// tarGz builds a GoReleaser-like archive holding the copiar binary.
func tarGz(t *testing.T, binary []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	for name, data := range map[string][]byte{"copiar": binary, "README.md": []byte("readme")} {
		if err := tw.WriteHeader(&tar.Header{Name: name, Mode: 0o755, Size: int64(len(data))}); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write(data); err != nil {
			t.Fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// release returns a release for the running platform whose archive contains
// binary; tamper corrupts the checksum.
func release(t *testing.T, version string, binary []byte, tamper bool) fakeRelease {
	t.Helper()
	archive := tarGz(t, binary)
	name := fmt.Sprintf("copiar_%s_%s_%s.tar.gz", version, runtime.GOOS, runtime.GOARCH)
	sum := sha256.Sum256(archive)
	if tamper {
		sum[0] ^= 0xff
	}
	checksums := fmt.Sprintf("%x  %s\n", sum, name)
	return fakeRelease{tag: "v" + version, assets: []fakeAsset{
		{id: 10, name: name, data: archive},
		{id: 11, name: "checksums.txt", data: []byte(checksums)},
	}}
}

func newService(t *testing.T, src fakeSource) *Service {
	t.Helper()
	s, err := New(src)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func fakeExe(t *testing.T) string {
	t.Helper()
	exe := filepath.Join(t.TempDir(), "copiar")
	if err := os.WriteFile(exe, []byte("old binary"), 0o755); err != nil {
		t.Fatal(err)
	}
	return exe
}

func TestCheck(t *testing.T) {
	src := fakeSource{releases: []fakeRelease{release(t, "0.2.0", []byte("new"), false)}}
	tests := []struct {
		name       string
		current    string
		wantLatest string
		wantNewer  bool
		wantErr    error
	}{
		{name: "older version sees the update", current: "0.1.0", wantLatest: "0.2.0", wantNewer: true},
		{name: "v prefix is accepted", current: "v0.1.0", wantLatest: "0.2.0", wantNewer: true},
		{name: "same version is up to date", current: "0.2.0", wantLatest: "0.2.0", wantNewer: false},
		{name: "newer local version is up to date", current: "0.3.0", wantLatest: "0.2.0", wantNewer: false},
		{name: "development build cannot be compared", current: "dev", wantErr: ErrDevBuild},
		{name: "snapshot build cannot be compared", current: "0.0.0-SNAPSHOT-20fe5c9", wantErr: ErrDevBuild},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			latest, newer, err := newService(t, src).Check(context.Background(), tt.current)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("err = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if latest != tt.wantLatest || newer != tt.wantNewer {
				t.Fatalf("Check = (%q, %v), want (%q, %v)", latest, newer, tt.wantLatest, tt.wantNewer)
			}
		})
	}
}

func TestCheckWithoutReleases(t *testing.T) {
	_, _, err := newService(t, fakeSource{}).Check(context.Background(), "0.1.0")
	if !errors.Is(err, ErrNoRelease) {
		t.Fatalf("err = %v, want ErrNoRelease", err)
	}
}

func TestCheckPropagatesSourceErrors(t *testing.T) {
	boom := errors.New("network down")
	_, _, err := newService(t, fakeSource{err: boom}).Check(context.Background(), "0.1.0")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestApplyReplacesExecutable(t *testing.T) {
	src := fakeSource{releases: []fakeRelease{release(t, "0.2.0", []byte("new binary"), false)}}
	exe := fakeExe(t)

	version, updated, err := newService(t, src).Apply(context.Background(), "0.1.0", exe)
	if err != nil {
		t.Fatal(err)
	}
	if !updated || version != "0.2.0" {
		t.Fatalf("Apply = (%q, %v), want (0.2.0, true)", version, updated)
	}
	got, _ := os.ReadFile(exe)
	if string(got) != "new binary" {
		t.Fatalf("executable content = %q", got)
	}
	info, _ := os.Stat(exe)
	if info.Mode().Perm()&0o100 == 0 {
		t.Fatalf("updated executable is not executable: %v", info.Mode())
	}
}

func TestApplyDoesNothingWhenUpToDate(t *testing.T) {
	src := fakeSource{releases: []fakeRelease{release(t, "0.2.0", []byte("new binary"), false)}}
	exe := fakeExe(t)

	version, updated, err := newService(t, src).Apply(context.Background(), "0.2.0", exe)
	if err != nil {
		t.Fatal(err)
	}
	if updated || version != "0.2.0" {
		t.Fatalf("Apply = (%q, %v), want (0.2.0, false)", version, updated)
	}
	if got, _ := os.ReadFile(exe); string(got) != "old binary" {
		t.Fatalf("executable changed: %q", got)
	}
}

func TestApplyRejectsChecksumMismatch(t *testing.T) {
	src := fakeSource{releases: []fakeRelease{release(t, "0.2.0", []byte("evil binary"), true)}}
	exe := fakeExe(t)

	if _, _, err := newService(t, src).Apply(context.Background(), "0.1.0", exe); err == nil {
		t.Fatal("expected checksum error")
	}
	if got, _ := os.ReadFile(exe); string(got) != "old binary" {
		t.Fatalf("executable replaced despite bad checksum: %q", got)
	}
}

func TestApplyRejectsReleaseWithoutChecksums(t *testing.T) {
	rel := release(t, "0.2.0", []byte("unverified"), false)
	rel.assets = rel.assets[:1]
	exe := fakeExe(t)

	if _, _, err := newService(t, fakeSource{releases: []fakeRelease{rel}}).Apply(context.Background(), "0.1.0", exe); err == nil {
		t.Fatal("expected error for a release without checksums.txt")
	}
	if got, _ := os.ReadFile(exe); string(got) != "old binary" {
		t.Fatalf("executable replaced without checksum: %q", got)
	}
}

func TestApplyRefusesDevelopmentBuild(t *testing.T) {
	src := fakeSource{releases: []fakeRelease{release(t, "0.2.0", []byte("new"), false)}}
	exe := fakeExe(t)
	if _, _, err := newService(t, src).Apply(context.Background(), "dev", exe); !errors.Is(err, ErrDevBuild) {
		t.Fatalf("err = %v, want ErrDevBuild", err)
	}
	if got, _ := os.ReadFile(exe); string(got) != "old binary" {
		t.Fatal("development build was replaced")
	}
}
