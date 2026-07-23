package ui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// inputPopup is a single-line text prompt (rename / add). filu-specific — kbu has
// no text-entry popup — but drawn in kbu's popup form (animator, layer colour,
// box). Replaces the old footer input.
type inputPopup struct {
	anim    popupAnimator
	kind    inputKind
	prompt  string
	buffer  string
	target  string // original name (rename)
	screenW int
}

func newInputPopup() inputPopup {
	return inputPopup{anim: newPopupAnimator("input", popupLayerColor(1))}
}

func (m *inputPopup) open(kind inputKind, prompt, buffer, target string) tea.Cmd {
	m.kind, m.prompt, m.buffer, m.target = kind, prompt, buffer, target
	return m.anim.open()
}

func (m *inputPopup) close() tea.Cmd     { return m.anim.close() }
func (m *inputPopup) setSize(w int)      { m.screenW = w }
func (m inputPopup) isActive() bool      { return m.anim.isActive() }
func (m inputPopup) isInteractive() bool { return m.anim.isInteractive() }
func (m *inputPopup) handleTick(msg AnimTickMsg) tea.Cmd {
	if msg.Target != m.anim.target {
		return nil
	}
	return m.anim.tick()
}

// update edits the buffer. committed is true on Enter (the caller performs the
// rename/add from kind/buffer/target); Esc cancels.
func (m inputPopup) update(msg tea.KeyMsg) (inputPopup, bool, tea.Cmd) {
	if !m.anim.isInteractive() {
		return m, false, nil
	}
	switch msg.Type {
	case tea.KeyEsc:
		return m, false, m.anim.close()
	case tea.KeyEnter:
		return m, true, m.anim.close()
	case tea.KeyBackspace:
		if r := []rune(m.buffer); len(r) > 0 {
			m.buffer = string(r[:len(r)-1])
		}
	case tea.KeySpace:
		m.buffer += " "
	case tea.KeyRunes:
		m.buffer += string(msg.Runes)
	}
	return m, false, nil
}

func (m inputPopup) renderPopup() string { return m.anim.renderFrame(m.renderFull()) }

func (m inputPopup) renderFull() string {
	bc := popupLayerColor(1)
	title := " " + m.prompt
	hint := " enter confirm   esc cancel "

	field := m.buffer + "█"
	innerW := max(lipgloss.Width(title)+4, lipgloss.Width(hint)+4)
	innerW = max(innerW, lipgloss.Width(m.target)+4)
	innerW = min(max(innerW, lipgloss.Width(field)+4), maxInnerWidth(m.screenW))

	// keep the cursor (tail) visible when the text overruns the box
	if lipgloss.Width(field) > innerW {
		field = ansi.TruncateLeft(field, lipgloss.Width(field)-(innerW-1), "…")
	}
	var rows []string
	if m.target != "" { // a description line naming what's being renamed
		rows = append(rows, " "+lipgloss.NewStyle().Foreground(dimColor).Render(m.target))
	}
	// full-width dark-grey input bar (catppuccin surface1); the text starts at
	// the left edge — no untouchable leading space the cursor can't reach.
	bar := lipgloss.NewStyle().Background(lipgloss.Color("#45475a")).Width(innerW).Render(field)
	rows = append(rows, bar)
	return drawPopupBox(bc, title, hint, rows, innerW)
}
