package ui

import (
	"os"

	osc52 "github.com/aymanbagabas/go-osc52/v2"
	tea "github.com/charmbracelet/bubbletea"
)

// clipboardCopiedMsg / clipboardFailedMsg report a yank's result; the app turns
// a success into a toast.
type clipboardCopiedMsg struct{ note string }
type clipboardFailedMsg struct{}

// copyToClipboardCmd writes text to the system clipboard via OSC 52, which works
// over SSH and inside tmux without a local clipboard tool. The escape is written
// to stderr because Bubble Tea owns stdout in alt-screen mode. note is the toast
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
		if _, err := seq.WriteTo(os.Stderr); err != nil {
			return clipboardFailedMsg{}
		}
		return clipboardCopiedMsg{note: note}
	}
}
