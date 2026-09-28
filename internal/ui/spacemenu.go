package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// popupLayerColor is kbu's popup colour-layer rule: border colour by nesting
// depth on the lavenphire→sapphire scale. Lavender is never used here (it is
// reserved for user footprint).
func popupLayerColor(layer int) lipgloss.Color {
	switch {
	case layer <= 1:
		return lipgloss.Color("#A4C0FA") // Lavenphire25
	case layer == 2:
		return lipgloss.Color("#94C3F5") // Lavenphire50
	case layer == 3:
		return lipgloss.Color("#84C5F0") // Lavenphire75
	default:
		return lipgloss.Color("#74c7ec") // Sapphire
	}
}

// menuItem is one Space-menu entry. A commit dispatches key to the focused
// panel's handler — the menu is a discoverability shell over the letter hotkeys.
type menuItem struct {
	label     string // e.g. "Carry" → rendered "[C]arry"
	key       string // hotkey dispatched on commit, e.g. "C" / "c" / "."
	hint      string // short description shown after the label
	separator bool   // non-selectable horizontal rule
	header    bool   // non-selectable region label (dim, or red when warn)
	warn      bool   // header rendered as a red warning line
	// disabled: the target exists but the action can't run right now (tdp M6).
	// The row is drawn dim with its usual hint; the cursor can rest on it, but
	// neither Enter nor its hotkey does anything.
	disabled bool
}

// disabledColor draws a row that can't run right now: dimmer than the hint and
// header text, so it reads as present but out of reach (tdp M6).
const disabledColor = lipgloss.Color("#585b70") // surface2

// spaceMenu is the Space menu (tdp K5, M2), following kbu's form (animation,
// layout, colour layer).
type spaceMenu struct {
	anim    popupAnimator
	items   []menuItem
	cursor  int
	title   string
	screenW int
	screenH int
	// top is the first row shown when the menu is taller than the screen: the
	// window follows the cursor (tdp L1 - every row stays reachable at 80 x 40).
	top int
	// hintRight right-aligns each row's hint to the box's right edge instead of
	// left-aligning it to a shared column. Suits a single trailing glyph (the quit
	// picker's launch icon / tab numeral), not an action description whose
	// left-aligned column reads better — so it is off for the normal Space menu.
	hintRight bool
	// spaceToggle marks the real Space menu: Space closes it again (tdp K5). Every
	// other spaceMenu instance is a picker opened by Enter or a hotkey, where Space
	// does nothing and only Esc (or its own flow) closes it.
	spaceToggle bool
}

func newSpaceMenu() spaceMenu {
	return spaceMenu{anim: newPopupAnimator("spacemenu", popupLayerColor(1)), spaceToggle: true}
}

// newGlobalMenu is the spaceMenu instance used as the global operation popup: a
// menu of globalActions opened from the Space menu's last row. Space does not
// close it (it is not the Space menu); Esc returns to the Space menu.
func newGlobalMenu() spaceMenu {
	return spaceMenu{anim: newPopupAnimator("globalmenu", popupLayerColor(1))}
}

// newSortMenu is a second spaceMenu instance reused as the sort picker; the
// distinct animator name keeps its ticks from colliding with the Space menu's.
func newSortMenu() spaceMenu {
	return spaceMenu{anim: newPopupAnimator("sortmenu", popupLayerColor(1))}
}

// newGotoMenu is a spaceMenu instance reused as the Goto picker: a root
// {Pinned, Search} choice, then (Pinned) a drill-down list of pinned dirs.
func newGotoMenu() spaceMenu {
	return spaceMenu{anim: newPopupAnimator("gotomenu", popupLayerColor(1))}
}

// newSearchMenu is a spaceMenu instance reused as the Search chooser: a flat
// {filename, content} pick that then opens the finder in that mode.
func newSearchMenu() spaceMenu {
	return spaceMenu{anim: newPopupAnimator("searchmenu", popupLayerColor(1))}
}

// newQuitMenu is a third spaceMenu instance reused as the cd-on-quit picker. Its
// hints are single glyphs (launch icon / tab numeral), right-aligned so they line
// up in a column on the right edge whatever the path labels' widths.
func newQuitMenu() spaceMenu {
	return spaceMenu{anim: newPopupAnimator("quitmenu", popupLayerColor(1)), hintRight: true}
}

// newOpenWithMenu is a fourth spaceMenu instance reused as the [o]pen picker
// (Default + the apps configured in config.yaml's open_with).
func newOpenWithMenu() spaceMenu {
	return spaceMenu{anim: newPopupAnimator("openwithmenu", popupLayerColor(1))}
}

