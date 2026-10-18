package ui

import (
	"io"
	"os"

	osc52 "github.com/aymanbagabas/go-osc52/v2"
	tea "github.com/charmbracelet/bubbletea"
)

// clipboardCopiedMsg / clipboardFailedMsg report a yank's result; the app turns
// a success into a toast.
type clipboardCopiedMsg struct{ note string }
type clipboardFailedMsg struct{}

// clipboardOut is where the OSC 52 sequence goes: stderr, the terminal, since
// Bubble Tea owns stdout; a variable so that tests can read it.
var clipboardOut io.Writer = os.Stderr

// copyToClipboardCmd writes text to the system clipboard via OSC 52, which works
// over SSH and inside tmux without a local clipboard tool. note is the toast
// shown on success.
func copyToClipboardCmd(text, note string) tea.Cmd {
	return func() tea.Msg {
		if text == "" {
			return clipboardFailedMsg{}
		}
		seq := osc52.New(text)
		if os.Getenv("TMUX") != "" {
			seq = seq.Tmux()
		}
		if _, err := seq.WriteTo(clipboardOut); err != nil {
			return clipboardFailedMsg{}
		}
		return clipboardCopiedMsg{note: note}
	}
}
