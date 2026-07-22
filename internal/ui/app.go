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
	tabMeta
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
	width        int
	height       int
	focus        panelID
	detail       detailTab
	detailScroll int          // panel [3] content scroll offset
	tabs         [3]listModel // panel [2]'s fixed 3 directory tabs
	tab          int          // active tab index
	preview      previewModel
	places       placesModel
	carry        carryModel
	carryTab     int // panel [4] active tab: 0 carry / 1 progress / 2 history
	input        inputState
	spaceMenu    spaceMenu // §A.1 contextual popup (kbu form)
}

// New returns the root model, focused on the file list. All 3 tabs open at the
// current dir (they diverge as the user navigates each).
func New() AppModel {
	dir, err := os.Getwd()
	if err != nil {
		dir = "/"
	}
	m := AppModel{focus: panelList, places: newPlaces(), spaceMenu: newSpaceMenu()}
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
		oldW := m.previewWidth()
		m.width, m.height = msg.Width, msg.Height
		m.spaceMenu.setSize(msg.Width)
		if m.preview.kind == previewImage && m.previewWidth() != oldW {
			m.refreshPreview() // ASCII art is sized to the panel width
		}
	case AnimTickMsg:
		return m, m.spaceMenu.handleTick(msg)
	case tea.KeyMsg:
		if m.input.kind != inputNone {
			m.handleInputKey(msg)
			return m, nil
		}
		if m.spaceMenu.isActive() { // popup owns the keyboard while open
			if !m.spaceMenu.isInteractive() {
				return m, nil // swallow keys mid-animation
			}
			var key string
			var cmd tea.Cmd
			m.spaceMenu, key, cmd = m.spaceMenu.update(msg)
			if key != "" { // committed: fire on the focused panel, then close
				m.dispatchFocusKey(key)
				cmd = tea.Batch(cmd, m.spaceMenu.close())
			}
			return m, cmd
		}
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case " ": // Space opens the contextual menu for the focused panel
			items, title := m.buildSpaceMenu()
			m.spaceMenu.setItems(items, title)
			return m, m.spaceMenu.open()
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
			case panelCarry:
				m.handleCarryKey(msg.String())
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
		m.input = inputState{kind: inputAdd, prompt: "New (trailing / = dir)"}
	}
	m.cur().ensureVisible(m.listRows())
	m.refreshPreview()
}

// handleDetailKey routes keys to panel [3] while it is focused: h/l swap tab,
// j/k/u/d/g/G scroll the content.
func (m *AppModel) handleDetailKey(key string) {
	switch key {
	case "h", "left", "l", "right":
		if m.detail == tabPreview {
			m.detail = tabMeta
		} else {
			m.detail = tabPreview
		}
		m.detailScroll = 0
	case "j", "down":
		m.detailScroll++
	case "k", "up":
		m.detailScroll--
	case "d", "ctrl+d":
		m.detailScroll += m.detailRows() / 2
	case "u", "ctrl+u":
		m.detailScroll -= m.detailRows() / 2
	case "g":
		m.detailScroll = 0
	case "G":
		m.detailScroll = len(m.detailLines())
	}
	m.clampDetailScroll()
}

// clampDetailScroll keeps panel [3] from scrolling past its last page.
func (m *AppModel) clampDetailScroll() {
	maxScroll := max(len(m.detailLines())-m.detailRows(), 0)
	m.detailScroll = max(0, min(m.detailScroll, maxScroll))
}

// handleCarryKey routes keys to panel [4] while it is focused (h/l swap tab).
func (m *AppModel) handleCarryKey(key string) {
	switch key {
	case "l", "right":
		m.carryTab = (m.carryTab + 1) % 3
	case "h", "left":
		m.carryTab = (m.carryTab + 2) % 3
	}
}

