// Command filu is a terminal file manager (terminu family).
package main

import (
	"fmt"
	"os"
	"runtime"

	terminu "github.com/vulcanshen/terminu-go"

	"github.com/vulcanshen/filu/internal/ui"
	"github.com/vulcanshen/filu/internal/version"
)

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "version", "--version", "-v":
			fmt.Println("filu " + version.Display()) // "filu v0.1.0" (a release) or "filu dev"
			return
		case "help", "--help", "-h":
			fmt.Print(usage)
			return
		case "shell": // cd-on-quit (shell.go)
			home, _ := os.UserHomeDir()
			fi, err := os.Stdout.Stat()
			tty := err == nil && fi.Mode()&os.ModeCharDevice != 0
			os.Exit(shellCmd(os.Args[2:], shellEnv{os.Getenv("SHELL"), home, os.Getenv("ZDOTDIR"), runtime.GOOS, tty}, os.Stdout, os.Stderr))
		case "iconwidth":
			r := terminu.Probe("filu") // what filu measures at startup (tdp E5)
			fmt.Printf("icon %d (%s)\npowerline %d (%s)\n", r.IconCells, r.IconFrom, r.PowerlineCells, r.PowerlineFrom)
			return
		}
	}
	var startDir, focusName string
	if len(os.Args) > 1 { // `filu <path>`: open there (a file: its directory, the cursor on it)
		dir, focus, err := ui.ResolveStartDir(os.Args[1])
		if err != nil {
			fmt.Fprintln(os.Stderr, "filu:", err)
			os.Exit(1)
		}
		startDir, focusName = dir, focus
	}
	if err := ui.NewApp(startDir, focusName).Run(); err != nil {
		fmt.Fprintln(os.Stderr, "filu:", err)
		os.Exit(1)
	}
}

// usage is what `filu help` prints (tdp E1); the keys are in the app, under ?.
const usage = `filu — a terminal file manager you don't have to learn

Usage:
  filu              open the current directory
  filu <directory>  open that directory
  filu <file>       open its directory, the cursor on it
  filu shell install
                    let quitting filu change your shell's directory: adds a
                    few lines to your zsh or bash rc file
  filu shell uninstall
                    take them out again
  filu iconwidth    print how many cells an icon and a powerline symbol take
  filu version      print the version (also --version, -v)
  filu help         print this (also --help, -h)

In filu, Space lists what you can do and ? lists the keys.

Environment:
  FILU__CONFIG               the settings directory (config.yaml, state.yaml)
  FILU__STATE                the directory of state.yaml, when not the settings one
  FILU__DATA                 the data directory: data.yaml (your favorites and the task
                             log), open_with.yaml and ignore_dirs.yaml
  FILU__ICON_WIDTH           1 or 2: the cells a Nerd Font icon takes, instead of asking
                             the terminal
  FILU__POWERLINE_WIDTH      1 or 2: the same for a powerline symbol
  TERMINU__ICON_WIDTH        the same as FILU__ICON_WIDTH, for every terminu family app
  TERMINU__POWERLINE_WIDTH   the same as FILU__POWERLINE_WIDTH, for every terminu family app
`
