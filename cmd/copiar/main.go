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

func parseArgs(args []string) (ui.Options, error) {
	fs := flag.NewFlagSet("copiar", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	noIcons := fs.Bool("no-icons", false, "disable Nerd Font icons")
	if err := fs.Parse(args); err != nil {
		return ui.Options{}, err
	}
	return ui.Options{Icons: !*noIcons}, nil
}

const usage = `Usage: copiar [--no-icons]

Dual-pane file manager with Vim keys. Both panels open in the current directory.

Flags:
  --no-icons   disable Nerd Font icons (use when your terminal font lacks them)
`

func main() {
	opts, err := parseArgs(os.Args[1:])
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

	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "copiar: cannot read current directory: %v\n", err)
		os.Exit(1)
	}

	p := tea.NewProgram(ui.New(cwd, opts), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "copiar: %v\n", err)
		os.Exit(1)
	}
}
