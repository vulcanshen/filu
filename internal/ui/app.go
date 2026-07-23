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

// inputKind selects what the input popup collects.
type inputKind int

const (
	inputNone inputKind = iota
	inputRename
	inputAdd
)

// AppModel is filu's root model.
type AppModel struct {
	width         int
	height        int
	focus         panelID
	detail        detailTab
	detailScroll  int          // panel [3] content scroll offset
	tabs          [3]listModel // panel [2]'s fixed 3 directory tabs
	tab           int          // active tab index
	preview       previewModel
	places        placesModel
	carry         carryModel
	carryTab      int          // panel [4] active tab: 0 carry / 1 progress / 2 history
	spaceMenu     spaceMenu    // §A.1 contextual popup (kbu form)
	zoom          panelID      // 0 = normal; else the panel expanded full-width
	confirm       confirmPopup // yes/no popup (delete)
	pendingDelete string       // path awaiting delete confirmation
	inputPopup    inputPopup   // text prompt (rename / add)
	help          helpPopup    // §A.2 global help cheatsheet
}

// New returns the root model, focused on the file list. All 3 tabs open at the
// current dir (they diverge as the user navigates each).
func New() AppModel {
	dir, err := os.Getwd()
	if err != nil {
		dir = "/"
	}
	m := AppModel{focus: panelList, places: newPlaces(), spaceMenu: newSpaceMenu(), confirm: newConfirmPopup(), inputPopup: newInputPopup(), help: newHelpPopup()}
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
		m.confirm.setSize(msg.Width)
		m.inputPopup.setSize(msg.Width)
		m.help.setSize(msg.Width)
		if m.preview.kind == previewImage && m.previewWidth() != oldW {
			m.refreshPreview() // ASCII art is sized to the panel width
		}
	case AnimTickMsg:
		return m, tea.Batch(m.spaceMenu.handleTick(msg), m.confirm.handleTick(msg), m.inputPopup.handleTick(msg), m.help.handleTick(msg))
	case tea.KeyMsg:
		if m.help.isActive() { // modal cheatsheet
			if !m.help.isInteractive() {
				return m, nil
			}
			var cmd tea.Cmd
			m.help, cmd = m.help.update(msg)
			return m, cmd
		}
		if m.inputPopup.isActive() { // text entry owns the keyboard while open
			if !m.inputPopup.isInteractive() {
				return m, nil
			}
			var ok bool
			var cmd tea.Cmd
			m.inputPopup, ok, cmd = m.inputPopup.update(msg)
			if ok {
				m.performInput()
			}
			return m, cmd
		}
		if m.confirm.isActive() { // modal: owns the keyboard while open
			if !m.confirm.isInteractive() {
				return m, nil
			}
			var ok bool
			var cmd tea.Cmd
			m.confirm, ok, cmd = m.confirm.update(msg)
			if ok {
				_ = moveToTrash(m.pendingDelete)
				m.pendingDelete = ""
				m.cur().reload()
				m.cur().ensureVisible(m.listRows())
				m.refreshPreview()
			}
			return m, cmd
		}
		if m.spaceMenu.isActive() { // popup owns the keyboard while open
			if !m.spaceMenu.isInteractive() {
				return m, nil // swallow keys mid-animation
			}
			var key string
			var cmd tea.Cmd
			m.spaceMenu, key, cmd = m.spaceMenu.update(msg)
			if key != "" { // committed: fire on the focused panel, then close
				cmd = tea.Batch(cmd, m.dispatchFocusKey(key), m.spaceMenu.close())
			}
			return m, cmd
		}
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "?": // §A.2 global help cheatsheet
			return m, m.help.open()
		case " ": // Space opens the contextual menu for the focused panel
			items, title := m.buildSpaceMenu()
			if len(items) == 0 {
				return m, nil // nothing contextual here
			}
			m.spaceMenu.setItems(items, title)
			return m, m.spaceMenu.open()
		case "tab":
			m.setFocus(m.focus%4 + 1) // 1→2→3→4→1
		case "shift+tab":
			m.setFocus((m.focus+2)%4 + 1) // 1→4→3→2→1
		case "1":
			m.setFocus(panelPin)
		case "2":
			m.setFocus(panelList)
		case "3":
			m.setFocus(panelDetail)
		case "4":
			m.setFocus(panelCarry)
		default:
			return m, m.dispatchFocusKey(msg.String())
		}
	}
	return m, nil
}

