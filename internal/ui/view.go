package ui

import (
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
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
	midH := h - 3 // minus header + spacer + footer rows
	if midH < 3 || w < 24 {
		return "terminal too small"
	}

	if m.splash.isActive() { // hidden easter-egg logo takes over the whole screen
		return m.splash.render(w, h)
	}

	out := joinV(m.headerBar(w), m.statusBar(w), m.middleView(w, midH), m.footerBar(w))
	// Compose-don't-Replace: overlay popups onto the canvas (last = on top).
	if m.spaceMenu.isActive() {
		out = overlay.Composite(m.spaceMenu.renderPopup(), out, overlay.Center, overlay.Center, 0, 0)
	}
	if m.sortMenu.isActive() {
		out = overlay.Composite(m.sortMenu.renderPopup(), out, overlay.Center, overlay.Center, 0, 0)
	}
	if m.gotoMenu.isActive() {
		out = overlay.Composite(m.gotoMenu.renderPopup(), out, overlay.Center, overlay.Center, 0, 0)
	}
	if m.quitMenu.isActive() {
		out = overlay.Composite(m.quitMenu.renderPopup(), out, overlay.Center, overlay.Center, 0, 0)
	}
	if m.openWithMenu.isActive() {
		out = overlay.Composite(m.openWithMenu.renderPopup(), out, overlay.Center, overlay.Center, 0, 0)
	}
	if m.confirm.isActive() {
		out = overlay.Composite(m.confirm.renderPopup(), out, overlay.Center, overlay.Center, 0, 0)
	}
	if m.inputPopup.isActive() {
		out = overlay.Composite(m.inputPopup.renderPopup(), out, overlay.Center, overlay.Center, 0, 0)
	}
	if m.help.isActive() {
		out = overlay.Composite(m.help.renderPopup(), out, overlay.Center, overlay.Center, 0, 0)
	}
	if m.detailYank.isActive() { // yank viewport over the panels
		out = overlay.Composite(m.detailYank.renderPopup(), out, overlay.Center, overlay.Center, 0, 0)
	}
	if m.search.isActive() { // fuzzy finder over the panels
		out = overlay.Composite(m.search.renderPopup(), out, overlay.Center, overlay.Center, 0, 0)
	}
	if m.breadcrumb.isActive() { // ancestor-jump popup over the panels
		out = overlay.Composite(m.breadcrumb.renderPopup(), out, overlay.Center, overlay.Center, 0, 0)
	}
	if m.pty.isRendered() { // shell popup: full width, pinned below header+status, down to the bottom
		out = overlay.Composite(m.pty.renderPopup(), out, overlay.Left, overlay.Top, 0, ptyChromeRows)
	}
	if m.toast.isActive() { // transient, always on top
		out = overlay.Composite(m.toast.renderPopup(), out, overlay.Center, overlay.Center, 0, 0)
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
	case panelMeta:
		return m.zoomMetaView(w, midH)
	case panelCarry:
		return m.zoomCarryView(w, midH)
	default:
		return m.normalMiddle(w, midH)
	}
}

// normalMiddle is the default layout:
//
//	[1][1][2][2]
//	[1][1][2][2]
//	[3][3][3][4]
//
// The top row is [1] list | [2] preview at 50/50; the bottom row is [3] carry |
// [4] meta at 2/3 : 1/3. The vertical seam deliberately differs between the two
// rows (50% on top, 66% on the bottom) — carry, the op centre, takes the wider
// bottom-left.
func (m AppModel) normalMiddle(w, midH int) string {
	listFocus := m.focus == panelList
	if w < 72 { // too narrow for the grid; the list alone (Space menu Zoom is the escape hatch)
		return m.panelBox(listFocus, m.listTitle(w), w, midH, m.active().view(w-2, midH-2, listFocus, m.carry.inBucket(), m.places.pinnedSet()))
	}
	topH := midH * 2 / 3
	botH := midH - topH

	// Top row: list | preview, 50/50.
	listW := w / 2
	previewW := w - listW
	list := m.panelBox(listFocus, m.listTitle(listW), listW, topH, m.active().view(listW-2, topH-2, listFocus, m.carry.inBucket(), m.places.pinnedSet()))
	preview := m.panelBox(m.focus == panelDetail, m.detailTitle(previewW), previewW, topH, m.detailBody(previewW-2, topH-2))
	topRow := joinH(list, preview)

	// Bottom row: carry | meta, 2/3 : 1/3.
	carryW := w * 2 / 3
	metaW := w - carryW
	carry := m.panelBox(m.focus == panelCarry, m.carryTitle(carryW), carryW, botH, m.carryBody(carryW-2, botH-2))
	meta := m.panelBox(m.focus == panelMeta, m.metaTitle(metaW), metaW, botH, m.metaBody(metaW-2, botH-2))
	botRow := joinH(carry, meta)

	return joinV(topRow, botRow)
}

