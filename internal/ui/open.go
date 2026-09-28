package ui

import (
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// openFile launches a path in the OS default application. It indirects to the
// platform osOpen (osopen_{darwin,linux}.go) through a var so tests can stub it
// without actually spawning anything.
var openFile = osOpen

// opFailedMsg reports a file operation that failed off the UI goroutine; Update
// shows it as a toast (tdp F5).
type opFailedMsg struct{ text string }

// opFailedText is the toast line for a failed operation: what was being done,
// then the bare reason — the OS error without the path it would repeat.
func opFailedText(what string, err error) string {
	var pe *fs.PathError
	var le *os.LinkError
	switch {
	case errors.As(err, &pe):
		err = pe.Err
	case errors.As(err, &le):
		err = le.Err
	}
	return "Cannot " + what + ": " + err.Error()
}

// openFileCmd opens path off the UI goroutine; the launcher exits quickly, so
// the goroutine is short-lived. A failure comes back as an opFailedMsg.
func openFileCmd(path string) tea.Cmd {
	return func() tea.Msg {
		if err := openFile(path); err != nil {
			return opFailedMsg{opFailedText("open "+filepath.Base(path), err)}
		}
		return nil
	}
}

// openWithCmd launches `cmd path` off the UI goroutine — the [o]pen picker's
// action for a configured app. cmd may carry args (e.g. "code -n"); path is
// appended as the last argument. GUI editors fork and return, so we Start and
// don't wait; a command that cannot start comes back as an opFailedMsg.
func openWithCmd(cmd, path string) tea.Cmd {
	return func() tea.Msg {
		fields := strings.Fields(cmd)
		if len(fields) == 0 {
			return nil
		}
		c := exec.Command(fields[0], append(fields[1:], path)...)
		if err := c.Start(); err != nil {
			return opFailedMsg{opFailedText("run "+fields[0], err)}
		}
		return nil
	}
}
