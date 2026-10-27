package ui

import (
	_ "embed"
	"errors"
	"os"
	"path/filepath"
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
	tb                     *tasksBody
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

	finds    chan finderMsg // finder results from their goroutines
	findGen  int            // the search running now; older batches are dropped
	findStop func()         // ends the search running now
	finder   *terminu.Finder
	found    []terminu.FinderItem          // all the search running now has listed
	keep     func(terminu.FinderItem) bool // which of found the finder shows; nil for all

	term *terminu.Terminal // the shell, when one was opened

	quitDir     string       // where the shell goes when filu ends (cd-on-quit)
	startupErrs []string     // problems found while starting (no directory watch, a move that failed), shown once it runs
	startupNews []string     // what filu did while starting (moved the settings), said once it runs
	moveTitle   string       // the error popup's title for moveRows
	moveRows    []string     // moving the old settings failed, or some were in both places: why, with both paths, a row each
	unread      []unreadFile // filu's own files it could not read, asked about once it runs
	keepState   bool         // state.yaml or data.yaml could not be read: leave them as they are rather than write over them
}

// unreadFile is one of filu's own files it could not read, to be moved aside
// or to quit over (tdp E3, when a file cannot be read).
type unreadFile struct {
	path  string
	err   error
	fresh func() tea.Cmd // the start without it, once it is moved aside
}

