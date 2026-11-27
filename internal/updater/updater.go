// Package updater is the reference client for the update catalog.
//
// It depends only on the Go standard library so it can be copied verbatim
// into any app. Flow: Check -> Download (verifies size, SHA-256 and the
// Ed25519 signature against the EMBEDDED public key) -> Apply.
package updater

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// Config is fixed at build time except Token, which comes from the environment.
type Config struct {
	BaseURL        string // e.g. https://updates.r0ma.cloud (HTTPS only in production)
	App            string // app name as registered in the panel
	CurrentVersion string // version of the running binary, e.g. "1.4.0"
	PublicKey      string // app Ed25519 public key, standard base64, EMBEDDED in the binary
	Token          string // download token (cat_dl_...), read from an env var
	HTTPClient     *http.Client
}

// Release is the catalog answer for this app and platform.
type Release struct {
	App       string `json:"app"`
	Version   string `json:"version"`
	OS        string `json:"os"`
	Arch      string `json:"arch"`
	SHA256    string `json:"sha256"`
	Size      int64  `json:"size"`
	Signature string `json:"signature"`
	Force     bool   `json:"force"`
}

// ErrUnauthorized means the download token is missing, wrong or rotated.
var ErrUnauthorized = errors.New("update catalog rejected the token")

func (c Config) client() *http.Client {
	if c.HTTPClient != nil {
		return c.HTTPClient
	}
	return &http.Client{Timeout: 10 * time.Minute}
}

// checkURL enforces HTTPS; plain HTTP is accepted only for loopback test servers.
func (c Config) checkURL() error {
	u, err := url.Parse(c.BaseURL)
	if err != nil {
		return fmt.Errorf("invalid catalog URL: %w", err)
	}
	host := u.Hostname()
	if u.Scheme == "https" || (u.Scheme == "http" && (host == "127.0.0.1" || host == "localhost" || host == "::1")) {
		return nil
	}
	return fmt.Errorf("catalog URL must use https, got %q", c.BaseURL)
}

func (c Config) get(ctx context.Context, path string, q url.Values) (*http.Response, error) {
	u := strings.TrimRight(c.BaseURL, "/") + "/api/" + url.PathEscape(c.App) + path + "?" + q.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)
	return c.client().Do(req)
}

// Check returns the release to install, or nil when the app is up to date,
// the channel is blocked (503) or nothing is published for this platform (404).
func Check(ctx context.Context, c Config) (*Release, error) {
	if c.Token == "" {
		return nil, nil // updates disabled: no token configured
	}
	if err := c.checkURL(); err != nil {
		return nil, err
	}
	q := url.Values{"os": {runtime.GOOS}, "arch": {runtime.GOARCH}}
	resp, err := c.get(ctx, "/version", q)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	switch resp.StatusCode {
	case http.StatusOK:
	case http.StatusNotFound, http.StatusServiceUnavailable:
		return nil, nil
	case http.StatusUnauthorized:
		return nil, ErrUnauthorized
	default:
		return nil, fmt.Errorf("version check: HTTP %d", resp.StatusCode)
	}
	var rel Release
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<16)).Decode(&rel); err != nil {
		return nil, fmt.Errorf("version check: %w", err)
	}
	if rel.App != c.App || rel.OS != runtime.GOOS || rel.Arch != runtime.GOARCH {
		return nil, fmt.Errorf("version check: answer is for %s %s/%s", rel.App, rel.OS, rel.Arch)
	}
	cmp, err := CompareVersions(rel.Version, c.CurrentVersion)
	if err != nil {
		return nil, err
	}
	// Newer wins; an older/different version is accepted ONLY when forced (rollback).
	if cmp > 0 || (rel.Force && cmp != 0) {
		return &rel, nil
	}
	return nil, nil
}

// Download fetches the exact version announced by Check into dir and returns
// the path of a fully verified file. On any mismatch the file is deleted.
func Download(ctx context.Context, c Config, rel *Release, dir string) (string, error) {
	if err := c.checkURL(); err != nil {
		return "", err
	}
	q := url.Values{"os": {rel.OS}, "arch": {rel.Arch}, "version": {rel.Version}}
	resp, err := c.get(ctx, "/download", q)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusUnauthorized {
		return "", ErrUnauthorized
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download: HTTP %d", resp.StatusCode)
	}
	if v := resp.Header.Get("X-Version"); v != rel.Version {
		return "", fmt.Errorf("download: server sent version %q, expected %q", v, rel.Version)
	}

	tmp, err := os.CreateTemp(dir, "."+c.App+"-update-*")
	if err != nil {
		return "", err
	}
	ok := false
	defer func() {
		tmp.Close()
		if !ok {
			os.Remove(tmp.Name())
		}
	}()
	h := sha256.New()
	n, err := io.Copy(io.MultiWriter(tmp, h), io.LimitReader(resp.Body, rel.Size+1))
	if err != nil {
		return "", fmt.Errorf("download: %w", err)
	}
	if n != rel.Size {
		return "", fmt.Errorf("download: got %d bytes, expected %d", n, rel.Size)
	}
	sum := hex.EncodeToString(h.Sum(nil))
	if sum != strings.ToLower(rel.SHA256) {
		return "", fmt.Errorf("download: sha256 mismatch (got %s)", sum)
	}
	if err := Verify(c.PublicKey, c.App, rel.Version, rel.OS, rel.Arch, sum, rel.Signature); err != nil {
		return "", err
	}
	if err := tmp.Sync(); err != nil {
		return "", err
	}
	ok = true
	return tmp.Name(), nil
}

