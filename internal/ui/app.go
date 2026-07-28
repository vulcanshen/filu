// Package ui holds filu's Bubble Tea models. The 4-panel layout (list, preview,
// Carries, Tasks) lives in AppModel. See .forge/meta/IDEA.md for the target
// design.
package ui

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/fsnotify/fsnotify"
)

type panelID int

const (
	panelList    panelID = iota + 1 // [1] CWD file list (main surface)
	panelDetail                     // [2] preview (right column, top, 1/3 wide)
	panelCarries                    // [3] Marks bucket (bottom-left)
	panelTasks                      // [4] land tasks (bottom-right)
)

// inputKind selects what the input popup collects.
type inputKind int

const (
	inputNone inputKind = iota
	inputRename
	inputAdd
)

// confirmKind selects what the yes/no popup commits to when accepted.
type confirmKind int

const (
	confirmNone confirmKind = iota
	confirmDelete
)

// sortStep tracks where the sort picker is in its column→direction flow.
type sortStep int

const (
	sortStepColumn sortStep = iota
	sortStepDirection
)

// AppModel is filu's root model.
type AppModel struct {
	width         int
	height        int
	focus         panelID
	detailScroll  int         // panel [2] preview scroll offset
	tabs          []listModel // panel [1]'s directory tabs (1..maxTabs, user-created)
	tab           int         // active tab index
	pendingG      bool        // vim g-prefix chord: a lone g is armed, awaiting the second key
	preview       previewModel
	places        placesModel
	carry         carryModel
	spaceMenu     spaceMenu         // §A.1 contextual popup (kbu form)
	sortMenu      spaceMenu         // sort picker (column→direction chain, kbu form)
	sortStep      sortStep          // which step the sort picker is on
	sortFlowCol   sortCol           // column carried from the column step to direction
	quitMenu      spaceMenu         // cd-on-quit picker (launch dir + distinct tab dirs)
	openWithMenu  spaceMenu         // [o]pen picker (Default + config.yaml open_with apps)
	openWithPath  string            // path the open-with picker acts on (captured when it opens)
	gotoMenu      spaceMenu         // Goto / new-tab picker: {Same?, Favorites, Search} → favorites drill-down
	gotoStep      gotoStep          // which step the Goto picker is on
	searchMenu    spaceMenu         // Search chooser: {filename, content} → opens the finder in that mode
	gotoNewTab    bool              // Goto picker in new-tab mode (open in a new tab vs move the active one)
	launchDir     string            // the dir filu was started in (cd-on-quit option 1)
	zoom          panelID           // 0 = normal; else the panel expanded full-width
	confirm       confirmPopup      // yes/no popup (delete / quit)
	confirmAction confirmKind       // what an accepted confirm commits to
	pendingDelete string            // path awaiting delete confirmation
	inputPopup    inputPopup        // text prompt (rename / add)
	help          helpPopup         // §A.2 global help cheatsheet
	splash        splashModel       // hidden easter-egg logo (V)
	toast         toastModel        // transient notification (yank feedback)
	detailYank    detailYank        // panel [2] yank viewport (cursor + visual selection)
	pty           *ptyPopup         // embedded editor; pointer — shared with its read goroutine
	search        searchModel       // native fuzzy file/dir finder
	searchCh      chan fileBatchMsg // finder's fd stream → UI
	breadcrumb    breadcrumbPopup   // [b] ancestor-directory jump popup
	tasks         []landTask        // land operations (Tasks tab: running + log)
	taskCh        chan landMsg      // land goroutines → UI
	nextTaskID    int
	spinnerFrame  int               // running-task spinner animation
	spinning      bool              // a spinner tick is in flight
	taskCursor    int               // cursor over the Tasks tab
	watcher       *fsnotify.Watcher // live directory watch (nil if unavailable)
	watchCh       chan watchMsg     // watcher goroutine → UI
	watched       map[string]bool   // dirs currently registered with the watcher
}

// maxTabs caps panel [2]'s directory tabs. It opens with one (at the CWD); the
// user creates more with `t` (up to this many) and closes them with `w`.
const maxTabs = 5

