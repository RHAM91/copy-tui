// Package update checks GitHub releases for a newer copiar and replaces the
// running executable with it, after verifying the release checksums.
package update

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/creativeprojects/go-selfupdate"
)

// Repository is the GitHub repository whose releases are used.
const Repository = "RHAM91/copy-tui"

// checksumsFile is the checksum list GoReleaser attaches to every release.
const checksumsFile = "checksums.txt"

var (
	// ErrDevBuild is returned for builds without a release version (local
	// "dev" or GoReleaser snapshot builds), which cannot be compared.
	ErrDevBuild = errors.New("development build: install a release to enable updates")
	// ErrNoRelease is returned when no release exists for this platform.
	ErrNoRelease = errors.New("no release found for this platform")
)

// Service checks for and applies updates.
type Service struct {
	updater *selfupdate.Updater
	repo    selfupdate.Repository
}

// New creates a Service reading releases from source; a nil source uses the
// public GitHub API.
func New(source selfupdate.Source) (*Service, error) {
	if source == nil {
		gh, err := selfupdate.NewGitHubSource(selfupdate.GitHubConfig{})
		if err != nil {
			return nil, err
		}
		source = gh
	}
	up, err := selfupdate.NewUpdater(selfupdate.Config{
		Source: source,
		// Every download is verified against checksums.txt; a release
		// without it, or with a mismatching sum, is rejected.
		Validator: &selfupdate.ChecksumValidator{UniqueFilename: checksumsFile},
	})
	if err != nil {
		return nil, err
	}
	return &Service{updater: up, repo: selfupdate.ParseSlug(Repository)}, nil
}

// releaseVersion parses current, rejecting development builds.
func releaseVersion(current string) (*semver.Version, error) {
	v, err := semver.NewVersion(strings.TrimPrefix(current, "v"))
	if err != nil || strings.HasPrefix(v.Prerelease(), "SNAPSHOT") {
		return nil, fmt.Errorf("%w (version %q)", ErrDevBuild, current)
	}
	return v, nil
}

func (s *Service) latest(ctx context.Context, current string) (*selfupdate.Release, bool, error) {
	cur, err := releaseVersion(current)
	if err != nil {
		return nil, false, err
	}
	rel, found, err := s.updater.DetectLatest(ctx, s.repo)
	if err != nil {
		return nil, false, fmt.Errorf("checking for updates: %w", err)
	}
	if !found {
		return nil, false, ErrNoRelease
	}
	latest, err := semver.NewVersion(rel.Version())
	if err != nil {
		return nil, false, fmt.Errorf("invalid release version %q: %w", rel.Version(), err)
	}
	return rel, latest.GreaterThan(cur), nil
}

// Check returns the latest released version and whether it is newer than
// current.
func (s *Service) Check(ctx context.Context, current string) (latest string, newer bool, err error) {
	rel, newer, err := s.latest(ctx, current)
	if err != nil {
		return "", false, err
	}
	return rel.Version(), newer, nil
}

// Apply replaces the executable at exePath with the latest release when it
// is newer than current. It returns the latest version and whether the
// executable was replaced.
func (s *Service) Apply(ctx context.Context, current, exePath string) (version string, updated bool, err error) {
	rel, newer, err := s.latest(ctx, current)
	if err != nil {
		return "", false, err
	}
	if !newer {
		return rel.Version(), false, nil
	}
	if err := s.updater.UpdateTo(ctx, rel, exePath); err != nil {
		return rel.Version(), false, fmt.Errorf("installing %s: %w", rel.Version(), err)
	}
	return rel.Version(), true, nil
}
