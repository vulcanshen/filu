package ui

import (
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
