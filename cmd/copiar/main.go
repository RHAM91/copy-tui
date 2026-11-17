// Command copiar is a dual-pane, Vim-style TUI to copy files between folders.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/RHAM91/copy-tui/internal/ui"
)

// version is set at build time with -ldflags "-X main.version=...".
var version = "dev"

type config struct {
	ui          ui.Options
	showVersion bool
}

func parseArgs(args []string) (config, error) {
	fs := flag.NewFlagSet("copiar", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	noIcons := fs.Bool("no-icons", false, "disable Nerd Font icons")
	showVersion := fs.Bool("version", false, "print the version and exit")
	if err := fs.Parse(args); err != nil {
		return config{}, err
	}
	return config{ui: ui.Options{Icons: !*noIcons}, showVersion: *showVersion}, nil
}

func versionString() string { return "copiar " + version }

const usage = `Usage: copiar [--no-icons] [--version]

Dual-pane file manager with Vim keys. Both panels open in the current directory.

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

	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "copiar: cannot read current directory: %v\n", err)
		os.Exit(1)
	}

	p := tea.NewProgram(ui.New(cwd, cfg.ui), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "copiar: %v\n", err)
		os.Exit(1)
	}
}
