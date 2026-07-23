package ui

import (
	"fmt"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// detailYank is panel [3]'s yank viewport (kbu yamlpopup form): a vim-style
// cursor over the detail content with character-wise visual selection. `y`
// copies the selection, or the whole content when nothing is selected.
type detailYank struct {
	title string
	lines []string // styled (syntax-highlighted) display lines
	plain []string // ANSI-stripped mirror — authoritative for cursor/selection
	full  string   // whole content joined, for "copy all"

	showGutter bool // draw a non-selectable line-number gutter (text/hex)

	cursorLine, cursorCol int
	visual                bool
	anchorLine, anchorCol int
	pendingG              bool
	scroll                int

	width, height int
	anim          popupAnimator
}

func newDetailYank() detailYank {
	return detailYank{anim: newPopupAnimator("detailyank", popupLayerColor(1))}
}

// open loads content and resets the cursor to the top. showGutter draws a
// non-selectable line-number gutter (for text/hex, like kbu's YAML popup).
func (m *detailYank) open(title string, lines []string, showGutter bool) tea.Cmd {
	m.title = title
	m.lines = lines
	m.showGutter = showGutter
	m.plain = make([]string, len(lines))
	for i, l := range lines {
		m.plain[i] = ansi.Strip(l)
	}
	m.full = strings.Join(m.plain, "\n")
	m.cursorLine, m.cursorCol, m.scroll = 0, 0, 0
	m.visual, m.pendingG = false, false
	return m.anim.open()
}

func (m *detailYank) setSize(w, h int) { m.width, m.height = w, h }
func (m detailYank) isActive() bool    { return m.anim.isActive() }
func (m detailYank) isInteractive() bool {
	return m.anim.isInteractive()
}

func (m *detailYank) handleTick(msg AnimTickMsg) tea.Cmd {
	if msg.Target != m.anim.target {
		return nil
	}
	return m.anim.tick()
}

func (m detailYank) contentRows() int { return max(m.height-6, 3) }
func (m detailYank) innerW() int      { return max(m.width-8, 20) }

func (m detailYank) lastLine() int { return max(len(m.plain)-1, 0) }

func (m detailYank) lastCol(line int) int {
	if line < 0 || line >= len(m.plain) {
		return 0
	}
	return max(len([]rune(m.plain[line]))-1, 0)
}

func (m *detailYank) clampCol() {
	if m.cursorCol > m.lastCol(m.cursorLine) {
		m.cursorCol = m.lastCol(m.cursorLine)
	}
	if m.cursorCol < 0 {
		m.cursorCol = 0
	}
}

func (m *detailYank) ensureVisible() {
	rows := m.contentRows()
	if m.cursorLine < m.scroll {
		m.scroll = m.cursorLine
	}
	if m.cursorLine >= m.scroll+rows {
		m.scroll = m.cursorLine - rows + 1
	}
	if m.scroll < 0 {
		m.scroll = 0
	}
}

// update handles motion (hjkl, 0/$, gg/G, d/u), v (visual toggle), y (copy), and
// Esc (leave visual, then close).
func (m detailYank) update(msg tea.KeyMsg) (detailYank, tea.Cmd) {
	if !m.anim.isInteractive() {
		return m, nil
	}
	switch msg.String() {
	case "esc":
		if m.visual {
			m.visual, m.pendingG = false, false
			return m, nil
		}
		m.pendingG = false
		return m, m.anim.close()
	case "h", "left":
		if m.cursorCol > 0 {
			m.cursorCol--
		}
		m.pendingG = false
	case "l", "right":
		if m.cursorCol < m.lastCol(m.cursorLine) {
			m.cursorCol++
		}
		m.pendingG = false
	case "j", "down":
		if m.cursorLine < m.lastLine() {
			m.cursorLine++
			m.clampCol()
		}
		m.pendingG = false
		m.ensureVisible()
	case "k", "up":
		if m.cursorLine > 0 {
			m.cursorLine--
			m.clampCol()
		}
		m.pendingG = false
		m.ensureVisible()
	case "0":
		m.cursorCol, m.pendingG = 0, false
	case "$":
		m.cursorCol, m.pendingG = m.lastCol(m.cursorLine), false
	case "d", "ctrl+d":
		m.cursorLine = min(m.cursorLine+m.contentRows()/2, m.lastLine())
		m.clampCol()
		m.pendingG = false
		m.ensureVisible()
	case "u", "ctrl+u":
		m.cursorLine = max(m.cursorLine-m.contentRows()/2, 0)
		m.clampCol()
		m.pendingG = false
		m.ensureVisible()
	case "G":
		m.cursorLine = m.lastLine()
		m.clampCol()
		m.pendingG = false
		m.ensureVisible()
	case "g":
		if m.pendingG {
			m.cursorLine, m.pendingG = 0, false
			m.clampCol()
			m.ensureVisible()
		} else {
			m.pendingG = true
		}
	case "v":
		if m.visual {
			m.visual = false
		} else {
			m.visual = true
			m.anchorLine, m.anchorCol = m.cursorLine, m.cursorCol
		}
		m.pendingG = false
	case "y":
		m.pendingG = false
		if m.visual {
			text := m.selectionText()
			m.visual = false
			if text == "" {
				return m, nil
			}
			return m, copyToClipboardCmd(text, "Copied selection to clipboard")
		}
		if m.full == "" {
			return m, nil
		}
		return m, copyToClipboardCmd(m.full, "Copied all to clipboard")
	}
	return m, nil
}

// selectionRange normalises (anchor, cursor) to forward order, inclusive.
func (m detailYank) selectionRange() (sL, sC, eL, eC int) {
	sL, sC = m.anchorLine, m.anchorCol
	eL, eC = m.cursorLine, m.cursorCol
	if sL > eL || (sL == eL && sC > eC) {
		sL, sC, eL, eC = eL, eC, sL, sC
	}
	return
}

// selectionText extracts the character-wise selection from the plain lines.
func (m detailYank) selectionText() string {
	if len(m.plain) == 0 {
		return ""
	}
	sL, sC, eL, eC := m.selectionRange()
	if sL == eL {
		rr := []rune(m.plain[sL])
		if len(rr) == 0 {
			return ""
		}
		if eC >= len(rr) {
			eC = len(rr) - 1
		}
		return string(rr[sC : eC+1])
	}
	var b strings.Builder
	start := []rune(m.plain[sL])
	if sC < len(start) {
		b.WriteString(string(start[sC:]))
	}
	b.WriteByte('\n')
	for i := sL + 1; i < eL; i++ {
		b.WriteString(m.plain[i])
		b.WriteByte('\n')
	}
	end := []rune(m.plain[eL])
	if len(end) > 0 {
		if eC >= len(end) {
			eC = len(end) - 1
		}
		b.WriteString(string(end[0 : eC+1]))
	}
	return b.String()
}

func (m detailYank) renderPopup() string { return m.anim.renderFrame(m.renderFull()) }

func (m detailYank) renderFull() string {
	bc := popupLayerColor(1)
	innerW, rows := m.innerW(), m.contentRows()
	selStyle := lipgloss.NewStyle().Background(userColor).Foreground(lipgloss.Color(baseHex)).Bold(true)
	curStyle := lipgloss.NewStyle().Reverse(true)
	gutStyle := lipgloss.NewStyle().Foreground(dimColor)
	numW := 0
	if m.showGutter {
		numW = max(len(strconv.Itoa(len(m.lines))), 2)
	}

	sL, sC, eL, eC := m.selectionRange()
	out := make([]string, 0, rows)
	for i := m.scroll; i < min(m.scroll+rows, len(m.lines)); i++ {
		styled, plain := m.lines[i], m.plain[i]
		var body string
		switch {
		case m.visual && i >= sL && i <= eL:
			ls, le := 0, m.lastCol(i)
			if i == sL {
				ls = sC
			}
			if i == eL {
				le = eC
			}
			body = overlaySelectionOnStyledLine(styled, plain, ls, le, i == m.cursorLine, m.cursorCol, selStyle, curStyle)
		case i == m.cursorLine:
			body = overlayCursorOnStyledLine(styled, plain, m.cursorCol, curStyle)
		default:
			body = styled
		}
		if m.showGutter { // gutter is display-only — the cursor never enters it
			body = gutStyle.Render(fmt.Sprintf("%*d "+string(rune(0x2502))+" ", numW, i+1)) + body
		}
		out = append(out, ansi.Truncate(body, innerW, ""))
	}
	for len(out) < rows {
		out = append(out, "")
	}
	return drawPopupBox(bc, " "+m.title, " v:visual   y:copy   Esc:close ", out, innerW)
}

// overlaySelectionOnStyledLine keeps the styled line intact outside the
// selection and paints the selection bg inside, nesting the reverse-video cursor
// cell at its endpoint. ansi.Cut is escape/grapheme aware so highlight spans that
// straddle the range don't break. (Ported from kbu.)
func overlaySelectionOnStyledLine(styled, plain string, selStart, selEnd int, hasCursor bool, cursorCol int, selectionStyle, cursorStyle lipgloss.Style) string {
	pr := []rune(plain)
	if len(pr) == 0 {
		if hasCursor {
			return cursorStyle.Render(" ")
		}
		return selectionStyle.Render(" ")
	}
	if selStart < 0 {
		selStart = 0
	}
	if selEnd >= len(pr) {
		selEnd = len(pr) - 1
	}
	const big = 1_000_000
	before := ansi.Cut(styled, 0, selStart)
	after := ansi.Cut(styled, selEnd+1, big)
	var block strings.Builder
	for i := selStart; i <= selEnd; i++ {
		cell := string(pr[i])
		if hasCursor && i == cursorCol {
			block.WriteString(cursorStyle.Render(cell))
		} else {
			block.WriteString(selectionStyle.Render(cell))
		}
	}
	return before + block.String() + after
}

// overlayCursorOnStyledLine flips only the cell at cursorCol to reverse video,
// keeping the rest of the syntax colouring. (Ported from kbu.)
func overlayCursorOnStyledLine(styled, plain string, cursorCol int, cursorStyle lipgloss.Style) string {
	pr := []rune(plain)
	if len(pr) == 0 {
		return cursorStyle.Render(" ")
	}
	if cursorCol < 0 {
		cursorCol = 0
	}
	if cursorCol >= len(pr) {
		cursorCol = len(pr) - 1
	}
	const big = 1_000_000
	before := ansi.Cut(styled, 0, cursorCol)
	after := ansi.Cut(styled, cursorCol+1, big)
	return before + cursorStyle.Render(string(pr[cursorCol])) + after
}
