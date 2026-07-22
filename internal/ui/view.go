package ui

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// kbu colour mindset (§2 / §8.0): blue = structure/focus, surface2 = unfocused
// chrome, overlay0 = dim body text, lavender = user footprint.
var (
	focusColor = lipgloss.Color("#89b4fa") // catppuccin blue  (focused chrome)
	borderDim  = lipgloss.Color("#585b70") // catppuccin surface2 (unfocused chrome)
	dimColor   = lipgloss.Color("#6c7086") // catppuccin overlay0 (dim body text)
	pathColor  = lipgloss.Color("#b4befe") // catppuccin lavender (user footprint)
)

// §8.0/§8.2 powerline caps (rune values so no glyph sits in source).
var (
	capLeft  = string(rune(0xe0b6)) // round-left  — chip start
	capRight = string(rune(0xe0b4)) // round-right — chip end
	capHard  = string(rune(0xe0b0)) // hard right triangle — tab boundary
	capThin  = string(rune(0xe0b1)) // thin chevron — inactive↔inactive divider
)

const crustHex = "#11111b" // catppuccin crust — tab-bar recessed background

func (m AppModel) View() string {
	if m.width == 0 || m.height == 0 {
		return "" // wait for the first WindowSizeMsg
	}

	w, h := m.width, m.height
	midH := h - 2 // minus header + footer rows
	if midH < 3 || w < 24 {
		return "terminal too small"
	}

	// left : middle : right = 1 : 2 : 2 (right absorbs rounding so they sum to w).
	leftW, midW := w/5, w*2/5
	rightW := w - leftW - midW
	if w < 72 { // too narrow for 3 columns; show just the list (real zoom comes later)
		leftW, rightW, midW = 0, 0, w
	}

	listBody := m.active().view(midW-2, midH-2, m.focus == panelList)
	list := m.panelBox(panelList, m.listTitle(midW), midW, midH, listBody)

	middle := list
	if leftW > 0 && rightW > 0 {
		pinH := midH * 2 / 3
		left := lipgloss.JoinVertical(
			lipgloss.Left,
			m.panelBox(panelPin, singleChip("[1] filu", m.focus == panelPin), leftW, pinH, m.places.view(leftW-2, pinH-2, m.focus == panelPin)),
			m.panelBox(panelCarry, singleChip("[4] carry", m.focus == panelCarry), leftW, midH-pinH, m.carry.view(leftW-2, (midH-pinH)-2, m.focus == panelCarry)),
		)
		right := m.panelBox(panelDetail, m.detailTitle(rightW), rightW, midH, m.detailBody(rightW-2, midH-2))
		middle = lipgloss.JoinHorizontal(lipgloss.Top, left, list, right)
	}

	return lipgloss.JoinVertical(lipgloss.Left, m.headerBar(w), middle, m.footerBar(w))
}

// listTitle renders panel [2]'s directory-tab bar, falling back to a single
// chip of the active tab if the bar would overflow the panel width.
func (m AppModel) listTitle(w int) string {
	focused := m.focus == panelList
	labels := make([]string, len(m.tabs))
	for i, t := range m.tabs {
		labels[i] = truncate(pathBase(t.dir), 10)
	}
	if tb := tabBar("[2]", labels, m.tab, focused); lipgloss.Width(tb) <= w-2 {
		return tb
	}
	return singleChip("[2] "+pathBase(m.active().dir), focused)
}

// detailTitle renders panel [3]'s Preview/Info tab bar.
func (m AppModel) detailTitle(w int) string {
	focused := m.focus == panelDetail
	labels := []string{"Preview", "Info"}
	if tb := tabBar("[3]", labels, int(m.detail), focused); lipgloss.Width(tb) <= w-2 {
		return tb
	}
	return singleChip("[3] "+labels[m.detail], focused)
}

// detailBody renders panel [3]'s active tab.
func (m AppModel) detailBody(w, rows int) string {
	if m.detail == tabInfo {
		return renderLines(infoLines(m.active().cursorItem(), m.active().dir), w, rows)
	}
	return m.preview.view(w, rows)
}

// panelBox draws a bordered panel with the title embedded in the top border
// (kbu style). Focused = double border + blue, else rounded + dim.
func (m AppModel) panelBox(id panelID, title string, w, h int, body string) string {
	focused := m.focus == id
	color := borderDim
	tl, tr, bl, br, hz, vt := "╭", "╮", "╰", "╯", "─", "│"
	if focused {
		color = focusColor
		tl, tr, bl, br, hz, vt = "╔", "╗", "╚", "╝", "═", "║"
	}
	bs := lipgloss.NewStyle().Foreground(color)
	inner := max(w-2, 1)
	// title is pre-rendered chrome (singleChip / tabBar); place it on the top edge.
	fill := max(inner-lipgloss.Width(title), 0)
	top := bs.Render(tl) + title + bs.Render(strings.Repeat(hz, fill)+tr)

	bodyLines := strings.Split(body, "\n")
	var b strings.Builder
	b.WriteString(top + "\n")
	for r := range h - 2 {
		line := ""
		if r < len(bodyLines) {
			line = bodyLines[r]
		}
		if d := inner - lipgloss.Width(line); d > 0 {
			line += strings.Repeat(" ", d)
		}
		b.WriteString(bs.Render(vt) + line + bs.Render(vt) + "\n")
	}
	b.WriteString(bs.Render(bl + strings.Repeat(hz, inner) + br))
	return b.String()
}

func (m AppModel) headerBar(w int) string {
	return lipgloss.NewStyle().Width(w).Foreground(pathColor).
		Render(truncate(" "+breadcrumb(m.active().dir), w))
}

func (m AppModel) footerBar(w int) string {
	color, content := dimColor, " space menu   ? help   tab/1-4 panels   q quit"
	if m.input.kind != inputNone {
		color = focusColor
		content = " " + m.input.prompt + ": " + m.input.buffer + "█"
	}
	return lipgloss.NewStyle().Width(w).Foreground(color).Render(truncate(content, w))
}

func pathBase(p string) string {
	if b := filepath.Base(p); b != "." && b != string(filepath.Separator) {
		return b
	}
	return "/"
}

// breadcrumb turns a path into "~ › a › b › c" (home folded to ~).
func breadcrumb(p string) string {
	if home, err := os.UserHomeDir(); err == nil && strings.HasPrefix(p, home) {
		p = "~" + strings.TrimPrefix(p, home)
	}
	var segs []string
	for s := range strings.SplitSeq(p, string(filepath.Separator)) {
		if s != "" {
			segs = append(segs, s)
		}
	}
	return strings.Join(segs, " › ")
}

// truncate tail-clips to width w (middle-elision comes later).
func truncate(s string, w int) string {
	if w <= 0 {
		return ""
	}
	if lipgloss.Width(s) <= w {
		return s
	}
	r := []rune(s)
	if len(r) > w-1 {
		r = r[:w-1]
	}
	return string(r) + "…"
}