// NewApp builds filu over startDir (the current directory when empty). With
// focusName set (`filu <file>`), the cursor starts on that entry of startDir.
func NewApp(startDir, focusName string) *App {
	mv := moveOldSettings()
	lm := moveLists()
	madeConfig, cfgErr := loadConfig()
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
	switch {
	case mv.err != nil:
		a.moveTitle = "Moving the settings failed"
		a.moveRows = append(a.moveRows, opFailedText("move filu's settings from "+mv.from+" to "+mv.to, mv.err)+".",
			"filu goes on using "+mv.from+". Move it to "+mv.to+" yourself to switch.")
	case len(mv.left) > 0:
		a.moveTitle = "Found in two places"
		var both []string
		for _, name := range mv.left {
			both = append(both, filepath.Join(mv.from, name)+" and "+filepath.Join(mv.to, name))
		}
		a.moveRows = append(a.moveRows, "These were in "+mv.from+" and in "+mv.to+" too: "+strings.Join(both, "; ")+".",
			"filu uses the ones in "+mv.to+" and left the others where they were: put what you need of them into "+
				mv.to+", then delete "+mv.from+".")
	}
	if len(mv.moved) > 0 {
		a.startupNews = append(a.startupNews, "Moved filu's settings from "+shortPath(mv.from)+" to "+shortPath(mv.to))
		for _, name := range mv.moved {
			a.t.Log(terminu.Info, "Moved "+shortPath(filepath.Join(mv.from, name))+" to "+shortPath(filepath.Join(mv.to, name)))
		}
	}
	if madeConfig != "" {
		a.t.Log(terminu.Info, "Wrote a commented config.yaml with the defaults at "+shortPath(madeConfig))
	}
	if cfgErr != nil {
		if p, ok := configFilePath(); ok {
			a.unread = append(a.unread, unreadFile{p, cfgErr, func() tea.Cmd {
				if err := writeDefaultConfig(p); err != nil {
					return a.t.Toast(terminu.Error, opFailedText("write a new config.yaml", err))
				}
				a.t.Log(terminu.Info, "Wrote a new config.yaml with the defaults at "+shortPath(p))
				return nil
			}})
		}
	}
	a.sayListsMove(lm)
	if lm.err == nil { // moved, or nothing to move: the lists are read from their files
		a.loadLists()
	}
	a.t.InputTabWidth = inputTabWidth
	a.t.HideSecretsInPreview = hideSecretsInPreview
	a.t.ShiftTab = true
	// 80 × 24, most terminals' size when they open, has room for every task;
	// so does 60 columns, [1], [2] and [3] then one at a time (dev-remarks).
	a.t.MinWidth, a.t.MinHeight = 60, 24
	a.t.Spinning = func() bool { return a.drawer.Tab == drawerTasks && a.anyRunning() }

	first := newTab(dir)
	if focusName != "" && !first.focus(focusName) {
		first.showHidden = true // not listed: a dotfile, so show the hidden ones and look again
		first.reload()
		first.focus(focusName)
	}
	a.tabs = []*tab{first}
	if failed, err := a.loadSession(); err != nil {
		// Left alone until the user says: writing over it would lose the
		// tabs, marks, favorites and sorts in it without a word (tdp E3, F5).
		a.keepState = true
		a.unread = append(a.unread, unreadFile{failed, err, a.reopenSession})
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

// loadSession brings back the session and the data, and moves the favorites
// and the tasks an older version left in state.yaml to data.yaml; failed is
// the file it could not read.
func (a *App) loadSession() (failed string, err error) {
	st, ok, moved, failed, err := loadState()
	if err != nil {
		return failed, err
	}
	if ok {
		a.applyState(st)
	}
	if len(leftInState.Favorites) > 0 || len(leftInState.Tasks) > 0 {
		sp, _ := stateFilePath()
		dp, _ := dataFilePath()
		if a.moveTitle == "" {
			a.moveTitle = "Found in two places"
		}
		a.moveRows = append(a.moveRows, "Favorites or tasks were in "+sp+" and in "+dp+" too.",
			"filu uses the ones in "+dp+" and left the others in "+sp+
				": put what you need of them into "+dp+", then take them out of "+sp+".")
	}
	if moved {
		if err := saveState(a.snapshotState()); err != nil {
			a.startupErrs = append(a.startupErrs, opFailedText("move the favorites and the task log", err))
		} else if p, ok := dataFilePath(); ok {
			a.startupNews = append(a.startupNews, "Moved the favorites and the task log to "+shortPath(p))
		}
	}
	return "", nil
}

// reopenSession is the start again once an unreadable state.yaml or
// data.yaml is moved aside; one more that cannot be read is asked about too.
// As on the first start, the new one is written at the first save (tdp E3).
func (a *App) reopenSession() tea.Cmd {
	a.keepState = false
	if failed, err := a.loadSession(); err != nil {
		a.keepState = true
		return a.t.MoveAside(failed, a.why(failed, err), a.reopenSession)
	}
	a.refreshAll()
	a.syncWatches()
	return a.said()
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
	// The names and the paths are long values, cut short as [3] narrows
	// (tdp panel.5).
	a.marksL = &terminu.List{Columns: []terminu.Column{{Title: "Name", Shrink: true}, {Title: "In", Path: true, Shrink: true}}, Marks: true,
		EmptyFact: "No marks yet", EmptyHint: "Press [m] on a file in [1] to mark it"}
	// Narrow, a task's words are cut short first, then its time goes.
	a.tasksL = &terminu.List{Columns: []terminu.Column{{Title: "When", Drop: 1}, {Title: "Task", Shrink: true}},
		EmptyFact: "No tasks yet", EmptyHint: "Mark files, then press [c] or [v] in [1] to copy or move them there"}
	a.tb = &tasksBody{List: a.tasksL, app: a}
	a.favsL = &terminu.List{Columns: []terminu.Column{{Title: "Name", Shrink: true}, {Title: "In", Path: true, Shrink: true}},
		EmptyFact: "No favorites yet", EmptyHint: "Press [f] on a directory in [1] to star it"}
	a.drawer = &terminu.Panel{Label: "Marks", Tabs: []string{"Tasks", "Favorites"}, Body: a.marksL, OnTab: a.drawerTab}
	a.drawer.Actions = a.drawerActions()
	a.screen = &terminu.Screen{Name: "Files", Panels: []*terminu.Panel{a.files, a.preview, a.drawer},
		Layout: a.layout}
	t.AddScreen(a.screen)
	t.OnMsg = a.onMsg
	t.Start = a.start
	t.Splash = splash()
}

// layout is [1] | [2] at 2:1 over the top two thirds, [3] across the rest.
// Zoomed, the focused panel takes the whole screen (tdp screen.6), but
// [1] takes the whole top and leaves [3] where it is (user 2026-10-24); a
// panel not drawn gets no room.
func (a *App) layout(w, h int) []terminu.Rect {
	topH := h * 2 / 3
	listW := w * 2 / 3
	if a.zoomed {
		rects := make([]terminu.Rect, 3)
		switch a.screen.Focused() {
		case a.files:
			rects[0] = terminu.Rect{W: w, H: topH}
			rects[2] = terminu.Rect{Y: topH, W: w, H: h - topH}
		case a.preview:
			rects[1] = terminu.Rect{W: w, H: h}
		default:
			rects[2] = terminu.Rect{W: w, H: h}
		}
		return rects
	}
	return []terminu.Rect{
		{X: 0, Y: 0, W: listW, H: topH},
		{X: listW, Y: 0, W: w - listW, H: topH},
		{X: 0, Y: topH, W: w, H: h - topH},
	}
}

// splash is filu's easter egg, V on a panel (tdp S1, S4).
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

// start reads the background channels, says what happened while starting,
// and asks about the files of filu's own it could not read.
func (a *App) start() tea.Cmd {
	return tea.Batch(a.waitLand(), a.waitWatch(), a.waitFind(), a.said(), a.askUnread())
}

// said says, once, what filu did and what went wrong while starting: a toast,
// or an error popup for several, or for a move's two paths, a row each (tdp
// components, control characters). Both go in the App Log (tdp F5).
func (a *App) said() tea.Cmd {
	var cmds []tea.Cmd
	if len(a.startupNews) > 0 {
		cmds = append(cmds, a.t.Toast(terminu.Info, a.startupNews...))
	}
	var errRows []string
	for _, e := range a.startupErrs {
		errRows = append(errRows, e+".")
	}
	switch {
	case len(a.moveRows) > 0:
		// Both paths, for the user to move it, or put the two together, by hand (tdp E3).
		cmds = append(cmds, a.t.Error(a.moveTitle, append(a.moveRows, errRows...)...))
	case len(a.startupErrs) == 1:
		cmds = append(cmds, a.t.Toast(terminu.Error, a.startupErrs[0]))
	case len(a.startupErrs) > 1:
		cmds = append(cmds, a.t.Error("Starting filu", errRows...))
	}
	a.startupNews, a.startupErrs, a.moveTitle, a.moveRows = nil, nil, "", nil
	return tea.Batch(cmds...)
}

// why is err as the confirm about path shows it: the line saying why when err
// has several (a yaml type error under its header), the whole of it in the
// App Log (tdp components, control characters).
func (a *App) why(path string, err error) error {
	var lines []string
	reason := ""
	for _, l := range strings.Split(err.Error(), "\n") {
		if l = strings.TrimSpace(l); l == "" {
			continue
		}
		lines = append(lines, l)
		if reason == "" && !strings.HasSuffix(l, ":") {
			reason = l
		}
	}
	if len(lines) < 2 {
		return err
	}
	if reason == "" {
		reason = lines[0]
	}
	a.t.Log(terminu.Error, "Cannot read "+shortPath(path)+": "+strings.Join(lines, " "))
	return errors.New(reason)
}

// askUnread asks about the first file of filu's own it could not read: moved
// aside, filu starts without it and asks about the next; or filu quits, the
// reason on stderr (tdp E3).
func (a *App) askUnread() tea.Cmd {
	if len(a.unread) == 0 {
		return nil
	}
	u := a.unread[0]
	a.unread = a.unread[1:]
	return a.t.MoveAside(u.path, a.why(u.path, u.err), func() tea.Cmd { return tea.Batch(u.fresh(), a.askUnread()) })
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
		return a.t.Toast(terminu.Success, msg.note)
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
