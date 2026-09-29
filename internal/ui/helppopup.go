package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// helpPopup is the ? key reference (tdp K6, M4): the keys the frontmost surface
// answers to — a panel, or the popup on top. Read only: it scrolls, but it has
// no cursor and nothing in it runs. ? or Esc closes it, back to what it
// describes (F4). One instance serves the panels and the popups; a second
// (quitHelp) serves the quit picker, which sits over everything else.
type helpPopup struct {
	anim    popupAnimator
	title   string
	rows    []helpRow
	top     int // first row shown when the list is taller than the screen
	screenW int
	screenH int
}

type helpRow struct {
	key, desc string
	header    bool
}

func newHelpPopup() helpPopup {
	return helpPopup{anim: newPopupAnimator("help", popupLayerColor(1))}
}

// newQuitHelp is the key reference of the quit picker: its own instance, so it
// can sit over the quit picker while the other one may sit under it.
func newQuitHelp() helpPopup {
	return helpPopup{anim: newPopupAnimator("quithelp", popupLayerColor(1))}
}

// open shows rows under title, scrolled to the top.
func (m *helpPopup) open(title string, rows []helpRow) tea.Cmd {
	m.title, m.rows, m.top = title, rows, 0
	return m.anim.open()
}

func (m *helpPopup) setSize(w, h int) { m.screenW, m.screenH = w, h }
func (m helpPopup) isActive() bool    { return m.anim.isActive() }
func (m helpPopup) owns() bool        { return m.anim.owns() }
func (m helpPopup) isInteractive() bool {
	return m.anim.isInteractive()
}
func (m *helpPopup) handleTick(msg AnimTickMsg) tea.Cmd {
	if msg.Target != m.anim.target {
		return nil
	}
	return m.anim.tick()
}

// visible is how many rows fit on screen (all of them when the height is not
// known yet): the screen minus the box chrome and a row of margin each side.
func (m helpPopup) visible() int {
	if m.screenH <= 0 {
		return max(len(m.rows), 1)
	}
	return max(m.screenH-menuChrome, 3)
}

// scrollBy moves the window by n rows, clamped to the list.
func (m *helpPopup) scrollBy(n int) {
	m.top = max(0, min(m.top+n, len(m.rows)-m.visible()))
}

func (m helpPopup) update(msg tea.KeyMsg) (helpPopup, tea.Cmd) {
	if !m.anim.isInteractive() {
		return m, nil
	}
	switch msg.String() {
	case "esc", "?": // q is the leave flow (tdp K9); Space only toggles the Space menu (K5)
		return m, m.anim.close()
	case "j", "down":
		m.scrollBy(1)
	case "k", "up":
		m.scrollBy(-1)
	case "d", "ctrl+d":
		m.scrollBy(max(m.visible()/2, 1))
	case "u", "ctrl+u":
		m.scrollBy(-max(m.visible()/2, 1))
	case "g":
		m.top = 0
	case "G":
		m.scrollBy(len(m.rows))
	}
	return m, nil
}

func (m helpPopup) renderPopup() string { return m.anim.renderFrame(m.renderFull()) }

// helpHint is the key reference's bottom border: only how to move and leave.
func helpHint() string { return keyLegend([][2]string{{"j/k", "scroll"}, {"?/Esc", "close"}}) }

// keyRefDesc is a key reference's description colour, Text (tdp D2); the keys
// are Blue, like the hints'.
const keyRefDesc = lipgloss.Color("#cdd6f4")

func (m helpPopup) renderFull() string {
	bc := popupLayerColor(m.anim.layer)
	keyStyle := lipgloss.NewStyle().Foreground(focusColor).Bold(true)
	descStyle := lipgloss.NewStyle().Foreground(keyRefDesc)
	headerStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#7f849c"))

	title := " " + string(rune(0xf059)) + " " + m.title // nf-fa-question-circle

	keyW := 0
	for _, r := range m.rows {
		if !r.header {
			keyW = max(keyW, lipgloss.Width(r.key))
		}
	}
	// The family width (tdp F7, which replaces D4's "as wide as the longest
	// description"); a description longer than the room left is cut.
	innerW := popupInnerWidth(m.screenW)

	rows := make([]string, 0, len(m.rows))
	for _, r := range m.rows {
		if r.header {
			rows = append(rows, " "+headerStyle.Render(truncate(r.desc, innerW-2)))
			continue
		}
		key := r.key + strings.Repeat(" ", max(0, keyW-lipgloss.Width(r.key)))
		desc := truncate(r.desc, max(innerW-(2+keyW+2)-1, 1))
		rows = append(rows, "  "+keyStyle.Render(key)+"  "+descStyle.Render(desc))
	}
	if vis := m.visible(); len(rows) > vis {
		top := max(0, min(m.top, len(rows)-vis))
		rows = rows[top : top+vis]
	}
	return drawPopupBox(bc, title, helpHint(), rows, innerW)
}
