package ui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// toastModel is a transient notification (kbu form): a small popup that opens on
// an event and auto-dismisses after a short delay. Body text only — no wide
// glyphs — so it can't disturb the popup border where icons take two cells.
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
func (m *toastModel) show(message string) tea.Cmd { return m.showFor(message, 1500*time.Millisecond) }

// showError displays a failure long enough to read (tdp F5); Esc still closes
// it at once, and it never blocks the app.
func (m *toastModel) showError(message string) tea.Cmd { return m.showFor(message, 4*time.Second) }

func (m *toastModel) showFor(message string, d time.Duration) tea.Cmd {
	m.message = message
	m.id++
	id := m.id
	dismiss := tea.Tick(d, func(time.Time) tea.Msg { return toastDismissMsg{id: id} })
	return tea.Batch(m.anim.open(), dismiss)
}

func (m *toastModel) setSize(w int) { m.screenW = w }
func (m toastModel) isActive() bool { return m.anim.isActive() }
func (m toastModel) owns() bool     { return m.anim.owns() }

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

// closeNow starts closing the toast right away (Esc, tdp F3); the pending
// auto-dismiss tick then finds it already closing and does nothing.
func (m *toastModel) closeNow() tea.Cmd { return m.anim.close() }

func (m toastModel) renderPopup() string { return m.anim.renderFrame(m.renderFull()) }

func (m toastModel) renderFull() string {
	bc := popupLayerColor(1)
	body := " " + m.message + " "
	innerW := popupInnerWidth(m.screenW) // the family width (tdp F7)
	return drawPopupBox(bc, " filu", " ", []string{truncate(body, innerW)}, innerW)
}
