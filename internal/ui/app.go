// Package ui holds filu's Bubble Tea models. The 3-panel layout (list, preview,
// and a tabbed Marks | Tasks panel) lives in AppModel. See .forge/meta/IDEA.md
// for the target design.
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
	panelList   panelID = iota + 1 // [1] CWD file list (main surface)
	panelDetail                    // [2] preview (right column, top, 1/3 wide)
	panelMarks                     // [3] Marks | Tasks | Favorites (tabbed, full-width bottom)
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
	confirmShell
	confirmOpen
	confirmUnfavorite
)

// sortStep tracks where the sort picker is in its column→direction flow.
type sortStep int

const (
	sortStepColumn sortStep = iota
	sortStepDirection
)

// AppModel is filu's root model.
type AppModel struct {
	width             int
	height            int
	focus             panelID
	detailScroll      int         // panel [2] preview scroll offset
	tabs              []listModel // panel [1]'s directory tabs (1..maxTabs, user-created)
	tab               int         // active tab index
	pendingG          bool        // vim g-prefix chord: a lone g is armed, awaiting the second key
	preview           previewModel
	places            placesModel
	marks             marksModel
	marksTab          int               // panel [3] active tab: 0 Marks / 1 Tasks / 2 Favorites
	spaceMenu         spaceMenu         // §A.1 contextual popup (kbu form)
	sortMenu          spaceMenu         // sort picker (column→direction chain, kbu form)
	sortStep          sortStep          // which step the sort picker is on
	sortFlowCol       sortCol           // column carried from the column step to direction
	quitMenu          spaceMenu         // cd-on-quit picker (launch dir + distinct tab dirs)
	openWithMenu      spaceMenu         // [o]pen picker (Default + config.yaml open_with apps)
	openWithPath      string            // path the open-with picker acts on (captured when it opens)
	gotoMenu          spaceMenu         // Goto / new-tab picker: {Same?, Favorites, Search} → favorites drill-down
	gotoStep          gotoStep          // which step the Goto picker is on
	searchMenu        spaceMenu         // Search chooser: {filename, content} → opens the finder in that mode
	openInMenu        spaceMenu         // Favorites "Open dir in…" picker: New tab / an existing panel [1] tab
	openInPath        string            // the favorite dir the openInMenu is acting on
	gotoNewTab        bool              // Goto picker in new-tab mode (open in a new tab vs move the active one)
	launchDir         string            // the dir filu was started in (cd-on-quit option 1)
	zoom              panelID           // 0 = normal; else the panel expanded full-width
	confirm           confirmPopup      // yes/no popup (delete / quit)
	confirmAction     confirmKind       // what an accepted confirm commits to
	pendingDelete     string            // path awaiting delete confirmation
	pendingUnfavorite string            // favorite path awaiting unfavorite confirmation
	inputPopup        inputPopup        // text prompt (rename / add)
	help              helpPopup         // §A.2 global help cheatsheet
	splash            splashModel       // hidden easter-egg logo (V)
	toast             toastModel        // transient notification (yank feedback)
	detailYank        detailYank        // panel [2] yank viewport (cursor + visual selection)
	pty               *ptyPopup         // embedded editor; pointer — shared with its read goroutine
	search            searchModel       // native fuzzy file/dir finder
	searchCh          chan fileBatchMsg // finder's fd stream → UI
	breadcrumb        breadcrumbPopup   // [b] ancestor-directory jump popup
	tasks             []landTask        // land operations (Tasks tab: running + log)
	taskCh            chan landMsg      // land goroutines → UI
	nextTaskID        int
	spinnerFrame      int               // running-task spinner animation
	spinning          bool              // a spinner tick is in flight
	taskCursor        int               // cursor over the Tasks tab
	watcher           *fsnotify.Watcher // live directory watch (nil if unavailable)
	watchCh           chan watchMsg     // watcher goroutine → UI
	watched           map[string]bool   // dirs currently registered with the watcher
}

// maxTabs caps panel [2]'s directory tabs. It opens with one (at the CWD); the
// user creates more with `t` (up to this many) and closes them with `w`.
const maxTabs = 5

