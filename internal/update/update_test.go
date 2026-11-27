package update

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/RHAM91/copy-tui/internal/updater"
)

type catalog struct {
	t        *testing.T
	priv     ed25519.PrivateKey
	pub      string
	version  string
	binary   []byte
	force    bool
	status   int // overrides /version when non-zero
	tamper   bool
	requests int
}

func newCatalog(t *testing.T, version string, binary []byte) (*catalog, *Service) {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	c := &catalog{t: t, priv: priv, pub: base64.StdEncoding.EncodeToString(pub), version: version, binary: binary}
	srv := httptest.NewServer(c)
	t.Cleanup(srv.Close)
	return c, NewWithConfig(updater.Config{BaseURL: srv.URL, App: App, PublicKey: c.pub, Token: "cat_dl_test"})
}

func (c *catalog) sum() string {
	h := sha256.Sum256(c.binary)
	return hex.EncodeToString(h[:])
}

func (c *catalog) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	c.requests++
	if r.Header.Get("Authorization") != "Bearer cat_dl_test" {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	switch r.URL.Path {
	case "/api/" + App + "/version":
		if c.status != 0 {
			w.WriteHeader(c.status)
			return
		}
		sig := ed25519.Sign(c.priv, updater.SignedMessage(App, c.version, runtime.GOOS, runtime.GOARCH, c.sum()))
		_ = json.NewEncoder(w).Encode(updater.Release{
			App: App, Version: c.version, OS: runtime.GOOS, Arch: runtime.GOARCH,
			SHA256: c.sum(), Size: int64(len(c.binary)),
			Signature: base64.StdEncoding.EncodeToString(sig), Force: c.force,
		})
	case "/api/" + App + "/download":
		if got := r.URL.Query().Get("version"); got != c.version {
			c.t.Errorf("download asked for version %q, want the pinned %q", got, c.version)
		}
		body := c.binary
		if c.tamper {
			body = append([]byte(nil), c.binary...)
			body[0] ^= 0xff
		}
		w.Header().Set("X-Version", c.version)
		_, _ = w.Write(body)
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

func fakeExe(t *testing.T) string {
	t.Helper()
	exe := filepath.Join(t.TempDir(), "copiar")
	if err := os.WriteFile(exe, []byte("old"), 0o755); err != nil {
		t.Fatal(err)
	}
	return exe
}

func read(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func leftovers(t *testing.T, dir string) int {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	return len(entries)
}

func TestCheck(t *testing.T) {
	tests := []struct {
		name, remote, current string
		force                 bool
		wantNewer             bool
		wantLatest            string
	}{
		{name: "newer release", remote: "0.2.0", current: "0.1.0", wantNewer: true, wantLatest: "0.2.0"},
		{name: "semver, not string, order", remote: "1.10.0", current: "1.9.0", wantNewer: true, wantLatest: "1.10.0"},
		{name: "same version", remote: "0.1.0", current: "0.1.0", wantLatest: "0.1.0"},
		{name: "same version even when forced", remote: "0.1.0", current: "0.1.0", force: true, wantLatest: "0.1.0"},
		{name: "older without force is ignored", remote: "0.1.0", current: "0.2.0", wantLatest: "0.2.0"},
		{name: "older with force is a rollback", remote: "0.1.0", current: "0.2.0", force: true, wantNewer: true, wantLatest: "0.1.0"},
		{name: "leading v on the running version", remote: "0.1.0", current: "v0.1.0", wantLatest: "0.1.0"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, svc := newCatalog(t, tt.remote, []byte("bin"))
			c.force = tt.force
			latest, newer, err := svc.Check(context.Background(), tt.current)
			if err != nil {
				t.Fatal(err)
			}
			if latest != tt.wantLatest || newer != tt.wantNewer {
				t.Fatalf("Check() = (%q, %v), want (%q, %v)", latest, newer, tt.wantLatest, tt.wantNewer)
			}
		})
	}
}

func TestCheckTreatsMissingOrBlockedChannelAsNoUpdate(t *testing.T) {
	for _, status := range []int{http.StatusNotFound, http.StatusServiceUnavailable} {
		c, svc := newCatalog(t, "9.9.9", []byte("bin"))
		c.status = status
		latest, newer, err := svc.Check(context.Background(), "0.1.0")
		if err != nil || newer || latest != "0.1.0" {
			t.Fatalf("HTTP %d: Check() = (%q, %v, %v), want (\"0.1.0\", false, nil)", status, latest, newer, err)
		}
	}
}

func TestCheckReportsRejectedToken(t *testing.T) {
	_, svc := newCatalog(t, "0.2.0", []byte("bin"))
	svc.cfg.Token = "cat_dl_wrong"
	if _, _, err := svc.Check(context.Background(), "0.1.0"); !errors.Is(err, updater.ErrUnauthorized) {
		t.Fatalf("err = %v, want ErrUnauthorized", err)
	}
}

func TestWithoutTokenNoRequestIsMade(t *testing.T) {
	c, svc := newCatalog(t, "0.2.0", []byte("bin"))
	svc.cfg.Token = ""
	if _, _, err := svc.Check(context.Background(), "0.1.0"); !errors.Is(err, ErrNoToken) {
		t.Fatalf("err = %v, want ErrNoToken", err)
	}
	if _, _, err := svc.Apply(context.Background(), "0.1.0", fakeExe(t)); !errors.Is(err, ErrNoToken) {
		t.Fatalf("Apply err = %v, want ErrNoToken", err)
	}
	if c.requests != 0 {
		t.Fatalf("%d requests made without a token", c.requests)
	}
}

func TestApplyReplacesExecutable(t *testing.T) {
	_, svc := newCatalog(t, "0.2.0", []byte("new binary"))
	exe := fakeExe(t)
	version, updated, err := svc.Apply(context.Background(), "0.1.0", exe)
	if err != nil || !updated || version != "0.2.0" {
		t.Fatalf("Apply() = (%q, %v, %v)", version, updated, err)
	}
	if got := read(t, exe); got != "new binary" {
		t.Fatalf("executable = %q, want the new binary", got)
	}
	if info, _ := os.Stat(exe); runtime.GOOS != "windows" && info.Mode().Perm() != 0o755 {
		t.Fatalf("mode = %v, want 0755", info.Mode().Perm())
	}
}

func TestApplyInstallsForcedRollback(t *testing.T) {
	c, svc := newCatalog(t, "0.1.0", []byte("old release"))
	c.force = true
	exe := fakeExe(t)
	if _, updated, err := svc.Apply(context.Background(), "0.2.0", exe); err != nil || !updated {
		t.Fatalf("Apply() updated=%v err=%v", updated, err)
	}
	if got := read(t, exe); got != "old release" {
		t.Fatalf("executable = %q", got)
	}
}

func TestApplyDoesNothingWhenUpToDate(t *testing.T) {
	_, svc := newCatalog(t, "0.1.0", []byte("same"))
	exe := fakeExe(t)
	version, updated, err := svc.Apply(context.Background(), "0.1.0", exe)
	if err != nil || updated || version != "0.1.0" {
		t.Fatalf("Apply() = (%q, %v, %v)", version, updated, err)
	}
	if got := read(t, exe); got != "old" {
		t.Fatalf("executable changed to %q", got)
	}
}

func TestApplyRejectsTamperedBinaryAndLeavesNoTempFile(t *testing.T) {
	c, svc := newCatalog(t, "0.2.0", []byte("new binary"))
	c.tamper = true
	exe := fakeExe(t)
	if _, updated, err := svc.Apply(context.Background(), "0.1.0", exe); err == nil || updated {
		t.Fatalf("Apply() err=%v updated=%v, want a rejection", err, updated)
	}
	if got := read(t, exe); got != "old" {
		t.Fatalf("executable changed to %q", got)
	}
	if n := leftovers(t, filepath.Dir(exe)); n != 1 {
		t.Fatalf("%d files in the executable directory, want only the executable", n)
	}
}

func TestApplyRejectsSignatureFromAnotherKey(t *testing.T) {
	_, svc := newCatalog(t, "0.2.0", []byte("new binary"))
	other, _, _ := ed25519.GenerateKey(nil)
	svc.cfg.PublicKey = base64.StdEncoding.EncodeToString(other)
	exe := fakeExe(t)
	if _, updated, err := svc.Apply(context.Background(), "0.1.0", exe); err == nil || updated {
		t.Fatalf("Apply() err=%v updated=%v, want a rejection", err, updated)
	}
	if got := read(t, exe); got != "old" {
		t.Fatalf("executable changed to %q", got)
	}
	if n := leftovers(t, filepath.Dir(exe)); n != 1 {
		t.Fatalf("%d files left behind", n)
	}
}

func TestRefusesDevelopmentBuilds(t *testing.T) {
	c, svc := newCatalog(t, "0.2.0", []byte("bin"))
	for _, v := range []string{"dev", "0.2.0-SNAPSHOT-abc123", ""} {
		if _, _, err := svc.Check(context.Background(), v); !errors.Is(err, ErrDevBuild) {
			t.Fatalf("Check(%q) err = %v, want ErrDevBuild", v, err)
		}
		if _, _, err := svc.Apply(context.Background(), v, fakeExe(t)); !errors.Is(err, ErrDevBuild) {
			t.Fatalf("Apply(%q) err = %v, want ErrDevBuild", v, err)
		}
	}
	if c.requests != 0 {
		t.Fatalf("%d requests made for development builds", c.requests)
	}
}

func TestOfficialConfigHasEmbeddedKey(t *testing.T) {
	pub, err := base64.StdEncoding.DecodeString(PublicKey)
	if err != nil || len(pub) != ed25519.PublicKeySize {
		t.Fatalf("PublicKey is not a valid Ed25519 key: %v", err)
	}
	t.Setenv(TokenEnv, "cat_dl_x")
	svc := New()
	if svc.cfg.BaseURL != CatalogURL || svc.cfg.App != App || svc.cfg.PublicKey != PublicKey || svc.cfg.Token != "cat_dl_x" {
		t.Fatalf("unexpected config %+v", svc.cfg)
	}
}