// zoomListView (panel [2] zoom): [2] fully expanded over [4] fully expanded,
// each as its tabs 1:1:1; [1]/[3] hidden. 2/4 pick which is focused, h/l its tab.
func (m AppModel) zoomListView(w, midH int) string {
	topH := midH * 2 / 3
	// [2]-zoom mixes two panels, so each column's chip carries its panel number.
	return joinV(
		m.expandedListTabs(w, topH),
		m.expandedCarryTabs(w, midH-topH, true))
}

// expandedListTabs lays panel [1]'s directory tabs out as equal-width columns;
// the active tab is the focused column when [1] holds focus.
func (m AppModel) expandedListTabs(w, h int) string {
	widths := splitN(w, len(m.tabs))
	cols := make([]string, len(m.tabs))
	carried := m.carry.inBucket()
	pinned := m.places.pinnedSet()
	for i := range m.tabs {
		cw := widths[i]
		focused := m.focus == panelList && m.tab == i
		// trailing space: singleChip sits flush against its round cap, so a wide
		// Roman-numeral glyph (Ⅱ/Ⅲ/Ⅳ) gets clipped by it — pad a cell as tabBar does.
		cols[i] = m.panelBox(focused, singleChip("[1] "+tabNumeral(i)+" ", focused), cw, h, m.tabs[i].view(cw-2, h-2, focused, carried, pinned))
	}
	return joinH(cols...)
}

// zoomDetailView (panel [3] zoom): the preview full-screen.
func (m AppModel) zoomDetailView(w, midH int) string {
	return m.panelBox(true, singleChip("Preview", true), w, midH,
		renderLinesFrom(m.preview.contentLines(), m.detailScroll, w-2, midH-2))
}

// zoomMetaView (panel [4] zoom): the file metadata full-screen.
func (m AppModel) zoomMetaView(w, midH int) string {
	return m.panelBox(true, singleChip("Meta", true), w, midH,
		renderLinesFrom(m.metaContent(), m.metaScroll, w-2, midH-2))
}

// zoomCarryView (panel [4] zoom): [4] full-screen, its three tabs 1:1:1.
// Single-panel zoom, so no panel-number prefix.
func (m AppModel) zoomCarryView(w, midH int) string {
	return m.expandedCarryTabs(w, midH, false)
}

// expandedCarryTabs lays panel [4]'s 2 tabs out as equal-width columns; the
// active tab (m.carryTab) is the focused column when [4] holds focus. numbered
// prefixes each chip with "[4]" (used in [2]-zoom, where panels are mixed).
func (m AppModel) expandedCarryTabs(w, h int, numbered bool) string {
	wd := splitN(w, 2)
	foc := func(i int) bool { return m.focus == panelCarry && m.carryTab == i }
	label := func(s string) string {
		if numbered {
			return "[3] " + s
		}
		return s
	}
	carries := m.panelBox(foc(0), singleChip(label("Carries"), foc(0)), wd[0], h, m.carry.view(wd[0]-2, h-2, foc(0)))
	tasks := m.panelBox(foc(1), singleChip(label("Tasks"), foc(1)), wd[1], h, m.tasksView(wd[1]-2, h-2, foc(1)))
	return joinH(carries, tasks)
}

// listTitle renders panel [2]'s tab bar: one Roman-numeral chip per directory tab
// (Ⅰ … Ⅴ), no text — the active tab's path is shown by the header bar, so the
// tabs only mark position and which is active. Fixed width, so the bar always fits.
func (m AppModel) listTitle(w int) string {
	focused := m.focus == panelList
	labels := make([]string, len(m.tabs))
	for i := range m.tabs {
		labels[i] = tabNumeral(i)
	}
	return tabBar("[1]", labels, m.tab, focused)
}

// tabNumerals mark a tab's position: the Roman numerals Ⅰ … Ⅴ (Unicode ROMAN
// NUMERAL ONE..FIVE). nf-md-roman_numeral_1 is absent from Nerd Fonts, so these
// true Roman-numeral codepoints are used instead for a consistent set.
var tabNumerals = []string{
	string(rune(0x2160)), string(rune(0x2161)), string(rune(0x2162)),
	string(rune(0x2163)), string(rune(0x2164)),
}

// tabNumeral is the position mark for the idx-th (0-based) tab.
func tabNumeral(idx int) string {
	if idx >= 0 && idx < len(tabNumerals) {
		return tabNumerals[idx]
	}
	return ""
}

// carryTitle renders panel [4]'s tab bar. Like panel [2] it prefers the full
// starship tab bar and only falls back to the compact carousel when the panel is
// too narrow to fit it — see carouselChip for that narrow-panel tab strategy.
func (m AppModel) carryTitle(w int) string {
	focused := m.focus == panelCarry
	labels := []string{"Carries", "Tasks"}
	if tb := tabBar("[3]", labels, m.carryTab, focused); lipgloss.Width(tb) <= w-2 {
		return tb
	}
	return carouselChip("[3]", labels, m.carryTab, focused)
}