// New returns the root model, focused on the file list. Panel [2] opens with a
// single tab at startDir (the CWD when empty — the `filu` no-arg case); when
// focusName is set (the `filu <file>` case) the cursor lands on that entry.
// Extra tabs the user created last session are restored by applyState.
func New(startDir, focusName string) AppModel {
	loadConfig() // apply config.yaml (finder cap) before any finder can open
	dir := startDir
	if dir == "" {
		wd, err := os.Getwd()
		if err != nil {
			wd = "/"
		}
		dir = wd
	}
	m := AppModel{focus: panelList, launchDir: dir, spaceMenu: newSpaceMenu(), sortMenu: newSortMenu(), quitMenu: newQuitMenu(), openWithMenu: newOpenWithMenu(), gotoMenu: newGotoMenu(), searchMenu: newSearchMenu(), openInMenu: newOpenInMenu(), confirm: newConfirmPopup(), inputPopup: newInputPopup(), help: newHelpPopup(), splash: newSplashModel(), toast: newToast(), detailYank: newDetailYank(), pty: newPtyPopup(), search: newSearch(), breadcrumb: newBreadcrumbPopup(), taskCh: make(chan landMsg, 64), searchCh: make(chan fileBatchMsg, 16), watched: map[string]bool{}}
	first := newList(dir)
	if focusName != "" && !first.focusEntry(focusName) { // `filu <file>`: land on the passed file
		first.showHidden = true // not listed — it's a dotfile; reveal hidden and retry
		first.reload()
		first.focusEntry(focusName)
	}
	m.tabs = []listModel{first}
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
		m.openInMenu.setSize(msg.Width)
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
		return m, tea.Batch(m.spaceMenu.handleTick(msg), m.sortMenu.handleTick(msg), m.quitMenu.handleTick(msg), m.openWithMenu.handleTick(msg), m.gotoMenu.handleTick(msg), m.openInMenu.handleTick(msg), m.searchMenu.handleTick(msg), m.confirm.handleTick(msg), m.inputPopup.handleTick(msg), m.help.handleTick(msg), m.toast.handleTick(msg), m.detailYank.handleTick(msg), m.pty.handleTick(msg), m.search.handleTick(msg), m.breadcrumb.handleTick(msg))
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
				switch m.confirmAction {
				case confirmDelete:
					_ = moveToTrash(m.pendingDelete)
					m.pendingDelete = ""
					m.cur().reload()
					m.cur().ensureVisible(m.listRows())
					m.refreshPreview()
				case confirmShell:
					cmd = tea.Batch(cmd, m.pty.start(buildShellCmd(), "Shell", m.cur().dir, m.width, m.height))
				case confirmOpen:
					cmd = tea.Batch(cmd, m.openDefault())
				case confirmUnfavorite:
					m.places.unpin(m.pendingUnfavorite)
					m.pendingUnfavorite = ""
					m.places.clampCursor()
					saveState(m.snapshotState())
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
		if m.openInMenu.isActive() { // Favorites "Open dir in…" picker
			if !m.openInMenu.isInteractive() {
				return m, nil
			}
			var key string
			var cmd tea.Cmd
			m.openInMenu, key, cmd = m.openInMenu.update(msg)
			if key != "" {
				cmd = tea.Batch(cmd, m.advanceOpenIn(key))
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
			m.setFocus(m.focus%3 + 1) // 1→2→3→1
		case "shift+tab":
			m.setFocus((m.focus+1)%3 + 1) // 1→3→2→1
		case "1":
			m.setFocus(panelList)
		case "2":
			m.setFocus(panelDetail)
		case "3":
			m.setFocus(panelMarks)
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
			m.marks.toggle(filepath.Join(l.dir, it.name))
		}
	case "c": // copy: land marked items here as copy (async)
		cmd = m.startLand(l.dir, false)
	case "v": // move: land marked items here as move (async)
		cmd = m.startLand(l.dir, true)
	case "f": // favorite: toggle the highlighted subdirectory into Favorites
		if it := l.cursorItem(); it.isDir {
			m.places.togglePin(filepath.Join(l.dir, it.name))
			m.places.clampCursor() // a removal may leave the Favorites cursor past the end
		}
	case "F": // favorite: toggle THIS tab's current directory (not the highlighted item)
		m.places.togglePin(l.dir)
		m.places.clampCursor()
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
	case "o": // open with the OS default app — confirm first (O opens the picker)
		if it := l.cursorItem(); it.name != "" {
			m.confirmAction = confirmOpen
			cmd = m.confirm.open("Open " + it.name + " with the default app?")
		}
	case "O": // Open with: pick an app (Default OS open, or a configured one)
		cmd = m.openOpenWith()
	case "s": // shell: confirm the directory first, then drop into $SHELL there
		m.confirmAction = confirmShell
		cmd = m.confirm.open("Open a shell in " + shortPath(l.dir) + "?")
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

// handleMarksKey routes keys to panel [3] (Marks | Tasks | Favorites): h/l cycle
// the tab, z zooms; then the active tab handles the rest — the Marks bucket, the
// Tasks land log, or the Favorites list.
func (m *AppModel) handleMarksKey(key string) tea.Cmd {
	switch key {
	case "l", "right":
		m.marksTab = (m.marksTab + 1) % 3 // Marks → Tasks → Favorites
		return nil
	case "h", "left":
		m.marksTab = (m.marksTab + 2) % 3
		return nil
	case "z":
		m.toggleZoom(panelMarks)
		return nil
	}
	switch m.marksTab {
	case 1: // Tasks tab
		return m.handleTasksKey(key)
	case 2: // Favorites tab
		return m.handleFavoritesKey(key)
	}
	// Marks tab
	switch key {
	case "j", "down":
		m.marks.moveCursor(1)
	case "k", "up":
		m.marks.moveCursor(-1)
	case "g":
		m.marks.cursor = 0
	case "G":
		m.marks.moveCursor(len(m.marks.items))
	case "p": // pick: toggle this item in the land subset
		m.marks.togglePick()
	case "m": // unmark: drop this item from the bucket (not the file)
		if m.marks.cursor >= 0 && m.marks.cursor < len(m.marks.items) {
			m.marks.removeItem(m.marks.items[m.marks.cursor])
		}
	case "y": // yank: copy this item's full path to the clipboard
		if m.marks.cursor >= 0 && m.marks.cursor < len(m.marks.items) {
			return copyToClipboardCmd(m.marks.items[m.marks.cursor], "Copied path to clipboard")
		}
	}
	return nil
}

// handleTasksKey routes keys to panel [3]'s Tasks tab: j/k move the cursor, D
// drops a task from the log.
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
	case "D": // delete: drop this task from the log
		if m.taskCursor >= 0 && m.taskCursor < len(m.tasks) {
			m.tasks = append(m.tasks[:m.taskCursor], m.tasks[m.taskCursor+1:]...)
			m.clampTaskCursor()
			saveState(m.snapshotState())
		}
	}
	return nil
}

// handleFavoritesKey routes keys to panel [3]'s Favorites tab: j/k move the
// cursor, D unfavorites the highlighted directory. Jumping to a favorite still
// lives in the Goto picker — this tab is purely for viewing and pruning the set.
func (m *AppModel) handleFavoritesKey(key string) tea.Cmd {
	switch key {
	case "j", "down":
		m.places.moveCursor(1)
	case "k", "up":
		m.places.moveCursor(-1)
	case "g":
		m.places.cursor = 0
	case "G":
		m.places.moveCursor(len(m.places.pinned))
	case "o": // open this favorite's dir in a tab (New tab / an existing tab)
		return m.openOpenInMenu()
	case "D": // unfavorite the highlighted directory — confirm first
		if m.places.cursor >= 0 && m.places.cursor < len(m.places.pinned) {
			p := m.places.pinned[m.places.cursor]
			m.pendingUnfavorite = p.path
			m.confirmAction = confirmUnfavorite
			return m.confirm.open("Unfavorite " + p.label + "?")
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
	case panelMarks:
		return m.handleMarksKey(key)
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
			itemOps = append(itemOps, menuItem{label: "Favorite", key: "f", hint: "favorite the highlighted subdirectory"})
		}
		if len(m.marks.items) > 0 {
			panelOps = append(panelOps,
				menuItem{label: "Copy", key: "c", hint: "land marked items as copy"},
				menuItem{label: "Move here", key: "v", hint: "land marked items as move"})
		}
		panelOps = append(panelOps,
			menuItem{label: "Search", key: "/", hint: "find a file by name or content in this tree"},
			menuItem{label: "Goto", key: "go", hint: "jump to a pinned dir, or search under home"},
			menuItem{label: "Favorite", key: "F", hint: "favorite this tab's current directory"},
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
			menuItem{label: "Sort", key: "S", hint: "order by a column (name / modified / perms / owner / size)"},
			menuItem{label: "Shell", key: "s", hint: "drop into $SHELL here (exit to return)"},
			menuItem{label: "Hidden", key: ".", hint: "toggle hidden files"},
			menuItem{label: "Zoom", key: "z", hint: "expand tabs to full-screen panels"})
		return groupedMenu(itemOps, panelOps), title
	case panelDetail:
		return groupedMenu(
			[]menuItem{{label: "Yank", key: "y", hint: "select & copy the preview"}},
			[]menuItem{{label: "Zoom", key: "z", hint: "expand the preview full-screen"}}), "Preview"
	case panelMarks:
		zoom := menuItem{label: "Zoom", key: "z", hint: "expand this panel full-screen"}
		tab := menuItem{label: "Switch tab", key: "l", hint: "Marks / Tasks / Favorites (h/l)"}
		switch m.marksTab {
		case 1: // Tasks tab
			var itemOps []menuItem
			if len(m.tasks) > 0 {
				itemOps = []menuItem{{label: "Delete", key: "D", hint: "remove this task from the log"}}
			}
			return groupedMenu(itemOps, []menuItem{tab, zoom}), "Tasks"
		case 2: // Favorites tab
			var itemOps []menuItem
			if len(m.places.pinned) > 0 {
				itemOps = []menuItem{
					{label: "Open in", key: "o", hint: "open this dir in a new or existing tab"},
					{label: "Delete", key: "D", hint: "unfavorite this directory"},
				}
			}
			return groupedMenu(itemOps, []menuItem{tab, zoom}), "Favorites"
		}
		var itemOps []menuItem
		if len(m.marks.items) > 0 {
			itemOps = []menuItem{
				{label: "Pick", key: "p", hint: "toggle this item in the land subset"},
				{label: "Yank", key: "y", hint: "copy full path to clipboard"},
				{label: "Unmark", key: "m", hint: "remove this item from the marks bucket"},
			}
		}
		return groupedMenu(itemOps, []menuItem{tab, zoom}), "Marks"
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
