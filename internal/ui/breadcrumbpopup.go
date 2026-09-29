package ui

import (
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// crumbLevel is one directory level in the breadcrumb chain: its absolute path
// (used for the jump) and the home-folded label shown in the popup.
type crumbLevel struct {
	path  string
	label string
}

// breadcrumbPopup lists the current tab's ancestor directories (root at top,
// current at the bottom) and jumps the active tab to any level on Enter. Ported
// from kbu's BreadcrumbPopupModel into filu's popup form. Opened by [b] on panel
// [1]; closed by Esc / b / Space. The cursor starts on the current level so
// Enter is a no-op until the user moves.
type breadcrumbPopup struct {
	anim    popupAnimator
	levels  []crumbLevel
	cursor  int
	screenW int
}

func newBreadcrumbPopup() breadcrumbPopup {
	return breadcrumbPopup{anim: newPopupAnimator("breadcrumb", popupLayerColor(1))}
}

// open builds the ancestor chain for dir and shows the popup with the cursor on
// the current (deepest) level.
func (m *breadcrumbPopup) open(dir string) tea.Cmd {
	m.levels = ancestorChain(dir)
	m.cursor = max(len(m.levels)-1, 0)
	return m.anim.open()
}

func (m *breadcrumbPopup) close() tea.Cmd     { return m.anim.close() }
func (m *breadcrumbPopup) setSize(w int)      { m.screenW = w }
func (m breadcrumbPopup) isActive() bool      { return m.anim.isActive() }
func (m breadcrumbPopup) owns() bool          { return m.anim.owns() }
func (m breadcrumbPopup) isInteractive() bool { return m.anim.isInteractive() }

func (m *breadcrumbPopup) handleTick(msg AnimTickMsg) tea.Cmd {
	if msg.Target != m.anim.target {
		return nil
	}
	return m.anim.tick()
}

// update handles a keystroke while interactive. When the returned path is
// non-empty the caller reveals it in the active tab; the popup has closed.
func (m breadcrumbPopup) update(msg tea.KeyMsg) (breadcrumbPopup, string, tea.Cmd) {
	if !m.anim.isInteractive() || len(m.levels) == 0 {
		return m, "", nil
	}
	switch msg.String() {
	case "j", "down":
		m.cursor = (m.cursor + 1) % len(m.levels)
	case "k", "up":
		m.cursor = (m.cursor - 1 + len(m.levels)) % len(m.levels)
	case "g":
		m.cursor = 0
	case "G":
		m.cursor = len(m.levels) - 1
	case "enter":
		return m, m.levels[m.cursor].path, m.anim.close()
	case "esc":
		return m, "", m.anim.close()
	}
	return m, "", nil
}

func (m breadcrumbPopup) renderPopup() string { return m.anim.renderFrame(m.renderFull()) }

func (m breadcrumbPopup) renderFull() string {
	bc := popupLayerColor(m.anim.layer)
	cursorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(baseHex)).Background(bc).Bold(true)
	hereStyle := lipgloss.NewStyle().Foreground(userColor) // current level, lavender = you-are-here

	title := " " + string(rune(0xf07c)) + " Breadcrumb" // nf-fa-folder-open
	hint := keyLegendFit([][2]string{{"j/k", "move"}, {"Enter", "jump"}, {"Esc", "close"}}, popupInnerWidth(m.screenW)-1)

	// Alignment comes from a plain 2-space gutter; the current level is flagged by
	// lavender text, not a symbol. Widths are display widths, so a glyph added
	// here later would still line up on CJK icon fonts (tdp D6).
	const gutter = "  "
	innerW := popupInnerWidth(m.screenW) // the family width (tdp F7)

	current := len(m.levels) - 1
	var rows []string
	for i, lv := range m.levels {
		label := truncPathLeft(lv.label, max(1, innerW-1-len(gutter)-1))
		body := gutter + label
		switch i {
		case m.cursor:
			padW := max(0, innerW-1-1-dispWidth(body))
			rows = append(rows, cursorStyle.Render(" "+body+strings.Repeat(" ", padW)))
		case current:
			rows = append(rows, " "+gutter+hereStyle.Render(label))
		default:
			rows = append(rows, " "+body)
		}
	}
	return drawPopupBox(bc, title, hint, rows, innerW)
}

// ancestorChain lists dir and every directory above it, root first and the given
// dir last, each labelled with the home-folded short path (matching the header
// breadcrumb's ~ fold).
func ancestorChain(dir string) []crumbLevel {
	dir = filepath.Clean(dir)
	var abs []string
	for {
		abs = append(abs, dir)
		parent := filepath.Dir(dir)
		if parent == dir { // reached the filesystem root (Dir("/") == "/")
			break
		}
		dir = parent
	}
	levels := make([]crumbLevel, len(abs))
	for i, p := range abs { // abs is current→root; store root→current
		levels[len(abs)-1-i] = crumbLevel{path: p, label: safeName(shortPath(p))}
	}
	return levels
}
