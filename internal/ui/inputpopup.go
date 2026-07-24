package ui

import (
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// inputGlyph (nf-fa-chevron_right) marks where the user types — a peach prompt
// chevron shown before every text-entry field (rename / add / search) so an
// input is instantly legible as one. The typed text begins right after it.
var inputGlyph = string(rune(0xf054))

// inputBlinkMsg toggles the input cursor; gen keeps one blink loop per open.
type inputBlinkMsg struct{ gen int }

// inputPopup is a single-line text prompt (rename / add). filu-specific — kbu has
// no text-entry popup — but drawn in kbu's popup form (animator, layer colour,
// box).
type inputPopup struct {
	anim     popupAnimator
	kind     inputKind
	prompt   string
	buffer   string
	item     fileItem // the item being renamed, for the description (zero for add)
	blink    bool     // cursor blink phase
	blinkGen int
	screenW  int
}

func newInputPopup() inputPopup {
	return inputPopup{anim: newPopupAnimator("input", popupLayerColor(1))}
}

func (m *inputPopup) open(kind inputKind, prompt, buffer string, item fileItem) tea.Cmd {
	m.kind, m.prompt, m.buffer, m.item = kind, prompt, buffer, item
	m.blink, m.blinkGen = true, m.blinkGen+1
	return tea.Batch(m.anim.open(), inputBlinkCmd(m.blinkGen))
}

// onBlink toggles the cursor and reschedules, as long as this is still the
// current open's blink loop.
func (m *inputPopup) onBlink(msg inputBlinkMsg) tea.Cmd {
	if !m.anim.isActive() || msg.gen != m.blinkGen {
		return nil
	}
	m.blink = !m.blink
	return inputBlinkCmd(msg.gen)
}

func inputBlinkCmd(gen int) tea.Cmd {
	return tea.Tick(530*time.Millisecond, func(time.Time) tea.Msg { return inputBlinkMsg{gen} })
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
// rename/add from kind/buffer/item); Esc cancels.
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

	// input row: peach chevron prompt + text + blinking block cursor, no bar (the
	// blinking cursor already marks it as an input — same style as Search).
	cur := " "
	if m.blink {
		cur = "█"
	}
	glyph := lipgloss.NewStyle().Foreground(lipgloss.Color("#fab387")).Bold(true).Render(inputGlyph)
	field := glyph + " " + safeName(m.buffer) + cur

	// description: the item exactly as panel [2] shows it — type icon + eza colour.
	var desc string
	if m.item.name != "" {
		icon := iconFile
		if m.item.isDir {
			icon = iconDir
		}
		desc = " " + lipgloss.NewStyle().Foreground(fileColor(m.item)).Render(icon+" "+safeName(m.item.name))
	}

	innerW := max(lipgloss.Width(title)+4, lipgloss.Width(hint)+4)
	innerW = max(innerW, lipgloss.Width(desc)+4)
	innerW = min(max(innerW, lipgloss.Width(field)+4), maxInnerWidth(m.screenW))

	if lipgloss.Width(field) > innerW { // keep the cursor (tail) visible
		field = ansi.TruncateLeft(field, lipgloss.Width(field)-(innerW-1), "…")
	}
	// pad=false so the content hugs the top border (no empty top); a grey divider
	// sits UNDER the input, same as Search — compact.
	divider := lipgloss.NewStyle().Foreground(dimColor).Render(strings.Repeat("─", innerW))
	var rows []string
	if desc != "" {
		rows = append(rows, desc)
	}
	rows = append(rows, field, divider)
	return drawPopupBoxPad(bc, title, hint, rows, innerW, false)
}
