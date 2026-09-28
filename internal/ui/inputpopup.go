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
	// check validates the trimmed value on Enter: "" lets the submit through,
	// anything else is the reason it can't go, shown under the field (tdp K3).
	check func(string) string
	// errMsg is the last failed check's reason; typing clears it.
	errMsg string
	// width is the box's inner width, fixed when the popup opens so it does not
	// grow or shrink with the value or the error line (tdp L2).
	width int
}

func newInputPopup() inputPopup {
	return inputPopup{anim: newPopupAnimator("input", popupLayerColor(1))}
}

func (m *inputPopup) open(kind inputKind, prompt, buffer string, item fileItem) tea.Cmd {
	m.kind, m.prompt, m.buffer, m.item = kind, prompt, buffer, item
	m.check, m.errMsg = nil, ""
	m.blink, m.blinkGen = true, m.blinkGen+1
	m.width = m.openWidth()
	return tea.Batch(m.anim.open(), inputBlinkCmd(m.blinkGen))
}

// minInputWidth keeps room to type even when the prompt and hint are short.
const minInputWidth = 40

// openWidth is the box width for this open: the title, the hint, the item
// description and the prefilled value, with room to type, capped by the screen.
func (m inputPopup) openWidth() int {
	w := max(minInputWidth, lipgloss.Width(" "+m.prompt)+4, lipgloss.Width(m.hint())+4,
		lipgloss.Width(m.desc())+4, lipgloss.Width(inputGlyph+" "+safeName(m.buffer)+"█")+4)
	return min(w, maxInnerWidth(m.screenW))
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
func (m inputPopup) owns() bool          { return m.anim.owns() }
func (m inputPopup) isInteractive() bool { return m.anim.isInteractive() }
func (m *inputPopup) handleTick(msg AnimTickMsg) tea.Cmd {
	if msg.Target != m.anim.target {
		return nil
	}
	return m.anim.tick()
}

// update edits the buffer. Enter is always a submit (tdp K3): the value is
// checked first, and only a value that passes closes the popup and reports
// committed (the caller performs the rename/add/zip from kind/buffer/item). A
// value that fails keeps the popup and the focus on the field and says why.
// Esc cancels.
func (m inputPopup) update(msg tea.KeyMsg) (inputPopup, bool, tea.Cmd) {
	if !m.anim.isInteractive() {
		return m, false, nil
	}
	switch msg.Type {
	case tea.KeyEsc:
		return m, false, m.anim.close()
	case tea.KeyEnter:
		if m.check != nil {
			if reason := m.check(strings.TrimSpace(m.buffer)); reason != "" {
				m.errMsg = reason
				return m, false, nil
			}
		}
		return m, true, m.anim.close()
	case tea.KeyBackspace:
		if r := []rune(m.buffer); len(r) > 0 {
			m.buffer = string(r[:len(r)-1])
		}
		m.errMsg = ""
	case tea.KeySpace:
		m.buffer += " "
		m.errMsg = ""
	case tea.KeyRunes:
		m.buffer += string(msg.Runes)
		m.errMsg = ""
	}
	return m, false, nil
}

// hint names what Enter does for this kind of input (tdp D3).
func (m inputPopup) hint() string {
	verb := "confirm"
	switch m.kind {
	case inputRename:
		verb = "rename"
	case inputAdd:
		verb = "create"
	case inputZip:
		verb = "zip"
	}
	return " Enter " + verb + " · Esc cancel "
}

// desc is the item exactly as panel [1] shows it — type icon + eza colour — or
// "" when the input is not about an existing item.
func (m inputPopup) desc() string {
	if m.item.name == "" {
		return ""
	}
	return " " + lipgloss.NewStyle().Foreground(fileColor(m.item)).Render(fileIcon(m.item)+" "+safeName(m.item.name))
}

func (m inputPopup) renderPopup() string { return m.anim.renderFrame(m.renderFull()) }

func (m inputPopup) renderFull() string {
	bc := popupLayerColor(m.anim.layer)
	title := " " + m.prompt

	// input row: peach chevron prompt + text + blinking block cursor, no bar (the
	// blinking cursor already marks it as an input — same style as Search).
	cur := " "
	if m.blink {
		cur = "█"
	}
	glyph := lipgloss.NewStyle().Foreground(lipgloss.Color("#fab387")).Bold(true).Render(inputGlyph)
	field := glyph + " " + safeName(m.buffer) + cur

	innerW := m.width
	if innerW <= 0 { // not opened through open() (tests): size to the content
		innerW = m.openWidth()
	}
	innerW = min(innerW, maxInnerWidth(m.screenW))

	if lipgloss.Width(field) > innerW { // keep the cursor (tail) visible
		field = ansi.TruncateLeft(field, lipgloss.Width(field)-(innerW-1), "…")
	}
	// pad=false so the content hugs the top border (no empty top); a grey divider
	// sits UNDER the input, same as Search — compact.
	divider := lipgloss.NewStyle().Foreground(dimColor).Render(strings.Repeat("─", innerW))
	var rows []string
	if d := m.desc(); d != "" {
		rows = append(rows, d)
	}
	rows = append(rows, field, divider)
	if m.errMsg != "" { // why Enter did not go through, wrapped to the fixed width
		red := lipgloss.NewStyle().Foreground(lipgloss.Color("#f38ba8"))
		for _, l := range wrapWords(m.errMsg, innerW-2) {
			rows = append(rows, red.Render(" "+l))
		}
	}
	return drawPopupBoxPad(bc, title, m.hint(), rows, innerW, false)
}