// SignedMessage reproduces the exact bytes the CI signs.
func SignedMessage(app, version, goos, arch, sha256Hex string) []byte {
	return []byte("actualizaciones/v1\n" +
		"app=" + app + "\n" +
		"version=" + version + "\n" +
		"os=" + goos + "\n" +
		"arch=" + arch + "\n" +
		"sha256=" + strings.ToLower(sha256Hex) + "\n")
}

// Verify checks an artifact signature with the embedded public key.
func Verify(publicKeyB64, app, version, goos, arch, sha256Hex, signatureB64 string) error {
	pub, err := base64.StdEncoding.DecodeString(publicKeyB64)
	if err != nil || len(pub) != ed25519.PublicKeySize {
		return errors.New("verify: embedded public key is invalid")
	}
	sig, err := base64.StdEncoding.DecodeString(signatureB64)
	if err != nil || len(sig) != ed25519.SignatureSize {
		return errors.New("verify: signature is malformed")
	}
	if !ed25519.Verify(ed25519.PublicKey(pub), SignedMessage(app, version, goos, arch, sha256Hex), sig) {
		return errors.New("verify: signature does not match; refusing update")
	}
	return nil
}

// Apply replaces the running executable with the verified file. The file
// must live in the same directory as the executable so the rename is atomic.
// The new version runs on the next start.
func Apply(verifiedPath string) error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	if exe, err = filepath.EvalSymlinks(exe); err != nil {
		return err
	}
	return ApplyTo(verifiedPath, exe)
}

// ApplyTo is Apply for an explicit executable path.
func ApplyTo(verifiedPath, exe string) error {
	if err := os.Chmod(verifiedPath, 0o755); err != nil {
		return err
	}
	if runtime.GOOS != "windows" {
		return os.Rename(verifiedPath, exe)
	}
	// Windows cannot overwrite a running .exe, but it can rename it.
	old := exe + ".old"
	_ = os.Remove(old) // leftover from a previous update
	if err := os.Rename(exe, old); err != nil {
		return err
	}
	if err := os.Rename(verifiedPath, exe); err != nil {
		_ = os.Rename(old, exe)
		return err
	}
	return nil
}

// CleanupOld removes the previous Windows executable; call it at startup.
func CleanupOld() {
	if exe, err := os.Executable(); err == nil {
		_ = os.Remove(exe + ".old")
	}
}

// CompareVersions compares MAJOR.MINOR.PATCH[-PRERELEASE] versions (a leading
// "v" is ignored) following semver precedence. It returns -1, 0 or 1.
func CompareVersions(a, b string) (int, error) {
	pa, err := parseVersion(a)
	if err != nil {
		return 0, err
	}
	pb, err := parseVersion(b)
	if err != nil {
		return 0, err
	}
	for i := 0; i < 3; i++ {
		if pa.core[i] != pb.core[i] {
			return sign(pa.core[i] - pb.core[i]), nil
		}
	}
	switch {
	case len(pa.pre) == 0 && len(pb.pre) == 0:
		return 0, nil
	case len(pa.pre) == 0:
		return 1, nil // a release outranks its pre-releases
	case len(pb.pre) == 0:
		return -1, nil
	}
	for i := 0; i < len(pa.pre) && i < len(pb.pre); i++ {
		if c := comparePre(pa.pre[i], pb.pre[i]); c != 0 {
			return c, nil
		}
	}
	return sign(len(pa.pre) - len(pb.pre)), nil
}

type version struct {
	core [3]int
	pre  []string
}

func parseVersion(v string) (version, error) {
	var out version
	v = strings.TrimPrefix(v, "v")
	core, pre, hasPre := strings.Cut(v, "-")
	parts := strings.Split(core, ".")
	if len(parts) != 3 {
		return out, fmt.Errorf("invalid version %q", v)
	}
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 {
			return out, fmt.Errorf("invalid version %q", v)
		}
		out.core[i] = n
	}
	if hasPre {
		out.pre = strings.Split(pre, ".")
	}
	return out, nil
}

func comparePre(a, b string) int {
	na, errA := strconv.Atoi(a)
	nb, errB := strconv.Atoi(b)
	switch {
	case errA == nil && errB == nil:
		return sign(na - nb)
	case errA == nil:
		return -1 // numeric identifiers sort before alphanumeric ones
	case errB == nil:
		return 1
	}
	return strings.Compare(a, b)
}

func sign(n int) int {
	switch {
	case n > 0:
		return 1
	case n < 0:
		return -1
	}
	return 0
}
