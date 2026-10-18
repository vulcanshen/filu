package ui

import (
	_ "embed"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/fsnotify/fsnotify"
	terminu "github.com/vulcanshen/terminu-go"

	"github.com/vulcanshen/filu/internal/version"
)

// icon is the family mark the splash draws (tdp S5): a copy of docs/icon.svg,
// which go:embed cannot reach. TestIconMatchesDocs keeps the two the same
// (tdp E6).
//
//go:embed icon.svg
var icon []byte

// splashIntro is the line under the name and the version on the splash.
const splashIntro = "A terminal file manager you don't have to learn"

// App is filu on terminu-go: one screen of three panels — [1] the directory
// tabs, [2] the preview of the item under [1]'s cursor, [3] Marks | Tasks |
// Favorites. terminu draws it and owns the core keys, the Space menu, the key
// reference, the popups and their stack; App keeps the files.
type App struct {
	t      *terminu.App
	screen *terminu.Screen

	files, preview, drawer *terminu.Panel
	fb                     *filesBody
	pb                     *previewBody
	marksL, tasksL, favsL  *terminu.List

	tabs      []*tab
	active    int    // the tab [1] shows
	launchDir string // where filu started: the first tab, and a quit destination
	zoomed    bool   // z: the focused panel takes the whole screen

	marks marksModel  // the bucket: marked paths and the picks among them
	favs  placesModel // starred directories

	tasks      []landTask
	taskCh     chan landMsg
	nextTaskID int

	watcher *fsnotify.Watcher
	watchCh chan watchMsg
	watched map[string]bool

	finds   chan finderMsg // finder results from their goroutines
	findGen int            // the finder open now; older batches are dropped
	finder  *terminu.Finder
	hits    map[string]int // a content search's first matching line of each file

	// The confirms of the actions that ask first, filled in when they run
	// (asking).
	openAsk, deleteAsk, shellAsk, clearAsk, unfavAsk *terminu.Ask
	term                                             *terminu.Terminal // the shell, when one was opened

	quitDir     string   // where the shell goes when filu ends (cd-on-quit)
	startupErrs []string // problems found while starting (a bad config.yaml, a broken state.yaml), shown once it runs
	keepState   bool     // state.yaml could not be read: leave it as it is rather than write over it
}

// NewApp builds filu over startDir (the current directory when empty). With
// focusName set (`filu <file>`), the cursor starts on that entry of startDir.
func NewApp(startDir, focusName string) *App {
	cfgErr := loadConfig()
	dir := startDir
	if dir == "" {
		wd, err := os.Getwd()
		if err != nil {
			wd = "/"
		}
		dir = wd
	}
	a := &App{
		t:         terminu.New("filu"),
		launchDir: dir,
		taskCh:    make(chan landMsg, 64),
		finds:     make(chan finderMsg, 16),
		watched:   map[string]bool{},
	}
	if cfgErr != nil {
		a.startupErrs = append(a.startupErrs, opFailedText("read config.yaml", cfgErr)+" (using the defaults)")
	}
	a.openAsk = &terminu.Ask{Title: "Open", Verb: "open"}
	a.deleteAsk = &terminu.Ask{Title: "Delete", Verb: "delete"}
	a.shellAsk = &terminu.Ask{Title: "Shell", Verb: "open"}
	a.clearAsk = &terminu.Ask{Title: "Clear", Verb: "clear"}
	a.unfavAsk = &terminu.Ask{Title: "Unfavorite", Verb: "unfavorite"}
	a.t.InputTabWidth = inputTabWidth
	a.t.HideSecretsInPreview = hideSecretsInPreview
	a.t.ShiftTab = true // TODO(lib FL7): the key reference does not list Shift-Tab
	a.t.QuitDesc = "leave filu, your shell in the directory it says"

	first := newTab(dir)
	if focusName != "" && !first.focus(focusName) {
		first.showHidden = true // not listed: a dotfile, so show the hidden ones and look again
		first.reload()
		first.focus(focusName)
	}
	a.tabs = []*tab{first}
	st, ok, err := loadState()
	switch {
	case err != nil:
		// Said, and left alone: writing over it would lose the tabs, marks,
		// favorites and sorts in it without a word (tdp F5).
		a.keepState = true
		a.startupErrs = append(a.startupErrs, opFailedText("read state.yaml", err)+" (the session is not saved until it is fixed or moved away)")
	case ok:
		a.applyState(st)
	}
	a.build()
	a.refreshAll()
	if a.watcher = newWatcher(); a.watcher != nil {
		a.watchCh = make(chan watchMsg, 16)
		a.syncWatches()
		go watchLoop(a.watcher, a.watchCh)
	} else {
		a.startupErrs = append(a.startupErrs, "Cannot watch the directories: what other programs change shows up only when you open the directory again")
	}
	return a
}

