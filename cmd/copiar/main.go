// Command copiar is a dual-pane, Vim-style TUI to copy files between folders.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/signal"
	"runtime"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/creativeprojects/go-selfupdate"

	"github.com/RHAM91/copy-tui/internal/bookmarks"
	"github.com/RHAM91/copy-tui/internal/ui"
	"github.com/RHAM91/copy-tui/internal/update"
)

// version is set at build time with -ldflags "-X main.version=...".
var version = "dev"

type config struct {
	ui          ui.Options
	showVersion bool
	// command is "" to open the UI, or "update".
	command string
}

func parseArgs(args []string) (config, error) {
	fs := flag.NewFlagSet("copiar", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	noIcons := fs.Bool("no-icons", false, "disable Nerd Font icons")
	showVersion := fs.Bool("version", false, "print the version and exit")
	if err := fs.Parse(args); err != nil {
		return config{}, err
	}
	cfg := config{ui: ui.Options{Icons: !*noIcons}, showVersion: *showVersion}
	switch rest := fs.Args(); {
	case len(rest) == 0:
	case rest[0] == "update" && len(rest) == 1:
		cfg.command = "update"
	case rest[0] == "update":
		return config{}, fmt.Errorf("update takes no arguments")
	default:
		return config{}, fmt.Errorf("unknown command %q", rest[0])
	}
	return cfg, nil
}

func versionString() string { return "copiar " + version }

// updater is the part of update.Service used here, so tests can fake it.
type updater interface {
	Check(ctx context.Context, current string) (latest string, newer bool, err error)
	Apply(ctx context.Context, current, exePath string) (version string, updated bool, err error)
}

// runUpdate replaces the executable with the latest release and returns the
// process exit code.
func runUpdate(ctx context.Context, up updater, current, exePath string, out io.Writer) int {
	fmt.Fprintln(out, "Checking for updates…")
	latest, updated, err := up.Apply(ctx, current, exePath)
	switch {
	case errors.Is(err, update.ErrDevBuild):
		fmt.Fprintf(out, "copiar update: %v\n", err)
		fmt.Fprintf(out, "Download a release from https://github.com/%s/releases\n", update.Repository)
		return 1
	case errors.Is(err, fs.ErrPermission):
		fmt.Fprintf(out, "copiar update: cannot replace %s: %v\n", exePath, err)
		fmt.Fprintln(out, permissionHint(runtime.GOOS))
		return 1
	case err != nil:
		fmt.Fprintf(out, "copiar update: %v\n", err)
		return 1
	case !updated:
		fmt.Fprintf(out, "copiar is up to date (v%s)\n", latest)
	default:
		fmt.Fprintf(out, "Updated copiar v%s → v%s\n", trimV(current), latest)
	}
	return 0
}

// startupCheck returns the background check shown in the UI, or nil for
// development builds, which have no version to compare.
func startupCheck(up updater, current string) func(context.Context) (string, error) {
	if current == "dev" {
		return nil
	}
	return func(ctx context.Context) (string, error) {
		latest, newer, err := up.Check(ctx, current)
		if err != nil || !newer {
			return "", err
		}
		return latest, nil
	}
}

// permissionHint tells how to retry when the executable is not writable.
func permissionHint(goos string) string {
	if goos == "windows" {
		return "Open the terminal as Administrator and run: copiar update"
	}
	return "Try again with: sudo copiar update"
}

func trimV(v string) string {
	if len(v) > 0 && v[0] == 'v' {
		return v[1:]
	}
	return v
}

const usage = `Usage: copiar [--no-icons] [--version]
       copiar update

Dual-pane file manager with Vim keys. Both panels open in the current directory.

Commands:
  update       download and install the latest release (checksum verified)

Flags:
  --no-icons   disable Nerd Font icons (use when your terminal font lacks them)
  --version    print the version and exit
`

func main() {
	cfg, err := parseArgs(os.Args[1:])
	if err != nil {
		if err != flag.ErrHelp {
			fmt.Fprintf(os.Stderr, "copiar: %v\n\n", err)
		}
		fmt.Fprint(os.Stderr, usage)
		if err == flag.ErrHelp {
			os.Exit(0)
		}
		os.Exit(2)
	}

	if cfg.showVersion {
		fmt.Println(versionString())
		return
	}

	svc, err := update.New(nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "copiar: %v\n", err)
		os.Exit(1)
	}

	if cfg.command == "update" {
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
		exe, err := selfupdate.ExecutablePath()
		if err != nil {
			stop()
			fmt.Fprintf(os.Stderr, "copiar update: cannot locate the executable: %v\n", err)
			os.Exit(1)
		}
		code := runUpdate(ctx, svc, version, exe, os.Stdout)
		stop()
		os.Exit(code)
	}

	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "copiar: cannot read current directory: %v\n", err)
		os.Exit(1)
	}

	cfg.ui.CheckUpdate = startupCheck(svc, version)
	// Without a config directory bookmarks stay disabled; ctrl+s and ctrl+b
	// then explain why in the status bar.
	if path, err := bookmarks.DefaultPath(); err == nil {
		cfg.ui.Bookmarks = bookmarks.NewStore(path)
	}
	p := tea.NewProgram(ui.New(cwd, cfg.ui), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "copiar: %v\n", err)
		os.Exit(1)
	}
}
