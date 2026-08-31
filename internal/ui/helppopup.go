package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// helpPopup is the §A.2 non-contextual entry: a global key cheatsheet opened by
// `?`. Informational (not a launcher) — any of esc/?/Space/q dismisses it.
type helpPopup struct {
	anim    popupAnimator
	screenW int
}

type helpRow struct {
	key, desc string
	header    bool
}

// helpRows is the global cheatsheet. Contextual verbs live in the Space menu;
// this lists only the app-wide core keys and navigation.
var helpRows = []helpRow{
	{header: true, desc: "panels"},
	{key: "Tab", desc: "focus the next panel"},
	{key: "1 2 3", desc: "focus a panel directly"},
	{key: "h l", desc: "switch the focused panel's tab"},
	{header: true, desc: "move"},
	{key: "j k", desc: "down / up"},
	{key: "g G", desc: "top / bottom"},
	{key: "u d", desc: "half page up / down"},
	{header: true, desc: "do"},
	{key: "Enter", desc: "enter a directory (o opens a file)"},
	{key: "Esc", desc: "back / up a directory"},
	{key: "Space", desc: "actions menu for this panel"},
	{key: "z", desc: "zoom the focused panel"},
	{key: "?", desc: "this help"},
	{key: "q", desc: "quit — pick a dir to cd to"},
}

func newHelpPopup() helpPopup {
	return helpPopup{anim: newPopupAnimator("help", popupLayerColor(1))}
}

func (m *helpPopup) open() tea.Cmd      { return m.anim.open() }
func (m *helpPopup) setSize(w int)      { m.screenW = w }
func (m helpPopup) isActive() bool      { return m.anim.isActive() }
func (m helpPopup) isInteractive() bool { return m.anim.isInteractive() }
func (m *helpPopup) handleTick(msg AnimTickMsg) tea.Cmd {
	if msg.Target != m.anim.target {
		return nil
	}
	return m.anim.tick()
}

func (m helpPopup) update(msg tea.KeyMsg) (helpPopup, tea.Cmd) {
	if !m.anim.isInteractive() {
		return m, nil
	}
	switch msg.String() {
	case "esc", "?", " ", "q":
		return m, m.anim.close()
	}
	return m, nil
}

func (m helpPopup) renderPopup() string { return m.anim.renderFrame(m.renderFull()) }

func (m helpPopup) renderFull() string {
	bc := popupLayerColor(1)
	keyStyle := lipgloss.NewStyle().Foreground(bc).Bold(true)
	descStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#7f849c"))

	title := " " + string(rune(0xf059)) + " Help" // nf-fa-question-circle
	hint := " esc close "

	const keyW = 8
	innerW := max(lipgloss.Width(title)+4, lipgloss.Width(hint)+4)
	for _, r := range helpRows {
		if w := 2 + keyW + 1 + lipgloss.Width(r.desc) + 1; w > innerW {
			innerW = w
		}
	}
	innerW = min(innerW, maxInnerWidth(m.screenW))

	var rows []string
	for _, r := range helpRows {
		if r.header {
			rows = append(rows, " "+descStyle.Render(r.desc))
			continue
		}
		key := r.key + strings.Repeat(" ", max(0, keyW-lipgloss.Width(r.key)))
		rows = append(rows, "  "+keyStyle.Render(key)+" "+descStyle.Render(r.desc))
	}
	return drawPopupBox(bc, title, hint, rows, innerW)
}
