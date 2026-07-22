package ui

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	overlay "github.com/rmhubbert/bubbletea-overlay"
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
	// file-type content colours live in theme.go (eza catppuccin-mocha).
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

	out := lipgloss.JoinVertical(lipgloss.Left, m.headerBar(w), m.middleView(w, midH), m.footerBar(w))
	// Compose-don't-Replace: overlay popups onto the canvas (last = on top).
	if m.spaceMenu.isActive() {
		out = overlay.Composite(m.spaceMenu.renderPopup(), out, overlay.Center, overlay.Center, 0, 0)
	}
	if m.confirm.isActive() {
		out = overlay.Composite(m.confirm.renderPopup(), out, overlay.Center, overlay.Center, 0, 0)
	}
	if m.inputPopup.isActive() {
		out = overlay.Composite(m.inputPopup.renderPopup(), out, overlay.Center, overlay.Center, 0, 0)
	}
	return out
}

// splitN divides w into n columns, the last absorbing the remainder so they
// sum to w exactly.
func splitN(w, n int) []int {
	out := make([]int, n)
	each := w / n
	for i := range out {
		out[i] = each
	}
	out[n-1] = w - each*(n-1)
	return out
}

// middleView builds the panel region, honouring the zoom state.
func (m AppModel) middleView(w, midH int) string {
	switch m.zoom {
	case panelList:
		return m.zoomListView(w, midH)
	case panelDetail:
		return m.zoomDetailView(w, midH)
	case panelCarry:
		return m.zoomCarryView(w, midH)
	default:
		return m.normalMiddle(w, midH)
	}
}

// normalMiddle is the default layout:
//
//	[1][2][3]
//	[1][2][3]
//	[4][4][3]
//
// [1] places + [2] list share the top 2/3; [4] carry spans their two columns
// along the bottom 1/3; [3] detail is the full-height right column.
func (m AppModel) normalMiddle(w, midH int) string {
	leftW, midW := w/3, w/3
	rightW := w - leftW - midW
	listFocus := m.focus == panelList
	if w < 72 { // too narrow for the grid; the list alone (Space menu Zoom is the escape hatch)
		return m.panelBox(listFocus, m.listTitle(w), w, midH, m.active().view(w-2, midH-2, listFocus))
	}
	topH := midH * 2 / 3
	botH := midH - topH

	pin := m.panelBox(m.focus == panelPin, singleChip("[1] filu", m.focus == panelPin), leftW, topH, m.places.view(leftW-2, topH-2, m.focus == panelPin))
	list := m.panelBox(listFocus, m.listTitle(midW), midW, topH, m.active().view(midW-2, topH-2, listFocus))
	top := lipgloss.JoinHorizontal(lipgloss.Top, pin, list)

	carryW := leftW + midW
	carry := m.panelBox(m.focus == panelCarry, m.carryTitle(carryW), carryW, botH, m.carryBody(carryW-2, botH-2))

	leftRegion := lipgloss.JoinVertical(lipgloss.Left, top, carry)
	detail := m.panelBox(m.focus == panelDetail, m.detailTitle(rightW), rightW, midH, m.detailBody(rightW-2, midH-2))
	return lipgloss.JoinHorizontal(lipgloss.Top, leftRegion, detail)
}

// zoomListView (panel [2] zoom): [2] fully expanded over [4] fully expanded,
// each as its tabs 1:1:1; [1]/[3] hidden. 2/4 pick which is focused, h/l its tab.
func (m AppModel) zoomListView(w, midH int) string {
	topH := midH * 2 / 3
	// [2]-zoom mixes two panels, so each column's chip carries its panel number.
	return lipgloss.JoinVertical(lipgloss.Left,
		m.expandedListTabs(w, topH),
		m.expandedCarryTabs(w, midH-topH, true))
}

// expandedListTabs lays panel [2]'s 3 directory tabs out as equal-width columns;
// the active tab is the focused column when [2] holds focus.
func (m AppModel) expandedListTabs(w, h int) string {
	widths := splitN(w, len(m.tabs))
	cols := make([]string, len(m.tabs))
	for i := range m.tabs {
		cw := widths[i]
		focused := m.focus == panelList && m.tab == i
		cols[i] = m.panelBox(focused, singleChip("[2] "+pathBase(m.tabs[i].dir), focused), cw, h, m.tabs[i].view(cw-2, h-2, focused))
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, cols...)
}

// zoomDetailView (panel [3] zoom): full-width, the two tabs become 1:1 panels;
// the active tab (m.detail) is the focused column.
func (m AppModel) zoomDetailView(w, midH int) string {
	leftW := w / 2
	rightW := w - leftW
	pv := m.focus == panelDetail && m.detail == tabPreview
	mt := m.focus == panelDetail && m.detail == tabMeta
	preview := m.panelBox(pv, singleChip("Preview", pv), leftW, midH,
		renderLinesFrom(m.preview.contentLines(), m.detailScroll, leftW-2, midH-2))
	meta := m.panelBox(mt, singleChip("Meta", mt), rightW, midH,
		renderLinesFrom(metaLines(m.active().cursorItem(), m.active().dir), 0, rightW-2, midH-2))
	return lipgloss.JoinHorizontal(lipgloss.Top, preview, meta)
}

