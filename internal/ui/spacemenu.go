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
	header    bool   // non-selectable dim region label
}

// spaceMenu is the §A.1 contextual popup, following kbu's form (animation,
// layout, colour layer).
type spaceMenu struct {
	anim    popupAnimator
	items   []menuItem
	cursor  int
	title   string
	screenW int
}

func newSpaceMenu() spaceMenu {
	return spaceMenu{anim: newPopupAnimator("spacemenu", popupLayerColor(1))}
}

// newSortMenu is a second spaceMenu instance reused as the sort picker; the
// distinct animator name keeps its ticks from colliding with the Space menu's.
func newSortMenu() spaceMenu {
	return spaceMenu{anim: newPopupAnimator("sortmenu", popupLayerColor(1))}
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
// the key's case so filu's C/c distinction reads correctly. When the key isn't a
// letter of the label (punctuation, or a mismatch) it is prefixed instead so the
// key is always visible. Multi-char keys (e.g. "Esc") render plain.
func bracketHotkey(label, key string) string {
	if label == "" || key == "" || len(key) > 1 {
		return label
	}
	idx := strings.Index(strings.ToUpper(label), strings.ToUpper(key))
	if idx < 0 {
		return "[" + key + "] " + label
	}
	return label[:idx] + "[" + key + "]" + label[idx+1:]
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

	innerW := max(lipgloss.Width(title)+4, lipgloss.Width(hint)+4)
	for _, it := range m.items {
		labelW := lipgloss.Width(bracketHotkey(it.label, it.key))
		gap := max(2, 16-labelW)
		if w := 1 + 2 + labelW + gap + lipgloss.Width(it.hint) + 1; w > innerW {
			innerW = w
		}
	}
	innerW = min(innerW, maxInnerWidth(m.screenW))

	const gutter = "  "
	var rows []string
	for i, it := range m.items {
		switch {
		case it.header:
			rows = append(rows, " "+gutter+hintStyle.Render(it.label))
			continue
		case it.separator:
			rows = append(rows, lipgloss.NewStyle().Foreground(bc).Render(strings.Repeat("─", innerW)))
			continue
		}
		labelDisplay := bracketHotkey(it.label, it.key)
		gap := strings.Repeat(" ", max(2, 16-lipgloss.Width(labelDisplay)))
		body := " " + gutter + labelDisplay + gap + it.hint
		padW := max(0, innerW-1-lipgloss.Width(body))
		if i == m.cursor {
			rows = append(rows, cursorStyle.Render(body+strings.Repeat(" ", padW)))
			continue
		}
		rows = append(rows, " "+gutter+labelDisplay+gap+hintStyle.Render(it.hint)+strings.Repeat(" ", padW))
	}
	return drawPopupBox(bc, title, hint, rows, innerW)
}
