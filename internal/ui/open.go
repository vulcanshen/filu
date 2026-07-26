package ui

import (
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// openFile launches a path in the OS default application. It indirects to the
// platform osOpen (osopen_{darwin,linux}.go) through a var so tests can stub it
// without actually spawning anything.
var openFile = osOpen

// openFileCmd opens path off the UI goroutine. Errors are dropped for now (a
// failure toast can come later); the launcher exits quickly, so the goroutine
// is short-lived.
func openFileCmd(path string) tea.Cmd {
	return func() tea.Msg {
		_ = openFile(path)
		return nil
	}
}

// openWithCmd launches `cmd path` off the UI goroutine — the [o]pen picker's
// action for a configured app. cmd may carry args (e.g. "code -n"); path is
// appended as the last argument. Fire-and-forget: GUI editors fork and return,
// so we Start and don't wait, and (like openFileCmd) errors are dropped.
func openWithCmd(cmd, path string) tea.Cmd {
	return func() tea.Msg {
		fields := strings.Fields(cmd)
		if len(fields) == 0 {
			return nil
		}
		c := exec.Command(fields[0], append(fields[1:], path)...)
		_ = c.Start()
		return nil
	}
}