// newOpenInMenu is a spaceMenu instance reused as the Favorites tab's "Open dir
// in…" picker: New tab (unless the tab count is at maxTabs) or one of the open
// panel [1] tabs.
func newOpenInMenu() spaceMenu {
	return spaceMenu{anim: newPopupAnimator("openinmenu", popupLayerColor(1))}
}

func (m *spaceMenu) setItems(items []menuItem, title string) {
	m.items = items
	m.title = title
	m.cursor = m.firstSelectable()
	m.top = 0
	m.scroll()
}

func (m *spaceMenu) setSize(w, h int) {
	m.screenW, m.screenH = w, h
	m.scroll()
}

// menuChrome is the rows around the menu's items: two borders, two padding
// rows, and a row of screen above and below so the box never touches the edge.
const menuChrome = 6

// visible is how many item rows fit on screen (all of them when the height is
// not known yet).
func (m spaceMenu) visible() int {
	if m.screenH <= 0 {
		return max(len(m.items), 1)
	}
	return max(m.screenH-menuChrome, 3)
}

// scroll moves the window just enough to keep the cursor row in it.
func (m *spaceMenu) scroll() {
	vis := m.visible()
	if m.cursor < m.top {
		m.top = m.cursor
	}
	if m.cursor >= m.top+vis {
		m.top = m.cursor - vis + 1
	}
	m.top = max(0, min(m.top, max(0, len(m.items)-vis)))
	if m.top > 0 && m.top <= len(m.items) && m.cursor == m.firstSelectable() {
		m.top = 0 // at the first stop, show the region header above it too
	}
}
func (m *spaceMenu) open() tea.Cmd      { return m.anim.open() }
func (m *spaceMenu) close() tea.Cmd     { return m.anim.close() }
func (m spaceMenu) isActive() bool      { return m.anim.isActive() }
func (m spaceMenu) owns() bool          { return m.anim.owns() }
func (m spaceMenu) isInteractive() bool { return m.anim.isInteractive() }
func (m *spaceMenu) handleTick(msg AnimTickMsg) tea.Cmd {
	if msg.Target != m.anim.target {
		return nil
	}
	return m.anim.tick()
}

// update handles a keystroke while the menu is interactive. The returned string
// is the committed hotkey (empty when nothing committed) — the caller dispatches
// it to the focused panel and closes the menu.
func (m spaceMenu) update(msg tea.KeyMsg) (spaceMenu, string, tea.Cmd) {
	if !m.anim.isInteractive() {
		return m, "", nil
	}
	switch msg.String() {
	case "j", "down":
		m.cursor = m.nextSelectable(m.cursor)
		m.scroll()
	case "k", "up":
		m.cursor = m.prevSelectable(m.cursor)
		m.scroll()
	case "g":
		m.cursor = m.firstSelectable()
		m.scroll()
	case "G":
		m.cursor = m.lastSelectable()
		m.top = len(m.items) // bottom out, so the rows after the last stop show too
		m.scroll()
	case "enter":
		if it := m.at(m.cursor); it != nil && !it.disabled {
			return m, it.key, nil
		}
	case "esc":
		return m, "", m.anim.close()
	case " ":
		if m.spaceToggle {
			return m, "", m.anim.close()
		}
	default:
		for _, it := range m.items {
			if !it.separator && !it.header && it.key == msg.String() {
				if it.disabled { // the hotkey of a dimmed row does nothing (tdp M6)
					return m, "", nil
				}
				return m, it.key, nil
			}
		}
	}
	return m, "", nil
}

func (m spaceMenu) renderPopup() string { return m.anim.renderFrame(m.renderFull()) }

func (m spaceMenu) at(i int) *menuItem {
	if i < 0 || i >= len(m.items) || m.items[i].separator || m.items[i].header {
		return nil
	}
	return &m.items[i]
}

func (m spaceMenu) firstSelectable() int {
	for i, it := range m.items {
		if !it.separator && !it.header {
			return i
		}
	}
	return 0
}

func (m spaceMenu) lastSelectable() int {
	for i := len(m.items) - 1; i >= 0; i-- {
		if !m.items[i].separator && !m.items[i].header {
			return i
		}
	}
	return 0
}

func (m spaceMenu) nextSelectable(from int) int {
	n := len(m.items)
	for step := 1; step <= n; step++ {
		idx := (from + step) % n
		if !m.items[idx].separator && !m.items[idx].header {
			return idx
		}
	}
	return from
}

func (m spaceMenu) prevSelectable(from int) int {
	n := len(m.items)
	for step := 1; step <= n; step++ {
		idx := (from - step + n) % n
		if !m.items[idx].separator && !m.items[idx].header {
			return idx
		}
	}
	return from
}