// handleListKey routes navigation keys to panel [2] while it is focused. It may
// return a tea.Cmd when a key opens an animated popup (delete confirm, input).
func (m *AppModel) handleListKey(key string) tea.Cmd {
	l := m.cur()
	var cmd tea.Cmd
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
	case "p": // paste: land carried items here as copy
		m.carry.land(l.dir, false)
		l.reload()
	case "m": // move: land carried items here (move)
		m.carry.land(l.dir, true)
		l.reload()
	case "P": // pin: toggle the cursor dir into [1] Pinned
		if it := l.cursorItem(); it.isDir {
			m.places.togglePin(filepath.Join(l.dir, it.name))
		}
	case ".": // toggle hidden files in this tab
		l.showHidden = !l.showHidden
		l.reload()
	case "D": // delete: confirm first, then move to the system trash
		if it := l.cursorItem(); it.name != "" {
			m.pendingDelete = filepath.Join(l.dir, it.name)
			cmd = m.confirm.open("Move " + it.name + " to the trash?")
		}
	case "R": // rename cursor item (input popup)
		if it := l.cursorItem(); it.name != "" {
			cmd = m.inputPopup.open(inputRename, "Rename", it.name, it.name)
		}
	case "a": // add file/dir — lazyvim style: trailing / = dir (input popup)
		cmd = m.inputPopup.open(inputAdd, "New (trailing / = dir)", "", "")
	case "z": // zoom panel [2]: 3 directory tabs full-screen (1:1:1)
		m.toggleZoom(panelList)
	}
	m.cur().ensureVisible(m.listRows())
	m.refreshPreview()
	return cmd
}

// handleDetailKey routes keys to panel [3] while it is focused: h/l swap tab,
// j/k/u/d/g/G scroll the content.
func (m *AppModel) handleDetailKey(key string) tea.Cmd {
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
	case "z": // zoom panel [3]: full-width, Preview | Meta side by side
		m.toggleZoom(panelDetail)
	}
	m.clampDetailScroll()
	return nil
}

// clampDetailScroll keeps panel [3] from scrolling past its last page.
func (m *AppModel) clampDetailScroll() {
	maxScroll := max(len(m.detailLines())-m.detailRows(), 0)
	m.detailScroll = max(0, min(m.detailScroll, maxScroll))
}

// handleCarryKey routes keys to panel [4] while it is focused (h/l swap tab; on
// the Carries tab j/k move the cursor and P toggles the pick subset).
func (m *AppModel) handleCarryKey(key string) tea.Cmd {
	switch key {
	case "l", "right":
		m.carryTab = (m.carryTab + 1) % 3
	case "h", "left":
		m.carryTab = (m.carryTab + 2) % 3
	case "z": // zoom panel [4]: full-width, Carries | Progress | History
		m.toggleZoom(panelCarry)
	}
	if m.carryTab == 0 { // Carries tab
		switch key {
		case "j", "down":
			m.carry.moveCursor(1)
		case "k", "up":
			m.carry.moveCursor(-1)
		case "g":
			m.carry.cursor = 0
		case "G":
			m.carry.moveCursor(len(m.carry.items))
		case "P": // pick: toggle this item in the land subset
			m.carry.togglePick()
		}
	}
	return nil
}

// toggleZoom expands panel p (hiding the others), or restores the normal layout
// when p is already zoomed.
func (m *AppModel) toggleZoom(p panelID) {
	if m.zoom == p {
		m.zoom = 0
		return
	}
	m.zoom = p
}

// setFocus moves focus to p, exiting zoom only when p isn't part of the current
// zoom layout. [2]-zoom shows [2]+[4], so 2/4 switch focus without un-zooming.
func (m *AppModel) setFocus(p panelID) {
	if !m.zoomVisible(p) {
		m.zoom = 0
	}
	m.focus = p
}

// zoomVisible reports whether panel p appears in the current zoom layout.
func (m AppModel) zoomVisible(p panelID) bool {
	switch m.zoom {
	case 0:
		return true // normal layout: everything is visible
	case panelList:
		return p == panelList || p == panelCarry // [2]-zoom stacks [2] over [4]
	default:
		return p == m.zoom // [3]/[4]-zoom show only that panel
	}
}

