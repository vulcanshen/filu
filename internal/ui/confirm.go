package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// confirmPopup is a yes/no confirmation, following kbu's confirm form: a bold
// message, Enter/y to confirm, Esc/n/Space to cancel.
type confirmPopup struct {
	anim    popupAnimator
	message string
	screenW int
}

func newConfirmPopup() confirmPopup {
	return confirmPopup{anim: newPopupAnimator("confirm", popupLayerColor(1))}
}

func (m *confirmPopup) open(message string) tea.Cmd {
	m.message = message
	return m.anim.open()
}

func (m *confirmPopup) close() tea.Cmd     { return m.anim.close() }
func (m *confirmPopup) setSize(w int)      { m.screenW = w }
func (m confirmPopup) isActive() bool      { return m.anim.isActive() }
func (m confirmPopup) isInteractive() bool { return m.anim.isInteractive() }

func (m *confirmPopup) handleTick(msg AnimTickMsg) tea.Cmd {
	if msg.Target != m.anim.target {
		return nil
	}
	return m.anim.tick()
}

// update handles a keystroke while interactive. confirmed is true when the user
// accepts (Enter/y); the caller then performs the action and the popup closes.
func (m confirmPopup) update(msg tea.KeyMsg) (confirmPopup, bool, tea.Cmd) {
	if !m.anim.isInteractive() {
		return m, false, nil
	}
	switch msg.String() {
	case "enter", "y":
		return m, true, m.anim.close()
	case "esc", "n", " ":
		return m, false, m.anim.close()
	}
	return m, false, nil
}

func (m confirmPopup) renderPopup() string { return m.anim.renderFrame(m.renderFull()) }

func (m confirmPopup) renderFull() string {
	bc := popupLayerColor(1)
	title := " " + string(rune(0xf071)) + " Confirm" // nf-fa-warning
	hint := " enter/y confirm   esc cancel "

	innerW := max(lipgloss.Width(title)+4, lipgloss.Width(hint)+4)
	innerW = min(max(innerW, lipgloss.Width(m.message)+4), maxInnerWidth(m.screenW))

	msgStyle := lipgloss.NewStyle().Bold(true)
	var rows []string
	for _, l := range wrapWords(m.message, innerW-2) {
		rows = append(rows, msgStyle.Render(" "+l))
	}
	return drawPopupBox(bc, title, hint, rows, innerW)
}

// wrapWords soft-wraps s to at most w columns on word boundaries.
func wrapWords(s string, w int) []string {
	if w < 1 {
		return []string{s}
	}
	var lines []string
	var line strings.Builder
	for word := range strings.FieldsSeq(s) {
		switch {
		case line.Len() == 0:
			line.WriteString(word)
		case line.Len()+1+len(word) <= w:
			line.WriteByte(' ')
			line.WriteString(word)
		default:
			lines = append(lines, line.String())
			line.Reset()
			line.WriteString(word)
		}
	}
	if line.Len() > 0 {
		lines = append(lines, line.String())
	}
	if len(lines) == 0 {
		return []string{""}
	}
	return lines
}