// carryBody renders panel [4]'s active tab.
func (m AppModel) carryBody(w, rows int) string {
	if m.carryTab == 1 { // Tasks (running + log)
		return m.tasksView(w, rows, m.focus == panelCarry)
	}
	return m.carry.view(w, rows, m.focus == panelCarry) // Carries
}

// detailTitle renders panel [2]'s title chip (Preview only).
func (m AppModel) detailTitle(w int) string {
	return singleChip("[2] Preview", m.focus == panelDetail)
}

// detailLines is panel [2]'s full preview content.
func (m AppModel) detailLines() []string { return m.preview.contentLines() }

// detailBody renders panel [2]'s preview from the scroll offset. Panels [2]/[4]
// are reference views (read while another panel has focus), so they keep their
// colour even when unfocused rather than dimming.
func (m AppModel) detailBody(w, rows int) string {
	return renderLinesFrom(m.detailLines(), m.detailScroll, w, rows)
}

// metaTitle renders panel [4]'s title chip.
func (m AppModel) metaTitle(w int) string {
	return singleChip("[4] Meta", m.focus == panelMeta)
}

// metaBody renders panel [4]'s file metadata from the scroll offset.
func (m AppModel) metaBody(w, rows int) string {
	return renderLinesFrom(m.metaContent(), m.metaScroll, w, rows)
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
	fill := max(inner-dispWidth(title), 0)
	top := bs.Render(tl) + title + bs.Render(strings.Repeat(hz, fill)+tr)

	bodyLines := strings.Split(body, "\n")
	var b strings.Builder
	b.WriteString(top + "\n")
	for r := range h - 2 {
		line := ""
		if r < len(bodyLines) {
			line = bodyLines[r]
		}
		b.WriteString(bs.Render(vt) + padDisp(line, inner) + bs.Render(vt) + "\n")
	}
	b.WriteString(bs.Render(bl + strings.Repeat(hz, inner) + br))
	return b.String()
}

// eza-style permission accents (catppuccin-mocha), matching eza's long-format
// colouring: read=yellow, write=red, execute=green. Used by the list's per-row
// Permissions column (colorPerm).
const (
	ezaYellow = "#f9e2af" // read bit / owner
	ezaRed    = "#f38ba8" // write bit
	ezaGreen  = "#a6e3a1" // execute bit / sizes
)

// statusBar is the top status row under the header. Its content is parked: the
// permissions / mtime it used to show now live in the list columns, so the row is
// intentionally blank for now — its height is kept for future content, and
// loadDirStat still caches the dir's perm/owner/disk ready to fill it.
func (m AppModel) statusBar(w int) string {
	return padDisp("", w)
}

// colorPerm paints a mode string (drwxr-xr-x) eza-style: the type char blue,
// read yellow, write red, execute green, and a bare '-' dim.
func colorPerm(perm string) string {
	typeS := lipgloss.NewStyle().Foreground(focusColor)
	rS := lipgloss.NewStyle().Foreground(lipgloss.Color(ezaYellow))
	wS := lipgloss.NewStyle().Foreground(lipgloss.Color(ezaRed))
	xS := lipgloss.NewStyle().Foreground(lipgloss.Color(ezaGreen))
	dS := lipgloss.NewStyle().Foreground(dimColor)
	var b strings.Builder
	for i, r := range perm {
		switch {
		case i == 0: // type: d / - / l / …
			b.WriteString(typeS.Render(string(r)))
		case r == 'r':
			b.WriteString(rS.Render("r"))
		case r == 'w':
			b.WriteString(wS.Render("w"))
		case r == 'x' || r == 's' || r == 't' || r == 'S' || r == 'T':
			b.WriteString(xS.Render(string(r)))
		default: // '-'
			b.WriteString(dS.Render(string(r)))
		}
	}
	return b.String()
}

// colorOwner paints "owner:group": owner in eza's user yellow, the group in
// subtext, the ':' dim. Parked with loadDirStat for the status bar's future
// content (see statusBar) — unused while the bar is blank.
func colorOwner(s string) string {
	owner := lipgloss.NewStyle().Foreground(lipgloss.Color(ezaYellow))
	group := lipgloss.NewStyle().Foreground(handColor)
	dim := lipgloss.NewStyle().Foreground(dimColor)
	if o, g, ok := strings.Cut(s, ":"); ok {
		return owner.Render(o) + dim.Render(":") + group.Render(g)
	}
	return owner.Render(s)
}

func (m AppModel) footerBar(w int) string {
	return lipgloss.NewStyle().Foreground(dimColor).
		Render(padDisp(" space menu   ? help   tab/1-5 panels   q quit", w))
}

// shortPath folds the home dir to ~ (keeps normal / separators).
func shortPath(p string) string {
	if home, err := os.UserHomeDir(); err == nil && strings.HasPrefix(p, home) {
		return "~" + strings.TrimPrefix(p, home)
	}
	return p
}
