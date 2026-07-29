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
	if m.searchMenu.isActive() {
		out = overlay.Composite(m.searchMenu.renderPopup(), out, overlay.Center, overlay.Center, 0, 0)
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
	case panelMarks:
		return m.zoomMarksView(w, midH)
	default:
		return m.normalMiddle(w, midH)
	}
}

// normalMiddle is the default layout:
//
//	| [1] list | [2] |    top    [1] list | [2] preview at 2:1
//	| [1] list | [2] |
//	| [3] [3] [3]    |    bottom [3] Marks | Tasks tabs, full width
//
// The top row is 2:1 (the info-rich list earns the width); the bottom is one
// full-width tabbed panel.
func (m AppModel) normalMiddle(w, midH int) string {
	listFocus := m.focus == panelList
	if w < 72 { // too narrow for the grid; the list alone (Space menu Zoom is the escape hatch)
		return m.panelBoxHint(listFocus, m.listTitle(w), listNavHint(listFocus), w, midH, m.active().view(w-2, midH-2, listFocus, m.marks.inBucket(), m.places.pinnedSet()))
	}
	topH := midH * 2 / 3
	botH := midH - topH

	// Top row: list | preview, 2:1.
	listW := w * 2 / 3
	previewW := w - listW
	list := m.panelBoxHint(listFocus, m.listTitle(listW), listNavHint(listFocus), listW, topH, m.active().view(listW-2, topH-2, listFocus, m.marks.inBucket(), m.places.pinnedSet()))
	preview := m.panelBox(m.focus == panelDetail, m.detailTitle(previewW), previewW, topH, m.detailBody(previewW-2, topH-2))
	topRow := joinH(list, preview)

	// Bottom row: [3] Marks | Tasks, one full-width tabbed panel.
	panel3Focus := m.focus == panelMarks
	body, hint := m.marksBody(w-2, botH-2, panel3Focus)
	botRow := m.panelBoxHint(panel3Focus, m.marksTitle(), hint, w, botH, body)

	return joinV(topRow, botRow)
}

// marksTitle renders panel [3]'s Marks | Tasks | Favorites tab bar (full width, so
// it always fits — no carousel fallback).
func (m AppModel) marksTitle() string {
	return tabBar("[3]", []string{"Marks", "Tasks", "Favorites"}, m.marksTab, m.focus == panelMarks)
}

// tabDirIndex maps each open list tab's directory (cleaned) to its tab index, so
// the Favorites tab can badge a favorite that a tab currently has open with that
// tab's Roman numeral instead of the star. Walked in reverse so the lowest tab
// index wins when two tabs share a directory.
func (m AppModel) tabDirIndex() map[string]int {
	out := make(map[string]int, len(m.tabs))
	for i := len(m.tabs) - 1; i >= 0; i-- {
		out[cleanDir(m.tabs[i].dir)] = i
	}
	return out
}

// marksBody renders panel [3]'s active tab — the Marks bucket (with the marks
// workflow hint), the Tasks land log, or the Favorites list.
func (m AppModel) marksBody(w, rows int, focused bool) (body, hint string) {
	switch m.marksTab {
	case 1:
		return m.tasksView(w, rows, focused), ""
	case 2:
		return m.places.view(w, rows, focused, m.tabDirIndex()), favoritesHint()
	}
	return m.marks.view(w, rows, focused), marksHint()
}

// zoomListView (panel [1] zoom): the directory tabs expanded 1:1:1 full-screen.
func (m AppModel) zoomListView(w, midH int) string {
	return m.expandedListTabs(w, midH)
}

