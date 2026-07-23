package ui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// toastModel is a transient notification (kbu form): a small popup that opens on
// an event and auto-dismisses after a short delay. Body text only — no wide
// glyphs — so it can't disturb the popup border on CJK icon fonts.
type toastModel struct {
	anim    popupAnimator
	message string
	id      int // generation counter; a stale dismiss tick from a superseded toast is ignored
	screenW int
}

type toastDismissMsg struct{ id int }

func newToast() toastModel {
	return toastModel{anim: newPopupAnimator("toast", popupLayerColor(1))}
}

// show displays message and schedules its auto-dismiss.
func (m *toastModel) show(message string) tea.Cmd {
	m.message = message
	m.id++
	id := m.id
	dismiss := tea.Tick(1500*time.Millisecond, func(time.Time) tea.Msg { return toastDismissMsg{id: id} })
	return tea.Batch(m.anim.open(), dismiss)
}

func (m *toastModel) setSize(w int) { m.screenW = w }
func (m toastModel) isActive() bool { return m.anim.isActive() }

func (m *toastModel) handleTick(msg AnimTickMsg) tea.Cmd {
	if msg.Target != m.anim.target {
		return nil
	}
	return m.anim.tick()
}

// dismiss closes the toast when the tick matches the current generation (a newer
// toast bumps id, so the older tick is a no-op).
func (m *toastModel) dismiss(msg toastDismissMsg) tea.Cmd {
	if msg.id != m.id {
		return nil
	}
	return m.anim.close()
}

func (m toastModel) renderPopup() string { return m.anim.renderFrame(m.renderFull()) }

func (m toastModel) renderFull() string {
	bc := popupLayerColor(1)
	body := " " + m.message + " "
	innerW := min(max(lipgloss.Width(body), lipgloss.Width(" filu")+2), maxInnerWidth(m.screenW))
	return drawPopupBox(bc, " filu", " ", []string{truncate(body, innerW)}, innerW)
}
