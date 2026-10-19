package ui

import (
	_ "embed"
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

	quitDir     string   // where the shell goes when filu ends (cd-on-quit)
	startupErrs []string // problems found while starting (a bad config.yaml, a broken state.yaml), shown once it runs
	startupNews []string // what filu did while starting (moved the settings), said once it runs
	moveTitle   string   // the error popup's title for moveFailed
	moveFailed  string   // moving the old settings failed, or some were in both places: why, with both paths
	keepState   bool     // state.yaml could not be read: leave it as it is rather than write over it
}

// NewApp builds filu over startDir (the current directory when empty). With
// focusName set (`filu <file>`), the cursor starts on that entry of startDir.
func NewApp(startDir, focusName string) *App {
	mv := moveOldSettings()
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
	switch {
	case mv.err != nil:
		a.moveTitle = "Moving the settings failed"
		a.moveFailed = opFailedText("move filu's settings from "+mv.from+" to "+mv.to, mv.err) +
			"\n\nfilu goes on using " + mv.from + ". Move it to " + mv.to + " yourself to switch."
	case len(mv.left) > 0:
		a.moveTitle = "Found in two places"
		var both []string
		for _, name := range mv.left {
			both = append(both, filepath.Join(mv.from, name)+"\n"+filepath.Join(mv.to, name))
		}
		a.moveFailed = "These were in " + mv.from + " and in " + mv.to + " too. filu uses the ones in " + mv.to +
			" and left the others where they were: put what you need of them into " + mv.to + ", then delete " + mv.from +
			".\n\n" + strings.Join(both, "\n\n")
	}
	if mv.moved {
		a.startupNews = append(a.startupNews, "Moved filu's settings from "+shortPath(mv.from)+" to "+shortPath(mv.to))
	}
	if cfgErr != nil {
		a.startupErrs = append(a.startupErrs, opFailedText("read config.yaml", cfgErr)+" (using the defaults)")
	}
	a.t.InputTabWidth = inputTabWidth
	a.t.HideSecretsInPreview = hideSecretsInPreview
	a.t.ShiftTab = true
	// 80 × 24, most terminals' size when they open, has room for every task;
	// so does 60 columns, [1], [2] and [3] then one at a time (dev-remarks).
	a.t.MinWidth, a.t.MinHeight = 60, 24
	a.t.Spinning = func() bool { return a.drawer.Tab == drawerTasks && a.anyRunning() }
	a.t.QuitDesc = "leave filu"

	first := newTab(dir)
	if focusName != "" && !first.focus(focusName) {
		first.showHidden = true // not listed: a dotfile, so show the hidden ones and look again
		first.reload()
		first.focus(focusName)
	}
	a.tabs = []*tab{first}
	st, ok, moved, err := loadState()
	switch {
	case err != nil:
		// Said, and left alone: writing over it would lose the tabs, marks,
		// favorites and sorts in it without a word (tdp F5).
		a.keepState = true
		a.startupErrs = append(a.startupErrs, opFailedText("read the session", err)+" (it is not saved until state.yaml and data.yaml are fixed or moved away)")
	case ok:
		a.applyState(st)
	}
	if len(leftInState.Favorites) > 0 || len(leftInState.Tasks) > 0 {
		sp, _ := stateFilePath()
		dp, _ := dataFilePath()
		if a.moveTitle == "" {
			a.moveTitle = "Found in two places"
		}
		a.moveFailed = strings.TrimPrefix(a.moveFailed+"\n\n", "\n\n") + "Favorites or tasks were in " + sp + " and in " + dp +
			" too. filu uses the ones in " + dp + " and left the others in " + sp +
			": put what you need of them into " + dp + ", then take them out of " + sp + "."
	}
	if moved {
		if err := saveState(a.snapshotState()); err != nil {
			a.startupErrs = append(a.startupErrs, opFailedText("move the favorites and the task log", err))
		} else if p, ok := dataFilePath(); ok {
			a.startupNews = append(a.startupNews, "Moved the favorites and the task log to "+shortPath(p))
		}
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
	// The names and the paths are long values, cut short as [3] narrows
	// (components/layout/panel).
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
		Layout: layout, NarrowBelow: 72}
	t.AddScreen(a.screen)
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
	if len(a.startupNews) > 0 {
		cmds = append(cmds, a.t.Toast(terminu.Info, strings.Join(a.startupNews, "; ")))
	}
	switch {
	case a.moveFailed != "":
		// Both paths, for the user to move it, or put the two together, by hand (tdp E3).
		cmds = append(cmds, a.t.Error(a.moveTitle, strings.Join(append([]string{a.moveFailed}, a.startupErrs...), "\n\n")))
	case len(a.startupErrs) == 1:
		cmds = append(cmds, a.t.Toast(terminu.Error, a.startupErrs[0]))
	case len(a.startupErrs) > 1:
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
