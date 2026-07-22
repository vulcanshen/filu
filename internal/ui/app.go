// Package ui holds filu's Bubble Tea models. The 4-panel layout (pin,
// carry bucket, list, tabbed detail) lives in AppModel. See
// .forge/meta/IDEA.md for the target design.
package ui

import (
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
)

type panelID int

const (
	panelPin    panelID = iota + 1 // [1] system places + pinned
	panelList                      // [2] CWD file list (main surface)
	panelDetail                    // [3] preview / info (tabbed)
	panelCarry                     // [4] carry bucket
)

// detailTab selects panel [3]'s active tab.
type detailTab int

const (
	tabPreview detailTab = iota
	tabInfo
)

// AppModel is filu's root model.
type AppModel struct {
	width   int
	height  int
	focus   panelID
	detail  detailTab
	tabs    [3]listModel // panel [2]'s fixed 3 directory tabs
	tab     int          // active tab index
	preview previewModel
	places  placesModel
	carry   carryModel
}

// New returns the root model, focused on the file list. All 3 tabs open at the
// current dir (they diverge as the user navigates each).
func New() AppModel {
	dir, err := os.Getwd()
	if err != nil {
		dir = "/"
	}
	m := AppModel{focus: panelList, places: newPlaces()}
	for i := range m.tabs {
		m.tabs[i] = newList(dir)
	}
	m.refreshPreview()
	return m
}

// cur returns a pointer to the active directory tab.
func (m *AppModel) cur() *listModel { return &m.tabs[m.tab] }

// active returns the active tab by value (read-only paths).
func (m AppModel) active() listModel { return m.tabs[m.tab] }

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
			case panelDetail:
				m.handleDetailKey(msg.String())
			}
		}
	}
	return m, nil
}

// handleListKey routes navigation keys to panel [2] while it is focused.
func (m *AppModel) handleListKey(key string) {
	l := m.cur()
	switch key {
	case "j", "down":
		l.move(1)
	case "k", "up":
		l.move(-1)
	case "g":
		l.cursor = 0
	case "G":
		l.move(len(l.items))
	case "u", "ctrl+u":
		l.move(-m.listRows() / 2)
	case "d", "ctrl+d":
		l.move(m.listRows() / 2)
	case "enter":
		l.enter()
	case "esc":
		l.parent()
	case "l", "right":
		m.tab = (m.tab + 1) % len(m.tabs)
	case "h", "left":
		m.tab = (m.tab + len(m.tabs) - 1) % len(m.tabs)
	case "C": // carry: toggle cursor item in the bucket
		if it := l.cursorItem(); it.name != "" {
			m.carry.toggle(filepath.Join(l.dir, it.name))
		}
	case "c": // provisional: land carried items here as copy
		m.carry.land(l.dir, false)
		l.reload()
	case "x": // provisional: land carried items here as move
		m.carry.land(l.dir, true)
		l.reload()
	}
	m.cur().ensureVisible(m.listRows())
	m.refreshPreview()
}

// handleDetailKey routes keys to panel [3] while it is focused (h/l swap tab).
func (m *AppModel) handleDetailKey(key string) {
	switch key {
	case "h", "left", "l", "right":
		if m.detail == tabPreview {
			m.detail = tabInfo
		} else {
			m.detail = tabPreview
		}
	}
}

// refreshPreview reloads panel [3]'s preview for the current cursor item.
func (m *AppModel) refreshPreview() {
	l := m.cur()
	m.preview = loadPreview(l.cursorItem(), l.dir)
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
	l := m.cur()
	l.dir = dir
	l.cursor, l.offset = 0, 0
	l.reload()
	m.focus = panelList
	l.ensureVisible(m.listRows())
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
