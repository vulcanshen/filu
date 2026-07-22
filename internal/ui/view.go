package ui

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// kbu colour hierarchy (§2 / §B): three reserved tiers.
var (
	// structural (system) — panel chrome + focus; never user state.
	focusColor = lipgloss.Color("#89b4fa") // blue    : focused border/chrome
	borderDim  = lipgloss.Color("#585b70") // surface2: unfocused chrome
	// user footprint — pinned / carried / current location / remembered cursor.
	userColor = lipgloss.Color("#b4befe") // lavender
	handColor = lipgloss.Color("#bac2de") // subtext1: focused cursor ("current hand")
	// neutral text.
	dimColor = lipgloss.Color("#6c7086") // overlay0: section headers / dim text
	// popup layer scale (lavenphire25→sapphire) comes when popups land.
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

	// left : middle : right = 1 : 1 : 1 (right absorbs rounding so they sum to w).
	leftW, midW := w/3, w/3
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
			m.panelBox(panelCarry, m.carryTitle(), leftW, midH-pinH, m.carryBody(leftW-2, (midH-pinH)-2)),
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

// carryTitle renders panel [4]'s compact carousel (active tab + next initial).
func (m AppModel) carryTitle() string {
	labels := []string{"Carry", "Progress", "History"}
	return carouselChip("[4]", labels, m.carryTab, m.focus == panelCarry)
}

// carryBody renders panel [4]'s active tab.
func (m AppModel) carryBody(w, rows int) string {
	switch m.carryTab {
	case 1: // progress — async land is future work
		return lipgloss.NewStyle().Foreground(dimColor).Render("(no active tasks)")
	case 2: // history
		return m.carry.historyView(w, rows)
	default: // carry
		return m.carry.view(w, rows, m.focus == panelCarry)
	}
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

// detailLines returns the full content of panel [3]'s active tab.
func (m AppModel) detailLines() []string {
	if m.detail == tabInfo {
		return infoLines(m.active().cursorItem(), m.active().dir)
	}
	return m.preview.contentLines()
}

// detailBody renders panel [3]'s active tab from the scroll offset.
func (m AppModel) detailBody(w, rows int) string {
	return renderLinesFrom(m.detailLines(), m.detailScroll, w, rows)
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
	return lipgloss.NewStyle().Width(w).Foreground(userColor).
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

// truncate tail-clips s to display width w (wide-char aware), appending "…" when
// clipped. Assumes s carries no ANSI (callers style after truncating).
func truncate(s string, w int) string {
	if w <= 0 {
		return ""
	}
	if lipgloss.Width(s) <= w {
		return s
	}
	var b strings.Builder
	width := 0
	for _, r := range s {
		rw := lipgloss.Width(string(r))
		if width+rw > w-1 {
			break
		}
		b.WriteRune(r)
		width += rw
	}
	return b.String() + "…"
}