// bracketHotkey wraps the hotkey inside the label (vim-help style), preserving
// the key's case so filu's C/c distinction reads correctly. A key that appears in
// the label is bracketed in place — a single letter ("[S]ort") or a chord that is
// a substring ("[go]to"). A single key not in the label is prefixed ("[/] Search")
// so it stays visible; a multi-char key not in the label (e.g. "Esc") renders plain.
// A digit key is ALWAYS a prefix: it is an ordinal, not a mnemonic — bracketing
// it in place would mangle an arbitrary label that happens to contain the digit
// (a quit-menu path like 432hz must never render as 4[3]2hz).
func bracketHotkey(label, key string) string {
	if label == "" || key == "" {
		return label
	}
	if len(key) == 1 && key[0] >= '0' && key[0] <= '9' {
		return "[" + key + "] " + label
	}
	idx := strings.Index(strings.ToUpper(label), strings.ToUpper(key))
	if idx < 0 {
		if len(key) == 1 {
			return "[" + key + "] " + label
		}
		return label
	}
	return label[:idx] + "[" + key + "]" + label[idx+len(key):]
}

// renderFull draws the popup box, ported from kbu's panel2menu renderFullPopup:
// title embedded in the top border, hint in the bottom border, rows of
// "[K]label   hint", cursor row reverse-highlighted.
func (m spaceMenu) renderFull() string {
	bc := popupLayerColor(m.anim.layer)
	hintStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#7f849c"))
	dimStyle := lipgloss.NewStyle().Foreground(dimColor)
	cursorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(baseHex)).Background(bc).Bold(true)

	title := " " + m.title
	hint := " j/k move · Enter run · Esc close "

	// One line per row (the family form): labels in a column as wide as the
	// widest label, each hint on the same line after it. The box widens to fit
	// the longest hint up to the screen cap; only past that is a hint cut.
	const gutter = "  "
	labelW, hintW := 0, 0
	for _, it := range m.items {
		if it.separator || it.header {
			continue
		}
		labelW = max(labelW, lipgloss.Width(bracketHotkey(it.label, it.key)))
		hintW = max(hintW, lipgloss.Width(it.hint))
	}
	labelCol := labelW + 2
	innerW := max(lipgloss.Width(title)+4, lipgloss.Width(hint)+4, 1+len(gutter)+labelCol+hintW+1)
	for _, it := range m.items {
		if it.header {
			innerW = max(innerW, 1+len(gutter)+lipgloss.Width(it.label)+1)
		}
	}
	innerW = min(innerW, maxInnerWidth(m.screenW))

	rows := make([]string, 0, len(m.items))
	for i, it := range m.items {
		switch {
		case it.header:
			style := hintStyle
			if it.warn {
				style = lipgloss.NewStyle().Foreground(lipgloss.Color("#f38ba8")).Bold(true) // red
			}
			rows = append(rows, " "+gutter+style.Render(truncate(it.label, innerW-1-len(gutter))))
			continue
		case it.separator: // a dim rule between regions (tdp M2), inset one cell from each side
			rows = append(rows, " "+dimStyle.Render(strings.Repeat("─", max(innerW-2, 0)))+" ")
			continue
		}
		labelDisplay := bracketHotkey(it.label, it.key)
		rowLabel, rowHint, rowCursor := lipgloss.NewStyle(), hintStyle, cursorStyle
		if it.disabled { // label and hint both dim; the cursor bar greys out too
			rowLabel = lipgloss.NewStyle().Foreground(disabledColor)
			rowHint = rowLabel
			rowCursor = lipgloss.NewStyle().Foreground(lipgloss.Color(baseHex)).Background(disabledColor).Bold(true)
		}

		lead := " " + gutter + labelDisplay
		var gap string
		if m.hintRight {
			// A single trailing glyph right-aligned to the inner edge (the quit
			// picker), so the glyphs line up in a column whatever the labels.
			gap = strings.Repeat(" ", max(2, innerW-1-lipgloss.Width(lead)-lipgloss.Width(it.hint)))
		} else {
			gap = strings.Repeat(" ", max(2, labelCol-lipgloss.Width(labelDisplay)))
		}
		h := truncate(it.hint, max(innerW-1-lipgloss.Width(lead)-len(gap), 1))
		pad := strings.Repeat(" ", max(0, innerW-lipgloss.Width(lead)-len(gap)-lipgloss.Width(h)))
		if i == m.cursor {
			rows = append(rows, rowCursor.Render(lead+gap+h+pad))
		} else {
			rows = append(rows, rowLabel.Render(lead)+gap+rowHint.Render(h)+pad)
		}
	}

	// Taller than the screen: show the window that holds the cursor (scroll).
	if vis := m.visible(); len(rows) > vis {
		top := max(0, min(m.top, len(rows)-vis))
		rows = rows[top : top+vis]
	}
	return drawPopupBox(bc, title, hint, rows, innerW)
}