// New returns the root model, focused on the file list. Panel [2] opens with a
// single tab at the current dir; extra tabs the user created last session are
// restored by applyState.
func New() AppModel {
	loadConfig() // apply config.yaml (finder cap) before any finder can open
	dir, err := os.Getwd()
	if err != nil {
		dir = "/"
	}
	m := AppModel{focus: panelList, launchDir: dir, spaceMenu: newSpaceMenu(), sortMenu: newSortMenu(), quitMenu: newQuitMenu(), openWithMenu: newOpenWithMenu(), gotoMenu: newGotoMenu(), searchMenu: newSearchMenu(), confirm: newConfirmPopup(), inputPopup: newInputPopup(), help: newHelpPopup(), splash: newSplashModel(), toast: newToast(), detailYank: newDetailYank(), pty: newPtyPopup(), search: newSearch(), breadcrumb: newBreadcrumbPopup(), taskCh: make(chan landMsg, 64), searchCh: make(chan fileBatchMsg, 16), watched: map[string]bool{}}
	m.tabs = []listModel{newList(dir)}
	if st, ok := loadState(); ok { // restore last session
		m.applyState(st)
	}
	m.refreshPreview()
	if m.watcher = newWatcher(); m.watcher != nil { // live refresh of the tab dirs
		m.watchCh = make(chan watchMsg, 16)
		m.syncWatches()
		go watchLoop(m.watcher, m.watchCh)
	}
	return m
}

// shutdown persists the session, stops the watcher, and quits.
func (m *AppModel) shutdown() tea.Cmd {
	saveState(m.snapshotState()) // restore this session on next launch
	if m.watcher != nil {
		m.watcher.Close()
	}
	return tea.Quit
}

// cur returns a pointer to the active directory tab.
func (m *AppModel) cur() *listModel { return &m.tabs[m.tab] }

// addTab appends a new panel [2] tab at dir and makes it active. Callers guard
// against exceeding maxTabs.
func (m *AppModel) addTab(dir string) {
	m.tabs = append(m.tabs, newList(dir))
	m.tab = len(m.tabs) - 1
}

// tabLimitToast is the message shown when t / T would exceed maxTabs.
func tabLimitToast() string {
	return "Tab limit reached (" + strconv.Itoa(maxTabs) + ") — close one with w"
}

// active returns the active tab by value (read-only paths).
func (m AppModel) active() listModel { return m.tabs[m.tab] }

func (m AppModel) Init() tea.Cmd { // persistent readers: land results, live-refresh, finder stream
	return tea.Batch(m.waitLand(), m.waitWatch(), m.waitSearch())
}

// waitSearch reads one batch of the finder's streamed file listing.
func (m AppModel) waitSearch() tea.Cmd {
	ch := m.searchCh
	return func() tea.Msg { return <-ch }
}

