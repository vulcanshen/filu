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
	width   int
	height  int
	focus   panelID
	list    listModel
	preview previewModel
	places  placesModel
}

// New returns the root model, focused on the file list at the current dir.
func New() AppModel {
	dir, err := os.Getwd()
	if err != nil {
		dir = "/"
	}
	m := AppModel{focus: panelList, list: newList(dir), places: newPlaces()}
	m.refreshPreview()
	return m
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
			switch m.focus {
			case panelList:
				m.handleListKey(msg.String())
			case panelPin:
				m.handlePinKey(msg.String())
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
	m.refreshPreview()
}

// refreshPreview reloads panel [3]'s preview for the current cursor item.
func (m *AppModel) refreshPreview() {
	var it fileItem
	if m.list.cursor < len(m.list.items) {
		it = m.list.items[m.list.cursor]
	}
	m.preview = loadPreview(it, m.list.dir)
}

// handlePinKey routes navigation keys to panel [1] while it is focused.
func (m *AppModel) handlePinKey(key string) {
	switch key {
	case "j", "down":
		m.places.move(1)
	case "k", "up":
		m.places.move(-1)
	case "enter":
		if p, ok := m.places.current(); ok {
			m.navigateTo(p.path)
		}
	}
}

// navigateTo points panel [2] at dir and moves focus there.
func (m *AppModel) navigateTo(dir string) {
	m.list.dir = dir
	m.list.cursor, m.list.offset = 0, 0
	m.list.reload()
	m.focus = panelList
	m.list.ensureVisible(m.listRows())
	m.refreshPreview()
}

// listRows is how many file rows panel [2] can show:
// height − header(1) − footer(1) − top/bottom border(2) − title(1).
func (m AppModel) listRows() int {
	if r := m.height - 5; r > 0 {
		return r
	}
	return 1
}
