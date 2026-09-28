package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// modeKey is one key of the yank viewport's selection mode (tdp K11): what to
// press, what it does, and the keystrokes that run it from the key list.
type modeKey struct {
	keys string   // as shown in the list and the help: "j ↓"
	desc string   //
	run  []string // keystrokes handed to the viewport: {"g", "g"} for gg
}

// selectKeys is the one table of the selection mode: its key list (Space) and
// its help (?) are both built from it, and the viewport's own key reference
// takes its movement rows, so the three cannot disagree (tdp K11, M3).
var selectKeys = []modeKey{
	{"h ←", "move left", []string{"h"}},
	{"l →", "move right", []string{"l"}},
	{"j ↓", "move down", []string{"j"}},
	{"k ↑", "move up", []string{"k"}},
	{"0", "start of the line", []string{"0"}},
	{"$", "end of the line", []string{"$"}},
	{"gg", "top", []string{"g", "g"}},
	{"G", "bottom", []string{"G"}},
	{"u", "half a page up", []string{"u"}},
	{"d", "half a page down", []string{"d"}},
	{"y", "copy the selection", []string{"y"}},
	{"v Esc", "leave the selection", []string{"v"}},
}

// isMoveKey reports a row that only moves the cursor — the rows the viewport
// shares with the mode.
func (k modeKey) isMoveKey() bool { return k.run[0] != "y" && k.run[0] != "v" }

// trigger is the key that runs the row from the key list: its first keystroke.
func (k modeKey) trigger() string { return k.run[0] }

// selectHelpRows is the selection mode's help (?): the table, then the keys
// that open the list and leave.
func selectHelpRows() []helpRow {
	var rows []helpRow
	for _, k := range selectKeys {
		rows = append(rows, helpRow{key: k.keys, desc: k.desc})
	}
	return append(rows,
		helpRow{key: "Space", desc: "list these keys, and run one from the list"},
		helpRow{key: "?", desc: "these keys"},
		helpRow{key: "q", desc: "quit — pick a directory to cd to"})
}

// keyMsgFor turns a table keystroke back into the key message the viewport
// would have received.
func keyMsgFor(k string) tea.KeyMsg {
	if k == "esc" {
		return tea.KeyMsg{Type: tea.KeyEsc}
	}
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(k)}
}

// modeList is the selection mode's key list (tdp K11), opened with Space while
// selecting and stacked on the viewport. The mode's keys overlap navigation
// (h j k l, g G, u d), so only the arrow keys move here; every other key runs
// its row, as does Enter on the highlighted row, and running a row closes the
// list. Space or Esc closes it without running anything.
type modeList struct {
	anim    popupAnimator
	cursor  int
	screenW int
}

func newModeList() modeList {
	return modeList{anim: newPopupAnimator("modelist", popupLayerColor(1))}
}

func (m *modeList) open() tea.Cmd {
	m.cursor = 0
	return m.anim.open()
}

func (m *modeList) setSize(w int)      { m.screenW = w }
func (m modeList) isActive() bool      { return m.anim.isActive() }
func (m modeList) owns() bool          { return m.anim.owns() }
func (m modeList) isInteractive() bool { return m.anim.isInteractive() }
func (m modeList) renderPopup() string { return m.anim.renderFrame(m.renderFull()) }
func (m *modeList) handleTick(msg AnimTickMsg) tea.Cmd {
	if msg.Target != m.anim.target {
		return nil
	}
	return m.anim.tick()
}

// update returns the keystrokes of the row to run (nil when none ran).
func (m modeList) update(msg tea.KeyMsg) (modeList, []string, tea.Cmd) {
	if !m.anim.isInteractive() {
		return m, nil, nil
	}
	switch msg.Type {
	case tea.KeyUp:
		m.cursor = (m.cursor - 1 + len(selectKeys)) % len(selectKeys)
		return m, nil, nil
	case tea.KeyDown:
		m.cursor = (m.cursor + 1) % len(selectKeys)
		return m, nil, nil
	case tea.KeyEnter:
		return m, selectKeys[m.cursor].run, m.anim.close()
	case tea.KeyEsc:
		return m, nil, m.anim.close()
	}
	if msg.String() == " " { // Space toggles the list shut, whichever form the terminal sends
		return m, nil, m.anim.close()
	}
	for _, k := range selectKeys {
		if k.trigger() == msg.String() {
			return m, k.run, m.anim.close()
		}
	}
	return m, nil, nil
}

// modeListHint is the list's bottom border.
const modeListHint = " ↑↓ move · Enter or its key run · Space close "

func (m modeList) renderFull() string {
	bc := popupLayerColor(m.anim.layer)
	keyStyle := lipgloss.NewStyle().Foreground(bc).Bold(true)
	descStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#7f849c"))
	cursorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(baseHex)).Background(bc).Bold(true)

	keyW, descW := 0, 0
	for _, k := range selectKeys {
		keyW = max(keyW, lipgloss.Width(k.keys))
		descW = max(descW, lipgloss.Width(k.desc))
	}
	title := " Selection keys"
	innerW := min(max(lipgloss.Width(title)+4, lipgloss.Width(modeListHint)+4, 2+keyW+2+descW+2), maxInnerWidth(m.screenW))
	rows := make([]string, 0, len(selectKeys))
	for i, k := range selectKeys {
		key := k.keys + strings.Repeat(" ", keyW-lipgloss.Width(k.keys))
		pad := strings.Repeat(" ", max(0, innerW-(2+keyW+2+lipgloss.Width(k.desc))))
		if i == m.cursor {
			rows = append(rows, cursorStyle.Render("  "+key+"  "+k.desc+pad))
		} else {
			rows = append(rows, "  "+keyStyle.Render(key)+"  "+descStyle.Render(k.desc)+pad)
		}
	}
	return drawPopupBox(bc, title, modeListHint, rows, innerW)
}