func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case landMsg:
		m.handleLandMsg(msg)
		return m, m.waitLand()
	case watchMsg:
		m.handleWatchMsg(msg)
		return m, m.waitWatch()
	case clipboardCopiedMsg:
		return m, m.toast.show(msg.note)
	case clipboardFailedMsg:
		return m, m.toast.show("Clipboard unavailable")
	case toastDismissMsg:
		return m, m.toast.dismiss(msg)
	case ptyTickMsg:
		return m, m.pty.update(msg)
	case ptyExitMsg:
		for i := range m.tabs { // an edit or shell session may have changed files — reload its dir
			if m.tabs[i].dir == msg.dir {
				m.tabs[i].reloadPreserving()
			}
		}
		m.refreshPreview()
		return m, nil
	case fileBatchMsg:
		m.search.onStreamBatch(msg)
		return m, m.waitSearch() // keep reading the stream

	case grepFireMsg:
		return m, m.search.onGrepFire(msg)
	case grepFilesMsg:
		m.search.onGrepResult(msg)
		return m, nil
	case searchBlinkMsg:
		return m, m.search.onBlink(msg)
	case inputBlinkMsg:
		return m, m.inputPopup.onBlink(msg)
	case searchConfirmMsg:
		if msg.newTab { // T: the picked directory becomes a new tab
			if len(m.tabs) < maxTabs {
				m.addTab(msg.path)
			}
		} else {
			m.revealPath(msg.path)
		}
		m.syncWatches() // the tab may have moved to a new dir
		m.refreshPreview()
		return m, nil
	case spinnerTickMsg:
		m.spinnerFrame++
		if m.anyRunning() {
			return m, spinnerTick()
		}
		m.spinning = false
		return m, nil
	case tea.WindowSizeMsg:
		oldW := m.previewWidth()
		m.width, m.height = msg.Width, msg.Height
		m.spaceMenu.setSize(msg.Width)
		m.sortMenu.setSize(msg.Width)
		m.quitMenu.setSize(msg.Width)
		m.openWithMenu.setSize(msg.Width)
		m.gotoMenu.setSize(msg.Width)
		m.confirm.setSize(msg.Width)
		m.inputPopup.setSize(msg.Width)
		m.help.setSize(msg.Width)
		m.toast.setSize(msg.Width)
		m.detailYank.setSize(msg.Width, msg.Height)
		m.pty.setSize(msg.Width, msg.Height)
		m.search.setSize(msg.Width, msg.Height)
		m.breadcrumb.setSize(msg.Width)
		m.cur().ensureVisible(m.listRows()) // scroll a restored cursor into view
		if m.preview.kind == previewImage && m.previewWidth() != oldW {
			m.refreshPreview() // ASCII art is sized to the panel width
		}
	case AnimTickMsg:
		return m, tea.Batch(m.spaceMenu.handleTick(msg), m.sortMenu.handleTick(msg), m.quitMenu.handleTick(msg), m.openWithMenu.handleTick(msg), m.gotoMenu.handleTick(msg), m.confirm.handleTick(msg), m.inputPopup.handleTick(msg), m.help.handleTick(msg), m.toast.handleTick(msg), m.detailYank.handleTick(msg), m.pty.handleTick(msg), m.search.handleTick(msg), m.breadcrumb.handleTick(msg))
	case splashTickMsg, splashIdentityMsg, splashHintMsg:
		var cmd tea.Cmd
		m.splash, cmd = m.splash.update(msg)
		return m, cmd
	case tea.KeyMsg:
		if m.splash.isActive() { // hidden easter-egg logo owns the keyboard until dismissed
			var cmd tea.Cmd
			m.splash, cmd = m.splash.update(msg)
			return m, cmd
		}
		if m.pty.isActive() { // the embedded editor owns every keystroke
			return m, m.pty.update(msg)
		}
		if m.detailYank.isActive() { // yank viewport owns the keyboard while open
			if !m.detailYank.isInteractive() {
				return m, nil
			}
			var cmd tea.Cmd
			m.detailYank, cmd = m.detailYank.update(msg)
			return m, cmd
		}
		if m.search.isActive() { // fuzzy finder owns the keyboard while open
			if !m.search.isInteractive() {
				return m, nil
			}
			var cmd tea.Cmd
			m.search, cmd = m.search.update(msg)
			return m, cmd
		}
		if m.breadcrumb.isActive() { // ancestor-jump popup owns the keyboard while open
			if !m.breadcrumb.isInteractive() {
				return m, nil
			}
			var path string
			var cmd tea.Cmd
			m.breadcrumb, path, cmd = m.breadcrumb.update(msg)
			if path != "" { // Enter on a level: jump the active tab there
				m.revealPath(path)
				m.cur().ensureVisible(m.listRows())
				m.refreshPreview()
			}
			return m, cmd
		}
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
				if m.confirmAction == confirmDelete {
					_ = moveToTrash(m.pendingDelete)
					m.pendingDelete = ""
					m.cur().reload()
					m.cur().ensureVisible(m.listRows())
					m.refreshPreview()
				}
				m.confirmAction = confirmNone
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
		if m.sortMenu.isActive() { // sort picker owns the keyboard; commits drive the chain flow
			if !m.sortMenu.isInteractive() {
				return m, nil
			}
			var key string
			var cmd tea.Cmd
			m.sortMenu, key, cmd = m.sortMenu.update(msg)
			if key != "" { // stays open, swapping to the next step / looping back
				cmd = tea.Batch(cmd, m.advanceSortFlow(key))
			}
			return m, cmd
		}
		if m.gotoMenu.isActive() { // Goto picker: Favorites drill-down or Search finder
			if !m.gotoMenu.isInteractive() {
				return m, nil
			}
			if m.gotoStep == gotoStepPinned && msg.String() == "f" { // unfavorite the highlighted dir, stay open
				return m, m.unpinAtGotoCursor()
			}
			var key string
			var cmd tea.Cmd
			m.gotoMenu, key, cmd = m.gotoMenu.update(msg)
			if key != "" { // stays open on a drill, closes on a terminal jump/search
				cmd = tea.Batch(cmd, m.advanceGotoFlow(key))
			}
			return m, cmd
		}
		if m.searchMenu.isActive() { // Search chooser: filename vs content, then open the finder
			if !m.searchMenu.isInteractive() {
				return m, nil
			}
			var key string
			var cmd tea.Cmd
			m.searchMenu, key, cmd = m.searchMenu.update(msg)
			switch key {
			case "f": // filename → the by-name (fd) finder
				return m, tea.Batch(m.searchMenu.close(), m.openSearch())
			case "c": // content → the by-content (rg) finder
				return m, tea.Batch(m.searchMenu.close(), m.openFind())
			}
			return m, cmd
		}
		if m.quitMenu.isActive() { // cd-on-quit picker; a commit cds and quits
			if !m.quitMenu.isInteractive() {
				return m, nil
			}
			var key string
			var cmd tea.Cmd
			m.quitMenu, key, cmd = m.quitMenu.update(msg)
			if idx, err := strconv.Atoi(key); err == nil { // a number → that distinct dir
				if targets := m.quitTargets(); idx >= 1 && idx <= len(targets) {
					return m, m.quitTo(targets[idx-1].dir)
				}
			}
			return m, cmd
		}
		if m.openWithMenu.isActive() { // [o]pen picker; a commit launches the app
			if !m.openWithMenu.isInteractive() {
				return m, nil
			}
			var key string
			var cmd tea.Cmd
			m.openWithMenu, key, cmd = m.openWithMenu.update(msg)
			if idx, err := strconv.Atoi(key); err == nil { // a number → that app (1 = Default)
				if run := m.runOpenWith(idx); run != nil {
					return m, tea.Batch(run, m.openWithMenu.close())
				}
			}
			return m, cmd
		}
		// vim g-prefix chord for the focused panel (kbu parity). Every popup above
		// owns the keyboard while open, so we only reach here for the main panels: a
		// lone g arms and waits, then `gg` jumps to the top (falling through to the
		// handlers' `case "g"`) and `go` opens goto. Any other second key cancels and
		// runs normally. (The Space menu and the finder keep a single g, as they do
		// in kbu; the panel [3] yank viewport carries its own gg.)
		if m.pendingG {
			m.pendingG = false
			if msg.String() == "o" && m.focus == panelList { // `go` → goto
				return m, m.handleListKey("go")
			}
		} else if msg.String() == "g" {
			m.pendingG = true
			return m, nil
		}
		switch msg.String() {
		case "ctrl+c": // hard quit — abandon any running task, no cd-on-quit
			return m, m.shutdown()
		case "q": // pick where to leave the shell, then quit (cd-on-quit)
			return m, m.openQuitMenu()
		case "?": // §A.2 global help cheatsheet
			return m, m.help.open()
		case "V": // hidden easter-egg: the u-family logo
			return m, m.splash.show()
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
			m.setFocus(panelList)
		case "2":
			m.setFocus(panelDetail)
		case "3":
			m.setFocus(panelCarries)
		case "4":
			m.setFocus(panelTasks)
		default:
			cmd := m.dispatchFocusKey(msg.String())
			m.syncWatches() // navigation may have moved a tab to a new dir
			if os.Getenv("FILU_REPAINT") != "" {
				// Some terminals draw Nerd Font glyphs wider than they advance the
				// cursor; the overflow can leave stray cells bubbletea's diff won't
				// clear. A forced repaint on navigation wipes them (opt-in — costs a
				// full redraw per key).
				cmd = tea.Batch(cmd, tea.ClearScreen)
			}
			return m, cmd
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
		// Enter navigates into directories only — in filu it is not "open a file".
		// Opening a file is [o]pen's job (open-with); a file row Enter is a no-op.
		if it := l.cursorItem(); it.isDir {
			l.enter()
		}
	case "esc":
		l.parent()
	case "l", "right":
		m.tab = (m.tab + 1) % len(m.tabs)
	case "h", "left":
		m.tab = (m.tab + len(m.tabs) - 1) % len(m.tabs)
	case "t": // new tab: open the Same / Favorites / Search picker (up to maxTabs)
		if len(m.tabs) < maxTabs {
			cmd = m.openTabMenu()
		} else {
			cmd = m.toast.show(tabLimitToast())
		}
	case "w": // close the active tab (always keep at least one)
		if len(m.tabs) > 1 {
			m.tabs = append(m.tabs[:m.tab], m.tabs[m.tab+1:]...)
			if m.tab >= len(m.tabs) {
				m.tab = len(m.tabs) - 1
			}
		}
	case "m": // mark: toggle cursor item into the marks bucket
		if it := l.cursorItem(); it.name != "" {
			m.carry.toggle(filepath.Join(l.dir, it.name))
		}
	case "c": // copy: land marked items here as copy (async)
		cmd = m.startLand(l.dir, false)
	case "v": // move: land marked items here as move (async)
		cmd = m.startLand(l.dir, true)
	case "f": // favorite: toggle the cursor dir into Favorites (reach it via Goto)
		if it := l.cursorItem(); it.isDir {
			m.places.togglePin(filepath.Join(l.dir, it.name))
		}
	case ".": // toggle hidden files in this tab
		l.showHidden = !l.showHidden
		l.reload()
	case "D": // delete: confirm first, then move to the system trash
		if it := l.cursorItem(); it.name != "" {
			m.pendingDelete = filepath.Join(l.dir, it.name)
			m.confirmAction = confirmDelete
			cmd = m.confirm.open("Move " + it.name + " to the trash?")
		}
	case "r": // rename cursor item (input popup: name as the description, pre-filled)
		if it := l.cursorItem(); it.name != "" {
			cmd = m.inputPopup.open(inputRename, "Rename", it.name, it)
		}
	case "a": // add file/dir — lazyvim style: trailing / = dir (input popup)
		cmd = m.inputPopup.open(inputAdd, "New (trailing / = dir)", "", fileItem{})
	case "y": // yank: copy the item's full path to the clipboard
		if it := l.cursorItem(); it.name != "" {
			cmd = copyToClipboardCmd(filepath.Join(l.dir, it.name), "Copied path to clipboard")
		}
	case "o": // open with the OS default app (fast path; O opens the picker)
		cmd = m.openDefault()
	case "O": // Open with: pick an app (Default OS open, or a configured one)
		cmd = m.openOpenWith()
	case "s": // shell: drop into $SHELL in this tab's directory (exit to return)
		cmd = m.pty.start(buildShellCmd(), "Shell", l.dir, m.width, m.height)
	case "S": // Sort: pick a column → direction; the column-header row shows the active sort
		cmd = m.openSortColumnPicker()
	case "/": // Search: choose filename (fd) or content (rg), then reveal the pick here
		cmd = m.openSearchMenu()
	case "go": // Goto: pick a pinned dir, or search under $HOME (chord `go`)
		cmd = m.openGotoMenu()
	case "b": // Breadcrumb: jump this tab up to any ancestor directory
		cmd = m.breadcrumb.open(m.cur().dir)
	case "z": // zoom panel [2]: 3 directory tabs full-screen (1:1:1)
		m.toggleZoom(panelList)
	}
	m.cur().ensureVisible(m.listRows())
	m.refreshPreview()
	return cmd
}

