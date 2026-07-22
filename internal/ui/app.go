// Package ui holds filu's Bubble Tea models. The 4-panel layout (pin,
// carry bucket, list, tabbed detail) lives in AppModel. See
// .forge/meta/IDEA.md for the target design.
package ui

import (
	"os"
	"path/filepath"
	"strings"

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

// inputKind is the pending footer text input, if any.
type inputKind int

const (
	inputNone inputKind = iota
	inputRename
	inputAdd
)

// inputState is a single-line input rendered in the footer (rename / add).
type inputState struct {
	kind   inputKind
	prompt string
	buffer string
	target string // original name (rename)
}

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
	input   inputState
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
		if m.input.kind != inputNone {
			m.handleInputKey(msg)
			return m, nil
		}
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
	case "P": // pin: toggle the cursor dir into [1] Pinned
		if it := l.cursorItem(); it.isDir {
			m.places.togglePin(filepath.Join(l.dir, it.name))
		}
	case ".": // toggle hidden files in this tab
		l.showHidden = !l.showHidden
		l.reload()
	case "D": // delete: move cursor item to the system trash
		if it := l.cursorItem(); it.name != "" {
			_ = moveToTrash(filepath.Join(l.dir, it.name))
			l.reload()
		}
	case "R": // rename cursor item (footer input)
		if it := l.cursorItem(); it.name != "" {
			m.input = inputState{kind: inputRename, prompt: "Rename", buffer: it.name, target: it.name}
		}
	case "A": // add file/dir — lazyvim style: trailing / = dir (footer input)
		m.input = inputState{kind: inputAdd, prompt: "New (末尾 / = 目錄)"}
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

// handleInputKey feeds keystrokes to the footer text input.
func (m *AppModel) handleInputKey(msg tea.KeyMsg) {
	switch msg.Type {
	case tea.KeyEsc:
		m.input = inputState{}
	case tea.KeyEnter:
		m.commitInput()
	case tea.KeyBackspace:
		if r := []rune(m.input.buffer); len(r) > 0 {
			m.input.buffer = string(r[:len(r)-1])
		}
	case tea.KeySpace:
		m.input.buffer += " "
	case tea.KeyRunes:
		m.input.buffer += string(msg.Runes)
	}
}

// commitInput performs the pending rename / add and closes the input.
func (m *AppModel) commitInput() {
	name := strings.TrimSpace(m.input.buffer)
	kind, target := m.input.kind, m.input.target
	m.input = inputState{}
	if name == "" {
		return
	}
	l := m.cur()
	switch kind {
	case inputRename:
		if target != "" {
			_ = os.Rename(filepath.Join(l.dir, target), filepath.Join(l.dir, name))
		}
	case inputAdd:
		full := filepath.Join(l.dir, name)
		if strings.HasSuffix(name, "/") {
			_ = os.MkdirAll(full, 0o755)
		} else {
			_ = os.MkdirAll(filepath.Dir(full), 0o755)
			if f, err := os.OpenFile(full, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644); err == nil {
				_ = f.Close()
			}
		}
	}
	l.reload()
	m.cur().ensureVisible(m.listRows())
	m.refreshPreview()
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
// height − header(1) − footer(1) − top/bottom border(2). Title is on the border.
func (m AppModel) listRows() int {
	if r := m.height - 4; r > 0 {
		return r
	}
	return 1
}
