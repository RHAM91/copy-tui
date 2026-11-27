// Package update checks the private update catalog for a newer copiar and
// replaces the running executable with it, after verifying its size, SHA-256
// and Ed25519 signature against the public key compiled into the binary.
package update

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/RHAM91/copy-tui/internal/updater"
)

const (
	// CatalogURL is the base URL of the update catalog.
	CatalogURL = "https://updates.r0ma.cloud"
	// App is the name the app is registered under in the catalog.
	App = "copiar-tui"
	// PublicKey is the app's Ed25519 public key (standard base64). It is
	// compiled in on purpose: it must never come from the network or the
	// environment.
	PublicKey = "C05OkZIYpmPm2CpIr+bgzU11piMBdFua7YaVne+eLPo="
	// TokenEnv names the environment variable holding the download token.
	TokenEnv = "COPIAR_TUI_UPDATE_TOKEN"
)

var (
	// ErrDevBuild is returned for builds without a release version (local
	// "dev" or GoReleaser snapshot builds), which cannot be compared.
	ErrDevBuild = errors.New("development build: install a release to enable updates")
	// ErrNoToken is returned when the download token is not configured.
	ErrNoToken = errors.New("updates are disabled: set " + TokenEnv + " to your download token")
)

// Service checks for and applies updates.
type Service struct {
	cfg updater.Config
}

// New creates a Service for the official catalog, reading the download token
// from the environment.
func New() *Service {
	return NewWithConfig(updater.Config{
		BaseURL:   CatalogURL,
		App:       App,
		PublicKey: PublicKey,
		Token:     os.Getenv(TokenEnv),
	})
}

// NewWithConfig creates a Service with an explicit catalog configuration.
func NewWithConfig(cfg updater.Config) *Service { return &Service{cfg: cfg} }

// releaseVersion rejects development builds, which have no comparable version.
func releaseVersion(current string) error {
	if strings.Contains(current, "SNAPSHOT") {
		return fmt.Errorf("%w (version %q)", ErrDevBuild, current)
	}
	if _, err := updater.CompareVersions(current, current); err != nil {
		return fmt.Errorf("%w (version %q)", ErrDevBuild, current)
	}
	return nil
}

func (s *Service) latest(ctx context.Context, current string) (*updater.Release, error) {
	if err := releaseVersion(current); err != nil {
		return nil, err
	}
	if s.cfg.Token == "" {
		return nil, ErrNoToken
	}
	cfg := s.cfg
	cfg.CurrentVersion = current
	rel, err := updater.Check(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("checking for updates: %w", err)
	}
	return rel, nil
}

// Check returns the version to install and whether it differs from current.
// When there is nothing to install it returns current.
func (s *Service) Check(ctx context.Context, current string) (latest string, newer bool, err error) {
	rel, err := s.latest(ctx, current)
	if err != nil {
		return "", false, err
	}
	if rel == nil {
		return trimV(current), false, nil
	}
	return rel.Version, true, nil
}

// Apply replaces the executable at exePath with the release published in the
// catalog when it is newer than current (or forced by the owner). It returns
// the resulting version and whether the executable was replaced.
func (s *Service) Apply(ctx context.Context, current, exePath string) (version string, updated bool, err error) {
	rel, err := s.latest(ctx, current)
	if err != nil {
		return "", false, err
	}
	if rel == nil {
		return trimV(current), false, nil
	}
	cfg := s.cfg
	cfg.CurrentVersion = current
	path, err := updater.Download(ctx, cfg, rel, filepath.Dir(exePath))
	if err != nil {
		return rel.Version, false, fmt.Errorf("downloading %s: %w", rel.Version, err)
	}
	if err := updater.ApplyTo(path, exePath); err != nil {
		_ = os.Remove(path)
		return rel.Version, false, fmt.Errorf("installing %s: %w", rel.Version, err)
	}
	return rel.Version, true, nil
}

func trimV(v string) string { return strings.TrimPrefix(v, "v") }