// handleDetailKey routes keys to panel [3] (the preview) while it is focused:
// j/k/u/d/g/G scroll, y yanks the preview, z zooms it full-screen.
func (m *AppModel) handleDetailKey(key string) tea.Cmd {
	switch key {
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
	case "y":
		return m.openDetailYank()
	case "z":
		m.toggleZoom(panelDetail)
	}
	m.clampDetailScroll()
	return nil
}

// openDetailYank opens the yank viewport over panel [2]'s preview — the file's
// own content, with a display-only line-number gutter for text/binary.
func (m *AppModel) openDetailYank() tea.Cmd {
	lines := m.preview.body
	if len(lines) == 0 {
		return nil
	}
	showGutter := m.preview.kind == previewText || m.preview.kind == previewBinary
	m.detailYank.setSize(m.width, m.height)
	return m.detailYank.open("Yank: Preview", lines, showGutter)
}

// clampDetailScroll keeps panel [2] (preview) from scrolling past its last page.
func (m *AppModel) clampDetailScroll() {
	maxScroll := max(len(m.detailLines())-m.detailRows(), 0)
	m.detailScroll = max(0, min(m.detailScroll, maxScroll))
}

// handleCarriesKey routes keys to the Carries bucket panel [3]: j/k move the
// cursor, p toggles the item in the land subset, D drops it from the bucket, y
// yanks its path, z zooms the panel.
func (m *AppModel) handleCarriesKey(key string) tea.Cmd {
	switch key {
	case "j", "down":
		m.carry.moveCursor(1)
	case "k", "up":
		m.carry.moveCursor(-1)
	case "g":
		m.carry.cursor = 0
	case "G":
		m.carry.moveCursor(len(m.carry.items))
	case "p": // pick: toggle this item in the land subset
		m.carry.togglePick()
	case "D": // delete: drop this item from the bucket (not the file)
		if m.carry.cursor >= 0 && m.carry.cursor < len(m.carry.items) {
			m.carry.removeItem(m.carry.items[m.carry.cursor])
		}
	case "y": // yank: copy this item's full path to the clipboard
		if m.carry.cursor >= 0 && m.carry.cursor < len(m.carry.items) {
			return copyToClipboardCmd(m.carry.items[m.carry.cursor], "Copied path to clipboard")
		}
	case "z":
		m.toggleZoom(panelCarries)
	}
	return nil
}