// dispatchFocusKey fires a Space-menu commit as if the letter were pressed on
// the focused panel — the menu is a shell over the letter hotkeys.
func (m *AppModel) dispatchFocusKey(key string) {
	switch m.focus {
	case panelList:
		m.handleListKey(key)
	case panelPin:
		m.handlePinKey(key)
	case panelDetail:
		m.handleDetailKey(key)
	case panelCarry:
		m.handleCarryKey(key)
	}
}

// buildSpaceMenu returns the contextual menu items + title for the focused
// panel. Every implemented contextual letter hotkey appears here (ZLC §A.1
// completeness); items are gated by what actually applies to the cursor state.
func (m AppModel) buildSpaceMenu() ([]menuItem, string) {
	switch m.focus {
	case panelList:
		it := m.active().cursorItem()
		title := "CWD"
		if it.name != "" {
			title = it.name
		}
		var items []menuItem
		if it.name != "" {
			items = append(items, menuItem{label: "Carry", key: "C", hint: "add to the carry bucket"})
		}
		if len(m.carry.items) > 0 {
			items = append(items,
				menuItem{label: "Copy here", key: "c", hint: "land carried items as copy"},
				menuItem{label: "Move here", key: "x", hint: "land carried items as move"})
		}
		if it.name != "" {
			items = append(items, menuItem{label: "Rename", key: "R", hint: "rename this item"})
		}
		items = append(items, menuItem{label: "Add", key: "A", hint: "new file / dir (trailing / = dir)"})
		if it.name != "" {
			items = append(items, menuItem{label: "Delete", key: "D", hint: "move to the system trash"})
		}
		if it.isDir {
			items = append(items, menuItem{label: "Pin", key: "P", hint: "pin dir into the sidebar"})
		}
		items = append(items, menuItem{label: "Hidden", key: ".", hint: "toggle hidden files"})
		return items, title
	case panelPin:
		return []menuItem{{label: "Jump", key: "enter", hint: "go to this place"}}, "Places"
	case panelDetail:
		return []menuItem{{label: "Tab", key: "l", hint: "switch Preview / Meta"}}, "Preview"
	case panelCarry:
		return []menuItem{{label: "Tab", key: "l", hint: "switch Carry / Progress / History"}}, "Bucket"
	}
	return nil, ""
}

// refreshPreview reloads panel [3]'s preview for the current cursor item.
func (m *AppModel) refreshPreview() {
	l := m.cur()
	m.preview = loadPreview(l.cursorItem(), l.dir, m.previewWidth())
	m.detailScroll = 0 // new content — back to the top
}

// previewWidth is panel [3]'s inner content width (1:1:1 columns, minus border),
// used to size image ASCII previews.
func (m AppModel) previewWidth() int {
	rightW := m.width - (m.width/3)*2 // panel [3] is the remainder column
	if w := rightW - 2; w > 0 {
		return w
	}
	return 1
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
		m.syncPlaceToList()
	case "k", "up":
		m.places.move(-1)
		m.syncPlaceToList()
	case "enter":
		if p, ok := m.places.current(); ok {
			m.navigateTo(p.path)
		}
	}
}

// navigateActive points the active tab at dir; focus stays put.
func (m *AppModel) navigateActive(dir string) {
	l := m.cur()
	l.dir = dir
	l.cursor, l.offset = 0, 0
	l.reload()
	l.ensureVisible(m.listRows())
	m.refreshPreview()
}

// navigateTo navigates and moves focus to panel [2].
func (m *AppModel) navigateTo(dir string) {
	m.navigateActive(dir)
	m.focus = panelList
}

// syncPlaceToList live-updates panel [2] to the highlighted place (focus stays).
func (m *AppModel) syncPlaceToList() {
	if p, ok := m.places.current(); ok {
		m.navigateActive(p.path)
	}
}

// listRows: panel [2] file rows = height − header − footer − border(2) − Files header.
func (m AppModel) listRows() int {
	if r := m.height - 5; r > 0 {
		return r
	}
	return 1
}

// detailRows: panel [3] content rows (no section header there).
func (m AppModel) detailRows() int {
	if r := m.height - 4; r > 0 {
		return r
	}
	return 1
}
