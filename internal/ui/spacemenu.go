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
}

// spaceMenu is the §A.1 contextual popup, following kbu's form (animation,
// layout, colour layer).
type spaceMenu struct {
	anim    popupAnimator
	items   []menuItem
	cursor  int
	title   string
	screenW int
	// hintRight right-aligns each row's hint to the box's right edge instead of
	// left-aligning it to a shared column. Suits a single trailing glyph (the quit
	// picker's launch icon / tab numeral), not an action description whose
	// left-aligned column reads better — so it is off for the normal Space menu.
	hintRight bool
}

func newSpaceMenu() spaceMenu {
	return spaceMenu{anim: newPopupAnimator("spacemenu", popupLayerColor(1))}
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

func (m *spaceMenu) setItems(items []menuItem, title string) {
	m.items = items
	m.title = title
	m.cursor = m.firstSelectable()
}

func (m *spaceMenu) setSize(w int)      { m.screenW = w }
func (m *spaceMenu) open() tea.Cmd      { return m.anim.open() }
func (m *spaceMenu) close() tea.Cmd     { return m.anim.close() }
func (m spaceMenu) isActive() bool      { return m.anim.isActive() }
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
	case "k", "up":
		m.cursor = m.prevSelectable(m.cursor)
	case "g":
		m.cursor = m.firstSelectable()
	case "G":
		m.cursor = m.lastSelectable()
	case "enter":
		if it := m.at(m.cursor); it != nil {
			return m, it.key, nil
		}
	case "esc", " ":
		return m, "", m.anim.close()
	default:
		for _, it := range m.items {
			if !it.separator && !it.header && it.key == msg.String() {
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
func bracketHotkey(label, key string) string {
	if label == "" || key == "" {
		return label
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
	bc := popupLayerColor(1)
	hintStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#7f849c"))
	cursorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(baseHex)).Background(bc).Bold(true)

	title := " " + m.title
	hint := " j/k move   Space close "

	const maxHintW = 44 // a longer hint wraps onto continuation lines, not widens the box
	innerW := max(lipgloss.Width(title)+4, lipgloss.Width(hint)+4)
	for _, it := range m.items {
		if it.separator {
			continue
		}
		labelW := lipgloss.Width(bracketHotkey(it.label, it.key))
		var w int
		switch {
		case it.header: // full-width label, no hint column
			w = 1 + 2 + labelW + 1
		case m.hintRight: // label + ≥2 gap + hint, hint flush to the right edge
			w = 1 + 2 + labelW + 2 + lipgloss.Width(it.hint) + 1
		default: // label padded to a 16-wide column, then the (wrappable) hint
			w = 1 + 2 + labelW + max(2, 16-labelW) + min(lipgloss.Width(it.hint), maxHintW) + 1
		}
		if w > innerW {
			innerW = w
		}
	}
	innerW = min(innerW, maxInnerWidth(m.screenW))

	const gutter = "  "
	var rows []string
	for i, it := range m.items {
		switch {
		case it.header:
			style := hintStyle
			if it.warn {
				style = lipgloss.NewStyle().Foreground(lipgloss.Color("#f38ba8")).Bold(true) // red
			}
			rows = append(rows, " "+gutter+style.Render(it.label))
			continue
		case it.separator:
			rows = append(rows, lipgloss.NewStyle().Foreground(bc).Render(strings.Repeat("─", innerW)))
			continue
		}
		labelDisplay := bracketHotkey(it.label, it.key)

		if m.hintRight {
			// Single trailing glyph right-aligned to the inner edge (the quit
			// picker). Front-pad so the glyph ends flush at innerW-1 whatever the
			// label's width, so the glyphs line up in a column on the right.
			lead := " " + gutter + labelDisplay
			line := lead + strings.Repeat(" ", max(2, innerW-1-lipgloss.Width(lead)-lipgloss.Width(it.hint)))
			if i == m.cursor {
				rows = append(rows, cursorStyle.Render(line+it.hint))
			} else {
				rows = append(rows, line+hintStyle.Render(it.hint))
			}
			continue
		}

		gap := strings.Repeat(" ", max(2, 16-lipgloss.Width(labelDisplay)))
		hintCol := 1 + len(gutter) + lipgloss.Width(labelDisplay) + lipgloss.Width(gap)
		hintW := max(innerW-1-hintCol, 8)
		indent := strings.Repeat(" ", hintCol)
		for j, hl := range wrapText(it.hint, hintW) { // long hints wrap under the label
			lead := indent
			if j == 0 {
				lead = " " + gutter + labelDisplay + gap
			}
			padW := max(0, innerW-1-lipgloss.Width(lead)-lipgloss.Width(hl))
			if i == m.cursor {
				rows = append(rows, cursorStyle.Render(lead+hl+strings.Repeat(" ", padW)))
			} else {
				rows = append(rows, lead+hintStyle.Render(hl)+strings.Repeat(" ", padW))
			}
		}
	}
	return drawPopupBox(bc, title, hint, rows, innerW)
}

// wrapText word-wraps s to width, hard-cutting any single word that alone
// overruns it so a row can never overflow the box.
func wrapText(s string, width int) []string {
	if width < 1 {
		width = 1
	}
	words := strings.Fields(s)
	if len(words) == 0 {
		return []string{""}
	}
	var lines []string
	cur := ""
	for _, w := range words {
		if lipgloss.Width(w) > width {
			w = truncate(w, width)
		}
		switch {
		case cur == "":
			cur = w
		case lipgloss.Width(cur)+1+lipgloss.Width(w) <= width:
			cur += " " + w
		default:
			lines = append(lines, cur)
			cur = w
		}
	}
	return append(lines, cur)
}