// handleTasksKey routes keys to the Tasks panel [4]: j/k move the cursor, R redoes
// a task, D drops it from the log, z zooms the panel.
func (m *AppModel) handleTasksKey(key string) tea.Cmd {
	switch key {
	case "j", "down":
		m.taskCursor++
		m.clampTaskCursor()
	case "k", "up":
		m.taskCursor--
		m.clampTaskCursor()
	case "g":
		m.taskCursor = 0
	case "G":
		m.taskCursor = len(m.tasks) - 1
		m.clampTaskCursor()
	case "R": // redo: run this task again
		if m.taskCursor >= 0 && m.taskCursor < len(m.tasks) {
			return m.redoTask(m.tasks[m.taskCursor])
		}
	case "D": // delete: drop this task from the log
		if m.taskCursor >= 0 && m.taskCursor < len(m.tasks) {
			m.tasks = append(m.tasks[:m.taskCursor], m.tasks[m.taskCursor+1:]...)
			m.clampTaskCursor()
			saveState(m.snapshotState())
		}
	case "z":
		m.toggleZoom(panelTasks)
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

// zoomVisible reports whether panel p appears in the current zoom layout. Each
// panel zooms to full-screen on its own, so only the zoomed panel is visible.
func (m AppModel) zoomVisible(p panelID) bool {
	return m.zoom == 0 || p == m.zoom
}

// dispatchFocusKey fires a Space-menu commit as if the letter were pressed on
// the focused panel — the menu is a shell over the letter hotkeys.
func (m *AppModel) dispatchFocusKey(key string) tea.Cmd {
	switch m.focus {
	case panelList:
		return m.handleListKey(key)
	case panelDetail:
		return m.handleDetailKey(key)
	case panelCarries:
		return m.handleCarriesKey(key)
	case panelTasks:
		return m.handleTasksKey(key)
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
				menuItem{label: "Open", key: "o", hint: "open with the OS default app"},
				menuItem{label: "Open with", key: "O", hint: "pick an app (open_with in config.yaml)"},
				menuItem{label: "Mark", key: "m", hint: "add to the marks bucket"},
				menuItem{label: "Yank", key: "y", hint: "copy full path to clipboard"},
				menuItem{label: "Rename", key: "r", hint: "rename this item"},
				menuItem{label: "Delete", key: "D", hint: "move to the system trash"})
		}
		if it.isDir {
			itemOps = append(itemOps, menuItem{label: "Favorite", key: "f", hint: "favorite this dir (reach it via Goto → Favorites)"})
		}
		if len(m.carry.items) > 0 {
			panelOps = append(panelOps,
				menuItem{label: "Copy", key: "c", hint: "land marked items as copy"},
				menuItem{label: "Move here", key: "v", hint: "land marked items as move"})
		}
		panelOps = append(panelOps,
			menuItem{label: "Search", key: "/", hint: "find a file by name or content in this tree"},
			menuItem{label: "Goto", key: "go", hint: "jump to a pinned dir, or search under home"},
			menuItem{label: "Breadcrumb", key: "b", hint: "jump this tab up to an ancestor directory"})
		if len(m.tabs) < maxTabs {
			panelOps = append(panelOps,
				menuItem{label: "Tab", key: "t", hint: "create a new tab"})
		}
		if len(m.tabs) > 1 {
			panelOps = append(panelOps,
				menuItem{label: "Close tab", key: "w", hint: "close the active tab"})
		}
		panelOps = append(panelOps,
			menuItem{label: "Add", key: "a", hint: "new file / dir (trailing / = dir)"},
			menuItem{label: "Sort", key: "S", hint: "order by a column (name / modified / perms)"},
			menuItem{label: "Shell", key: "s", hint: "drop into $SHELL here (exit to return)"},
			menuItem{label: "Hidden", key: ".", hint: "toggle hidden files"},
			menuItem{label: "Zoom", key: "z", hint: "expand tabs to full-screen panels"})
		return groupedMenu(itemOps, panelOps), title
	case panelDetail:
		return groupedMenu(
			[]menuItem{{label: "Yank", key: "y", hint: "select & copy the preview"}},
			[]menuItem{{label: "Zoom", key: "z", hint: "expand the preview full-screen"}}), "Preview"
	case panelCarries:
		zoom := []menuItem{{label: "Zoom", key: "z", hint: "expand this panel full-screen"}}
		if len(m.carry.items) > 0 {
			itemOps := []menuItem{
				{label: "Pick", key: "p", hint: "toggle this item in the land subset"},
				{label: "Yank", key: "y", hint: "copy full path to clipboard"},
				{label: "Delete", key: "D", hint: "remove this item from the bucket"},
			}
			return groupedMenu(itemOps, zoom), "Marks"
		}
		return groupedMenu(nil, zoom), "Marks"
	case panelTasks:
		zoom := []menuItem{{label: "Zoom", key: "z", hint: "expand this panel full-screen"}}
		if len(m.tasks) > 0 {
			itemOps := []menuItem{
				{label: "Redo", key: "R", hint: "run this task again"},
				{label: "Delete", key: "D", hint: "remove this task from the log"},
			}
			return groupedMenu(itemOps, zoom), "Tasks"
		}
		return groupedMenu(nil, zoom), "Tasks"
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
	m.detailScroll = 0 // new cursor item — the preview scrolls back to the top
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
	kind, target := m.inputPopup.kind, m.inputPopup.item.name
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

// midHeight is the panel region's height: total minus the header, status bar,
// and footer rows.
func (m AppModel) midHeight() int { return m.height - 3 }

// listPanelHeight is panel [2]'s box height: full in the narrow list-only
// fallback, otherwise the top 2/3 (both the normal grid and [2]-zoom put [2] over
// [4] at 2/3 : 1/3).
func (m AppModel) listPanelHeight() int {
	midH := m.midHeight()
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

// detailRows: panel [3] preview content rows (right column, top 2/3, minus its
// two borders).
func (m AppModel) detailRows() int {
	if r := m.midHeight()*2/3 - 2; r > 0 {
		return r
	}
	return 1
}