// dispatchFocusKey fires a Space-menu commit as if the letter were pressed on
// the focused panel — the menu is a shell over the letter hotkeys.
func (m *AppModel) dispatchFocusKey(key string) tea.Cmd {
	switch m.focus {
	case panelList:
		return m.handleListKey(key)
	case panelPin:
		return m.handlePinKey(key)
	case panelDetail:
		return m.handleDetailKey(key)
	case panelCarry:
		return m.handleCarryKey(key)
	}
	return nil
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
		var itemOps, panelOps []menuItem
		if it.name != "" {
			itemOps = append(itemOps,
				menuItem{label: "Carry", key: "C", hint: `add to "carries" bucket`},
				menuItem{label: "Rename", key: "R", hint: "rename this item"},
				menuItem{label: "Delete", key: "D", hint: "move to the system trash"})
		}
		if it.isDir {
			itemOps = append(itemOps, menuItem{label: "Pin", key: "P", hint: "pin dir into the sidebar"})
		}
		if len(m.carry.items) > 0 {
			panelOps = append(panelOps,
				menuItem{label: "Paste here", key: "p", hint: "land carried items as copy"},
				menuItem{label: "Move here", key: "m", hint: "land carried items as move"})
		}
		panelOps = append(panelOps,
			menuItem{label: "Add", key: "a", hint: "new file / dir (trailing / = dir)"},
			menuItem{label: "Hidden", key: ".", hint: "toggle hidden files"},
			menuItem{label: "Zoom", key: "z", hint: "expand tabs to full-screen panels"})
		return groupedMenu(itemOps, panelOps), title
	case panelPin:
		items := []menuItem{{label: "Jump", key: "enter", hint: "go to this place"}}
		if m.places.currentIsPinned() {
			items = append(items, menuItem{label: "UnPin", key: "U", hint: "remove from Pinned"})
		}
		return items, "Places"
	case panelDetail:
		return groupedMenu(nil, []menuItem{
			{label: "Tab", key: "l", hint: "switch Preview / Meta"},
			{label: "Zoom", key: "z", hint: "expand tabs to full-screen panels"},
		}), "Preview"
	case panelCarry:
		panelOps := []menuItem{
			{label: "Tab", key: "l", hint: "switch Carries / Progress / History"},
			{label: "Zoom", key: "z", hint: "expand tabs to full-screen panels"},
		}
		if m.carryTab == 0 && len(m.carry.items) > 0 { // Carries tab, with items
			itemOps := []menuItem{{label: "Pick", key: "P", hint: "toggle this item in the land subset"}}
			return groupedMenu(itemOps, panelOps), "Carries"
		}
		return groupedMenu(nil, panelOps), "Bucket"
	}
	return nil, ""
}

// groupedMenu assembles a Space menu from item-level and panel-level actions.
// With both groups present it labels each region (kbu's "item operation" /
// "panel operation" headers, split by a rule); a single region stays flat and
// header-less so the menu doesn't shout when there's nothing to disambiguate.
func groupedMenu(itemOps, panelOps []menuItem) []menuItem {
	if len(itemOps) == 0 {
		return panelOps
	}
	if len(panelOps) == 0 {
		return itemOps
	}
	out := append([]menuItem{{header: true, label: "item operation"}}, itemOps...)
	out = append(out, menuItem{separator: true}, menuItem{header: true, label: "panel operation"})
	return append(out, panelOps...)
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

// performInput applies the committed input popup (rename / add) to the CWD.
func (m *AppModel) performInput() {
	name := strings.TrimSpace(m.inputPopup.buffer)
	kind, target := m.inputPopup.kind, m.inputPopup.target
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
func (m *AppModel) handlePinKey(key string) tea.Cmd {
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
	case "U": // unpin the cursor place (only meaningful on a Pinned entry)
		if m.places.currentIsPinned() {
			if p, ok := m.places.current(); ok {
				m.places.unpin(p.path)
				m.places.move(0) // reclamp the cursor after the list shrank
				m.syncPlaceToList()
			}
		}
	}
	return nil
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

// listPanelHeight is panel [2]'s box height: full in the narrow list-only
// fallback, otherwise the top 2/3 (both the normal grid and [2]-zoom put [2] over
// [4] at 2/3 : 1/3).
func (m AppModel) listPanelHeight() int {
	midH := m.height - 2
	if m.width < 72 {
		return midH
	}
	return midH * 2 / 3
}

// listRows: panel [2] file rows = box height − border(2) − Files header(1).
func (m AppModel) listRows() int {
	if r := m.listPanelHeight() - 3; r > 0 {
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