// build lays out the screen: [1] and [2] side by side at 2:1 on top, [3]
// across the bottom.
func (a *App) build() {
	t := a.t
	a.fb = &filesBody{app: a}
	a.files = &terminu.Panel{Body: a.fb, OnTab: a.switchTab, OnEsc: a.up}
	a.files.Actions = a.filesActions()
	a.pb = &previewBody{app: a, Pager: &terminu.Pager{}}
	a.preview = &terminu.Panel{Label: "Preview", Body: a.pb}
	a.preview.Actions = a.previewActions()
	a.files.Details = a.preview
	a.marksL = &terminu.List{Columns: []terminu.Column{{Title: "Name"}, {Title: "In", Path: true, Flex: true}}, Marks: true,
		EmptyFact: "No marks yet", EmptyHint: "Press [m] on a file in [1] to mark it"}
	a.tasksL = &terminu.List{Columns: []terminu.Column{{Title: "When"}, {Title: "Task", Flex: true}},
		EmptyFact: "No tasks yet", EmptyHint: "Mark files, then press [c] or [v] in [1] to copy or move them there"}
	a.favsL = &terminu.List{Columns: []terminu.Column{{Title: "Name"}, {Title: "In", Path: true, Flex: true}},
		EmptyFact: "No favorites yet", EmptyHint: "Press [f] on a directory in [1] to star it"}
	a.drawer = &terminu.Panel{Label: "Marks", Tabs: []string{"Tasks", "Favorites"}, Body: a.marksL, OnTab: a.drawerTab}
	a.drawer.Actions = a.drawerActions()
	a.screen = &terminu.Screen{Name: "Files", Panels: []*terminu.Panel{a.files, a.preview, a.drawer},
		Layout: layout, NarrowBelow: 72}
	t.AddScreen(a.screen)
	t.Global = a.globalActions()
	t.OnMsg = a.onMsg
	t.Start = a.start
	t.Splash = splash()
}

// layout is [1] | [2] at 2:1 over the top two thirds, [3] across the rest.
func layout(w, h int) []terminu.Rect {
	topH := h * 2 / 3
	listW := w * 2 / 3
	return []terminu.Rect{
		{X: 0, Y: 0, W: listW, H: topH},
		{X: listW, Y: 0, W: w - listW, H: topH},
		{X: 0, Y: topH, W: w, H: h - topH},
	}
}

// splash is filu's easter egg, V on a panel (tdp S).
func splash() terminu.Splash {
	s, err := terminu.NewSplash(icon, "filu", version.Display(), splashIntro)
	if err != nil {
		panic(err) // the embedded icon is broken: a mistake in the program
	}
	return s
}

// Run asks the terminal how wide an icon is, runs filu, and saves the session
// when it ends.
func (a *App) Run() error {
	err := a.t.Run()
	a.shutdown()
	return err
}

// start reads the background channels and says what went wrong while starting.
func (a *App) start() tea.Cmd {
	cmds := []tea.Cmd{a.waitLand(), a.waitWatch(), a.waitFind()}
	switch len(a.startupErrs) {
	case 0:
	case 1:
		cmds = append(cmds, a.t.Toast(terminu.Error, a.startupErrs[0]))
	default:
		cmds = append(cmds, a.t.Error("Starting filu", strings.Join(a.startupErrs, "\n\n")))
	}
	return tea.Batch(cmds...)
}

// shutdown saves the session, stops the watcher and the shell, and tells the
// shell filu runs in where to go.
func (a *App) shutdown() {
	if !a.keepState {
		_ = saveState(a.snapshotState())
	}
	if a.watcher != nil {
		a.watcher.Close()
	}
	if a.term != nil && a.term.Running() {
		a.term.Stop()
	}
	dir := a.quitDir
	if dir == "" {
		dir = a.cur().dir
	}
	writeLastDir(dir)
}

// onMsg takes the messages terminu leaves to the app.
func (a *App) onMsg(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case landMsg:
		return tea.Batch(a.handleLand(msg), a.waitLand())
	case watchMsg:
		a.handleWatch(msg)
		return a.waitWatch()
	case finderMsg:
		a.handleFind(msg)
		return a.waitFind()
	case clipboardCopiedMsg:
		return a.t.Toast(terminu.Info, msg.note)
	case clipboardFailedMsg:
		return a.t.Toast(terminu.Error, "Could not copy to the clipboard")
	case opFailedMsg:
		return a.t.Toast(terminu.Error, msg.text)
	}
	return nil
}

// cur is the tab [1] shows.
func (a *App) cur() *tab { return a.tabs[a.active] }

// refreshAll redraws every list from the model: the tabs, [3], the preview
// and what quitting says.
func (a *App) refreshAll() {
	for _, t := range a.tabs {
		a.fillTab(t)
	}
	a.syncTabs()
	a.refreshDrawer()
	a.refreshPreview()
	a.syncQuit()
}

// persist saves the session now; a failure is said (tdp F5): it would lose the
// tabs, marks, favorites and sorts.
func (a *App) persist() tea.Cmd {
	if a.keepState {
		return nil
	}
	if err := saveState(a.snapshotState()); err != nil {
		return a.t.Toast(terminu.Error, opFailedText("save the session", err))
	}
	return nil
}