// zoomCarryView (panel [4] zoom): [4] full-screen, its three tabs 1:1:1.
// Single-panel zoom, so no panel-number prefix.
func (m AppModel) zoomCarryView(w, midH int) string {
	return m.expandedCarryTabs(w, midH, false)
}

// expandedCarryTabs lays panel [4]'s 3 tabs out as equal-width columns; the
// active tab (m.carryTab) is the focused column when [4] holds focus. numbered
// prefixes each chip with "[4]" (used in [2]-zoom, where panels are mixed).
func (m AppModel) expandedCarryTabs(w, h int, numbered bool) string {
	wd := splitN(w, 3)
	foc := func(i int) bool { return m.focus == panelCarry && m.carryTab == i }
	label := func(s string) string {
		if numbered {
			return "[4] " + s
		}
		return s
	}
	carry := m.panelBox(foc(0), singleChip(label("Carries"), foc(0)), wd[0], h, m.carry.view(wd[0]-2, h-2, foc(0)))
	progress := m.panelBox(foc(1), singleChip(label("Progress"), foc(1)), wd[1], h, centeredNote(wd[1]-2, h-2, "(no active tasks)"))
	history := m.panelBox(foc(2), singleChip(label("History"), foc(2)), wd[2], h, m.carry.historyView(wd[2]-2, h-2))
	return lipgloss.JoinHorizontal(lipgloss.Top, carry, progress, history)
}

// listTitle renders panel [2]'s fixed 3-tab bar. The tabs are always shown; when
// the bar would overflow the column we shrink the per-tab labels rather than
// collapsing to a single chip (the 3 tabs are a fixed part of the design).
func (m AppModel) listTitle(w int) string {
	focused := m.focus == panelList
	labels := make([]string, len(m.tabs))
	for maxLen := 10; ; maxLen-- {
		for i, t := range m.tabs {
			labels[i] = truncate(pathBase(t.dir), maxLen)
		}
		tb := tabBar("[2]", labels, m.tab, focused)
		if maxLen == 1 || lipgloss.Width(tb) <= w-2 {
			return tb
		}
	}
}

// carryTitle renders panel [4]'s tab bar. Like panel [2] it prefers the full
// starship tab bar and only falls back to the compact carousel when the panel is
// too narrow to fit it — see carouselChip for that narrow-panel tab strategy.
func (m AppModel) carryTitle(w int) string {
	focused := m.focus == panelCarry
	labels := []string{"Carries", "Progress", "History"}
	if tb := tabBar("[4]", labels, m.carryTab, focused); lipgloss.Width(tb) <= w-2 {
		return tb
	}
	return carouselChip("[4]", labels, m.carryTab, focused)
}

// carryBody renders panel [4]'s active tab.
func (m AppModel) carryBody(w, rows int) string {
	switch m.carryTab {
	case 1: // progress — async land is future work
		return centeredNote(w, rows, "(no active tasks)")
	case 2: // history
		return m.carry.historyView(w, rows)
	default: // carry
		return m.carry.view(w, rows, m.focus == panelCarry)
	}
}

// detailTitle renders panel [3]'s Preview/Meta tab bar.
func (m AppModel) detailTitle(w int) string {
	focused := m.focus == panelDetail
	labels := []string{"Preview", "Meta"}
	if tb := tabBar("[3]", labels, int(m.detail), focused); lipgloss.Width(tb) <= w-2 {
		return tb
	}
	return singleChip("[3] "+labels[m.detail], focused)
}

// detailLines returns the full content of panel [3]'s active tab.
func (m AppModel) detailLines() []string {
	if m.detail == tabMeta {
		return metaLines(m.active().cursorItem(), m.active().dir)
	}
	return m.preview.contentLines()
}

// detailBody renders panel [3]'s active tab from the scroll offset. Panel [3] is
// a reference view (read while another panel has focus), so it keeps its colour
// even when unfocused rather than dimming.
func (m AppModel) detailBody(w, rows int) string {
	return renderLinesFrom(m.detailLines(), m.detailScroll, w, rows)
}

// panelBox draws a bordered panel with the title embedded in the top border
// (kbu style). Focused = double border + blue, else rounded + dim.
func (m AppModel) panelBox(focused bool, title string, w, h int, body string) string {
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
	glyph := string(rune(0xf07c)) // nf-fa-folder-open
	return lipgloss.NewStyle().Width(w).Foreground(userColor).
		Render(truncate(" "+glyph+" "+shortPath(m.active().dir), w))
}

func (m AppModel) footerBar(w int) string {
	return lipgloss.NewStyle().Width(w).Foreground(dimColor).
		Render(truncate(" space menu   ? help   tab/1-4 panels   q quit", w))
}

func pathBase(p string) string {
	if b := filepath.Base(p); b != "." && b != string(filepath.Separator) {
		return b
	}
	return "/"
}

// shortPath folds the home dir to ~ (keeps normal / separators).
func shortPath(p string) string {
	if home, err := os.UserHomeDir(); err == nil && strings.HasPrefix(p, home) {
		return "~" + strings.TrimPrefix(p, home)
	}
	return p
}

// truncate clips s to display width w (ANSI- and wide-char-aware), adding "…".
func truncate(s string, w int) string {
	if w <= 0 {
		return ""
	}
	return ansi.Truncate(s, w, "…")
}
