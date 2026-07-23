package ui

import (
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

// openFile launches a path in the OS default application. It indirects to the
// platform osOpen (osopen_{darwin,linux}.go) through a var so tests can stub it
// without actually spawning anything.
var openFile = osOpen

// isTextFile reports whether path looks like editable text (valid UTF-8, no NUL
// bytes) — the gate for opening it in the embedded editor vs handing it to the
// OS.
func isTextFile(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()
	buf := make([]byte, 512)
	n, _ := f.Read(buf)
	return isText(buf[:n])
}

// openFileCmd opens path off the UI goroutine. Errors are dropped for now (a
// failure toast can come later); the launcher exits quickly, so the goroutine
// is short-lived.
func openFileCmd(path string) tea.Cmd {
	return func() tea.Msg {
		_ = openFile(path)
		return nil
	}
}
