// Package ui holds filu's Bubble Tea models. The 4-panel layout (pin,
// carry bucket, list, tabbed detail) lives in AppModel. See
// .forge/meta/IDEA.md for the target design.
package ui

import (
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

type panelID int

const (
	panelPin    panelID = iota + 1 // [1] system places + pinned
	panelList                      // [2] CWD file list (main surface)
	panelDetail                    // [3] preview / info (tabbed)
	panelCarry                     // [4] carry bucket
)

// AppModel is filu's root model.
type AppModel struct {
	width  int
	height int
	focus  panelID
	list   listModel
}

// New returns the root model, focused on the file list at the current dir.
func New() AppModel {
	dir, err := os.Getwd()
	if err != nil {
		dir = "/"
	}
	return AppModel{focus: panelList, list: newList(dir)}
}

func (m AppModel) Init() tea.Cmd { return nil }

func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "tab":
			m.focus = m.focus%4 + 1 // 1→2→3→4→1
		case "shift+tab":
			m.focus = (m.focus+2)%4 + 1 // 1→4→3→2→1
		case "1":
			m.focus = panelPin
		case "2":
			m.focus = panelList
		case "3":
			m.focus = panelDetail
		case "4":
			m.focus = panelCarry
		default:
			if m.focus == panelList {
				m.handleListKey(msg.String())
			}
		}
	}
	return m, nil
}

// handleListKey routes navigation keys to panel [2] while it is focused.
func (m *AppModel) handleListKey(key string) {
	switch key {
	case "j", "down":
		m.list.move(1)
	case "k", "up":
		m.list.move(-1)
	case "g":
		m.list.cursor = 0
	case "G":
		m.list.move(len(m.list.items))
	case "u", "ctrl+u":
		m.list.move(-m.listRows() / 2)
	case "d", "ctrl+d":
		m.list.move(m.listRows() / 2)
	case "enter":
		m.list.enter()
	case "esc":
		m.list.parent()
	}
	m.list.ensureVisible(m.listRows())
}

// listRows is how many file rows panel [2] can show:
// height − header(1) − footer(1) − top/bottom border(2) − title(1).
func (m AppModel) listRows() int {
	if r := m.height - 5; r > 0 {
		return r
	}
	return 1
}