// expandedListTabs lays panel [1]'s directory tabs out as equal-width columns;
// the active tab is the focused column when [1] holds focus.
func (m AppModel) expandedListTabs(w, h int) string {
	widths := splitN(w, len(m.tabs))
	cols := make([]string, len(m.tabs))
	carried := m.marks.inBucket()
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

// zoomDetailView (panel [2] zoom): the preview full-screen.
func (m AppModel) zoomDetailView(w, midH int) string {
	return m.panelBox(true, singleChip("Preview", true), w, midH,
		renderLinesFrom(m.preview.contentLines(), m.detailScroll, w-2, midH-2))
}

// zoomMarksView (panel [3] zoom): the Marks | Tasks panel full-screen.
func (m AppModel) zoomMarksView(w, midH int) string {
	body, hint := m.marksBody(w-2, midH-2, true)
	return m.panelBoxHint(true, m.marksTitle(), hint, w, midH, body)
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

// detailTitle renders panel [2]'s title chip (Preview only).
func (m AppModel) detailTitle(w int) string {
	return singleChip("[2] Preview", m.focus == panelDetail)
}

// detailLines is panel [2]'s full preview content.
func (m AppModel) detailLines() []string { return m.preview.contentLines() }

// detailBody renders panel [2]'s preview from the scroll offset. The preview is a
// reference view (read while another panel has focus), so it keeps its colour even
// when unfocused rather than dimming.
func (m AppModel) detailBody(w, rows int) string {
	return renderLinesFrom(m.detailLines(), m.detailScroll, w, rows)
}

// panelBox draws a bordered panel with the title embedded in the top border
// (kbu style). Focused = double border + blue, else rounded + dim.
func (m AppModel) panelBox(focused bool, title string, w, h int, body string) string {
	return m.panelBoxHint(focused, title, "", w, h, body)
}

// panelBoxHint is panelBox with a key legend embedded in the bottom border
// (kbu popup form: title on top, hint on the bottom). hint is pre-styled chrome;
// "" leaves the bottom edge plain.
func (m AppModel) panelBoxHint(focused bool, title, hint string, w, h int, body string) string {
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
	if hint == "" {
		b.WriteString(bs.Render(bl + strings.Repeat(hz, inner) + br))
	} else {
		if dispWidth(hint) > inner {
			hint = truncate(hint, inner)
		}
		botFill := max(inner-dispWidth(hint), 0)
		b.WriteString(bs.Render(bl) + hint + bs.Render(strings.Repeat(hz, botFill)+br))
	}
	return b.String()
}

// keyLegend renders a "key desc   key desc …" hint line — each key in the chrome
// blue, each description dim — wrapped with a space on both sides. Shared by the
// list panel's bottom-border hint and the footer.
func keyLegend(pairs [][2]string) string {
	keyStyle := lipgloss.NewStyle().Foreground(focusColor)
	descStyle := lipgloss.NewStyle().Foreground(dimColor)
	parts := make([]string, len(pairs))
	for i, p := range pairs {
		parts[i] = keyStyle.Render(p[0]) + " " + descStyle.Render(p[1])
	}
	return " " + strings.Join(parts, "   ") + " "
}

// listNavHint is the key legend shown in the focused list panel's bottom border:
// the core open-model navigation keys (Enter enters a dir, Esc goes up, j/k move,
// d/u half-page). "" when the list is unfocused so an idle panel keeps a clean edge.
func listNavHint(focused bool) string {
	if !focused {
		return ""
	}
	return keyLegend([][2]string{
		{"enter", "into"}, {"esc", "back"}, {"j/k", "move"}, {"d/u", "page"},
	})
}

// marksHint is the always-shown key legend on the Marks panel's bottom border: the
// marks workflow — mark a file, then copy/move the set here. These keys fire on the
// LIST panel; the legend lives on Marks as a reference so it is visible while you
// mark from the list.
func marksHint() string {
	return keyLegend([][2]string{{"m", "mark"}, {"c", "copy"}, {"v", "move"}})
}

// favoritesHint is the Favorites tab's bottom-border legend: the one action that
// lives on this tab. `f` on the LIST still creates/removes favorites; here D
// prunes the highlighted one (jumping to a favorite lives in the Goto picker).
func favoritesHint() string {
	return keyLegend([][2]string{{"D", "remove"}})
}

// eza-style permission accents (catppuccin-mocha), matching eza's long-format
// colouring: read=yellow, write=red, execute=green. Used by the list's per-row
// Permissions column (colorPerm).
const (
	ezaYellow = "#f9e2af" // read bit / owner
	ezaRed    = "#f38ba8" // write bit
	ezaGreen  = "#a6e3a1" // execute bit / sizes
)

// statusBar is the top status row under the header: the directory filu was
// launched from, marked with the launch glyph — a fixed reference (where a
// cd-on-quit "LaunchDir" returns to), rendered recessively.
func (m AppModel) statusBar(w int) string {
	icon := lipgloss.NewStyle().Foreground(userColor).Render(iconCWD) // lavender: the launch/return anchor
	avail := max(1, w-dispWidth(iconCWD)-3)                           // icon + gap + trailing margin
	path := lipgloss.NewStyle().Foreground(dimColor).Render(fitPath(m.launchDir, avail))
	return padDispRight(icon+" "+path+" ", w) // right-aligned, one-cell right margin
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

// colorSize paints a file's size eza color-scale style — warmer as it grows:
// green under 1 MiB, yellow under 1 GiB, peach under 1 TiB, red beyond. A
// directory has no size (filu never recurses to compute one), so it shows a dim
// dash placeholder instead.
func colorSize(it fileItem) string {
	if it.isDir {
		return lipgloss.NewStyle().Foreground(dimColor).Render("-")
	}
	c := lipgloss.Color(ezaGreen)
	switch {
	case it.size >= 1<<40:
		c = lipgloss.Color(ezaRed)
	case it.size >= 1<<30:
		c = lipgloss.Color("#fab387") // catppuccin peach
	case it.size >= 1<<20:
		c = lipgloss.Color(ezaYellow)
	}
	return lipgloss.NewStyle().Foreground(c).Render(compactSize(it.size))
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
	return padDisp(keyLegend([][2]string{
		{"space", "menu"}, {"?", "help"}, {"tab/1-3", "panels"}, {"q", "quit"},
	}), w)
}

// shortPath folds the home dir to ~ (keeps normal / separators).
func shortPath(p string) string {
	if home, err := os.UserHomeDir(); err == nil && strings.HasPrefix(p, home) {
		return "~" + strings.TrimPrefix(p, home)
	}
	return p
}
