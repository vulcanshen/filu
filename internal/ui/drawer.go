package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	terminu "github.com/vulcanshen/terminu-go"
)

// [3] is Marks | Tasks | Favorites: the files gathered to land somewhere, the
// log of copies, moves and zips, and the starred directories.
const (
	drawerMarks = iota
	drawerTasks
	drawerFavs
)

// drawerTab is h|l on [3].
func (a *App) drawerTab(i int) tea.Cmd {
	switch i {
	case drawerTasks:
		a.drawer.Body = a.tasksL
	case drawerFavs:
		a.drawer.Body = a.favsL
	default:
		a.drawer.Body = a.marksL
	}
	return nil
}

// refreshDrawer fills [3]'s three lists from the model.
func (a *App) refreshDrawer() {
	var rows []terminu.Row
	for _, p := range a.marks.items {
		rows = append(rows, terminu.Row{ID: p, Cells: []string{pathIcon(p) + " " + safeName(filepath.Base(p)), safeName(shortPath(filepath.Dir(p)))}})
	}
	a.marksL.SetRows(rows)
	syncMarks(a.marksL, a.marks.picked)

	rows = nil
	for _, t := range a.tasks {
		rows = append(rows, terminu.Row{ID: strconv.Itoa(t.id), Cells: []string{
			terminu.Fg(terminu.Overlay0).Render(taskTime(t.at)), taskText(t)}})
	}
	a.tasksL.SetRows(rows)

	rows = nil
	for _, f := range a.favs.pinned {
		rows = append(rows, terminu.Row{ID: f.path, Cells: []string{terminu.Fg(terminu.Yellow).Render(iconPin) + " " + safeName(filepath.Base(f.path)),
			safeName(shortPath(filepath.Dir(f.path)))}})
	}
	a.favsL.SetRows(rows)
}

// taskText is a task in plain words: what it did to what, and where.
func taskText(t landTask) string {
	subject := fmt.Sprintf("%d items", t.total)
	if len(t.srcs) == 1 {
		subject = safeName(filepath.Base(t.srcs[0]))
	}
	verb, verbing, verbed := "Copy", "Copying", "Copied"
	switch t.action {
	case "mv":
		verb, verbing, verbed = "Move", "Moving", "Moved"
	case "zip":
		verb, verbing, verbed = "Zip", "Zipping", "Zipped"
	}
	to := " → " + safeName(t.dest)
	grey := terminu.Fg(terminu.Overlay0)
	switch t.status {
	case taskRunning: // a row that is still going, the app's to show (tdp F7)
		return terminu.Fg(terminu.Green).Render(glyphRunning) + " " + verbing + " " + subject + to + "  " + grey.Render(fmt.Sprintf("%d/%d", t.done, t.total))
	case taskDone:
		return terminu.Fg(terminu.Green).Render(glyphDone) + " " + verbed + " " + subject + to
	case taskPending:
		return terminu.Fg(terminu.Peach).Render(glyphPending) + " " + verb + " " + subject + to + "  " + grey.Render("· interrupted")
	case taskError:
		return terminu.Fg(terminu.Red).Render(glyphFailed) + " " + verb + " " + subject + to + "  " + terminu.Fg(terminu.Red).Render(fmt.Sprintf("· %d/%d failed", t.failed, t.total))
	}
	return verb + " " + subject + to
}

// The glyphs of a task's state.
const (
	glyphRunning = "\U000F051F" // nf-md-timer_sand; TODO(lib FL11): the family's turning loading icon
	glyphDone    = "\U0000F00C" // nf-fa-check
	glyphPending = "\U0000F017" // nf-fa-clock_o
	glyphFailed  = "\U0000F00D" // nf-fa-times
)

// drawerActions are [3]'s, each tab's own.
func (a *App) drawerActions() []terminu.Action {
	item, panel := terminu.ItemScope, terminu.PanelScope
	not := func(tab int) func() bool { return func() bool { return a.drawer.Tab != tab } }
	return []terminu.Action{
		// Marks
		{Key: "Enter", Label: "Show", Desc: "it in [1]: the tab on its directory, or a new one", Scope: item, Hint: "show",
			Hidden: not(drawerMarks), Do: func() tea.Cmd {
				r, _ := a.marksL.Current()
				return a.showInTabs(filepath.Dir(r.ID), r.ID)
			}},
		{Key: "p", Label: "pick", Desc: "it: when you pick some, only the picks land", Scope: item, Hint: "pick",
			Hidden: not(drawerMarks), Do: func() tea.Cmd {
				r, _ := a.marksL.Current()
				a.marks.cursor = max(indexOf(a.marks.items, r.ID), 0)
				a.marks.togglePick()
				a.marksChanged()
				return a.persist()
			}},
		{Key: "y", Label: "yank", Desc: "its full path, to the clipboard", Scope: item,
			Hidden: not(drawerMarks), Do: func() tea.Cmd {
				r, _ := a.marksL.Current()
				return copyToClipboardCmd(r.ID, "Copied "+safeName(r.ID))
			}},
		{Key: "m", Label: "unmark", Desc: "it: out of the marks", Scope: item, Hint: "unmark",
			Hidden: not(drawerMarks), Do: func() tea.Cmd {
				r, _ := a.marksL.Current()
				a.marks.removeItem(r.ID)
				a.marksChanged()
				return a.persist()
			}},
		{Key: "Z", Label: "Zip", Desc: "the picks, or every mark, into one zip to land", Scope: panel, Hint: "zip",
			Hidden: not(drawerMarks), Disabled: a.nothingMarked, Do: a.zip},
		{Key: "C", Label: "Clear", Desc: "every mark; the files stay where they are", Scope: panel, Hint: "clear",
			Hidden: not(drawerMarks), Ask: a.clearAsk,
			Disabled: asking(a.clearAsk, func(k *terminu.Ask) {
				k.Question = fmt.Sprintf("Clear all %d marks?", len(a.marks.items))
			}, func() bool { return len(a.marks.items) == 0 }),
			Do: func() tea.Cmd {
				a.marks.clear()
				a.marksChanged()
				return a.persist()
			}},
		// Tasks
		{Key: "Enter", Label: "Go to", Desc: "where it landed, in the tab [1] shows", Scope: item, Hint: "go",
			Hidden: not(drawerTasks), Do: func() tea.Cmd {
				if t := a.currentTask(); t != nil {
					return tea.Batch(a.cd(t.destPath, ""), a.t.Toast(terminu.Info, "[1] is on "+safeName(shortPath(t.destPath))))
				}
				return nil
			}},
		{Key: "D", Label: "Delete", Desc: "it from the log", Scope: item,
			Hidden: not(drawerTasks), Disabled: func() bool { t := a.currentTask(); return t != nil && t.status == taskRunning },
			Do: func() tea.Cmd {
				if t := a.currentTask(); t != nil {
					a.removeTask(t.id)
					a.refreshDrawer()
					return a.persist()
				}
				return nil
			}},
		// Favorites
		{Key: "Enter", Label: "Go", Desc: "there: the tab on it, or a new one", Scope: item, Hint: "go",
			Hidden: not(drawerFavs), Do: func() tea.Cmd {
				r, _ := a.favsL.Current()
				return a.showInTabs(r.ID, "")
			}},
		{Key: "o", Label: "open in", Desc: "the tab you choose, or a new one", Scope: item, Hint: "open in",
			Hidden: not(drawerFavs), Do: func() tea.Cmd {
				r, _ := a.favsL.Current()
				return a.openInMenu(r.ID)
			}},
		{Key: "D", Label: "Unfavorite", Desc: "it: take its star off", Scope: item,
			Hidden: not(drawerFavs), Ask: a.unfavAsk,
			Disabled: asking(a.unfavAsk, func(k *terminu.Ask) {
				r, _ := a.favsL.Current()
				where := safeName(shortPath(r.ID))
				k.Subject, k.Question = where, "Take the star off "+where+"?"
			}, nil),
			Do: func() tea.Cmd {
				r, _ := a.favsL.Current()
				return a.toggleFavorite(r.ID)
			}},
		a.zoomAction(),
	}
}

func indexOf(s []string, v string) int {
	for i, x := range s {
		if x == v {
			return i
		}
	}
	return -1
}

func (a *App) currentTask() *landTask {
	r, ok := a.tasksL.Current()
	if !ok {
		return nil
	}
	for i := range a.tasks {
		if strconv.Itoa(a.tasks[i].id) == r.ID {
			return &a.tasks[i]
		}
	}
	return nil
}

func (a *App) removeTask(id int) {
	for i := range a.tasks {
		if a.tasks[i].id == id {
			a.tasks = append(a.tasks[:i], a.tasks[i+1:]...)
			return
		}
	}
}

// showInTabs brings dir up in [1]: the tab on it if there is one, else a new
// tab there; with every tab in use it says so. With focus set (a path) the
// cursor goes there. The focus stays where it is (tdp K3): a toast says
// where to look.
func (a *App) showInTabs(dir, focus string) tea.Cmd {
	at := -1
	for i, t := range a.tabs {
		if cleanDir(t.dir) == cleanDir(dir) {
			at = i
			break
		}
	}
	if at < 0 {
		if len(a.tabs) >= maxTabs {
			return a.t.Toast(terminu.Warning, fmt.Sprintf("All %d tabs are in use: close one with [w] to open %s", maxTabs, safeName(filepath.Base(dir))))
		}
		a.openTab(dir, "")
		at = len(a.tabs) - 1
	}
	a.switchTab(at)
	a.syncTabs()
	t := a.cur()
	if focus != "" {
		if _, ok := t.byPath[focus]; !ok && !t.showHidden { // a dotfile the tab hides
			t.showHidden = true
			t.reload()
			a.fillTab(t)
		}
		t.list.Select(focus)
	}
	a.refreshPreview()
	return tea.Batch(a.t.Toast(terminu.Info, "In [1]: "+safeName(shortPath(dir))), a.persist())
}

// openInMenu puts a favorite in the tab chosen, or a new one.
func (a *App) openInMenu(dir string) tea.Cmd {
	acts := []terminu.Action{{Key: "n", Label: "New tab", Desc: "a tab of its own", Disabled: func() bool { return len(a.tabs) >= maxTabs },
		Do: func() tea.Cmd { return a.openTab(dir, "") }}}
	for i, t := range a.tabs {
		i, t := i, t
		desc := "now on " + safeName(shortPath(t.dir))
		if cleanDir(t.dir) == cleanDir(dir) {
			desc = "already there"
		}
		acts = append(acts, terminu.Action{Key: strconv.Itoa(i + 1), Label: tabMark(i) + " Tab " + strconv.Itoa(i+1), Desc: desc,
			Do: func() tea.Cmd {
				a.switchTab(i)
				a.syncTabs()
				return a.cd(dir, "")
			}})
	}
	return a.t.Menu("Open in", // TODO(lib FL14): a menu title cannot keep a name as it is
		terminu.MenuSection{Actions: acts})
}

// --- tasks: copies, moves and zips in the background --------------------------

func (a *App) waitLand() tea.Cmd {
	ch := a.taskCh
	return func() tea.Msg { return <-ch }
}

func (a *App) anyRunning() bool {
	for _, t := range a.tasks {
		if t.status == taskRunning {
			return true
		}
	}
	return false
}

// startLand copies or moves items into destDir, logged in the Tasks tab (and
// in the session, as unfinished, until it ends).
func (a *App) startLand(items []string, destDir string, move bool) tea.Cmd {
	if len(items) == 0 {
		return nil
	}
	action := "cp"
	if move {
		action = "mv"
	}
	id := a.addTask(action, filepath.Base(destDir), destDir, items)
	go runLand(id, items, destDir, move, a.taskCh)
	return a.persist()
}

func (a *App) addTask(action, dest, destPath string, items []string) int {
	a.nextTaskID++
	a.tasks = append(a.tasks, landTask{id: a.nextTaskID, action: action, dest: dest, destPath: destPath, srcs: items,
		total: len(items), at: time.Now(), status: taskRunning})
	if len(a.tasks) > maxTasks {
		a.tasks = a.tasks[len(a.tasks)-maxTasks:]
	}
	a.refreshDrawer()
	a.syncQuit()
	return a.nextTaskID
}

// handleLand is a task's progress or its end.
func (a *App) handleLand(msg landMsg) tea.Cmd {
	for i := range a.tasks {
		t := &a.tasks[i]
		if t.id != msg.taskID {
			continue
		}
		t.done = msg.done
		if !msg.finished {
			a.refreshDrawer()
			return nil
		}
		t.failed = msg.failed
		t.status = taskDone
		if msg.failed > 0 {
			t.status = taskError
		}
		for _, src := range msg.moved {
			a.marks.removeItem(src)
		}
		if msg.produced != "" { // a zip: the only pick, ready to land
			a.marks.items = append(a.marks.items, msg.produced)
			a.marks.picked = map[string]bool{msg.produced: true}
		}
		a.reloadDir(t.destPath)
		for _, src := range msg.moved {
			a.reloadDir(filepath.Dir(src))
		}
		a.marksChanged()
		a.refreshPreview()
		a.syncQuit()
		toast := a.t.Toast(terminu.Info, terminu.StripStyles(taskText(*t)))
		if t.status == taskError {
			toast = a.t.Toast(terminu.Error, terminu.StripStyles(taskText(*t)))
		}
		return tea.Batch(toast, a.persist())
	}
	return nil
}

// zip packs the picks (or every mark) into a zip in a temp directory: it joins
// the marks as the only pick, to land with [c] or [v].
func (a *App) zip() tea.Cmd {
	items := a.marks.landItems()
	if len(items) == 0 {
		return nil
	}
	return a.t.Text(terminu.TextInput{
		// A grey suggestion is only ever the value now (components/input/README);
		// a new zip has none, so the name filu would choose is in the prompt.
		// TODO(lib FL9): the suggested name goes in grey, Enter takes it (tdp S6).
		Title: "Zip", Glyph: glyphZip, Name: "The name", Verb: "zip",
		Prompt: zipPrompt(len(items)) + ": a name for the zip, like " + suggestZipName(items),
		Check: func(v string) error {
			if zipFileName(v) == "" {
				return fmt.Errorf("Not a name for a zip file")
			}
			return nil
		},
		Done: func(v string) tea.Cmd {
			name := zipFileName(v)
			dir, err := os.MkdirTemp("", "filu-zip-")
			if err != nil {
				return a.t.Error("Zip failed", opFailedText("make a temporary directory", err))
			}
			id := a.addTask("zip", name, dir, items)
			go runZip(id, items, filepath.Join(dir, name), a.taskCh)
			return a.persist()
		},
	})
}

// --- quitting ------------------------------------------------------------------

// syncQuit writes into the quit confirm where the shell will be, and warns
// while a task is still running (tdp K9: the quit flow is the app's).
func (a *App) syncQuit() {
	ask := &terminu.Ask{Title: "Quit", Question: "Quit filu?", Verb: "quit"}
	if os.Getenv(envLastDirFile) != "" {
		ask.Details = []string{"Your shell goes to " + safeName(shortPath(a.cur().dir)) + ".",
			"[Q]uit to chooses another directory."}
	}
	if a.anyRunning() {
		if len(ask.Details) > 0 {
			ask.Details = append(ask.Details, "")
		}
		ask.Warn = []int{len(ask.Details)}
		ask.Details = append(ask.Details, "A task is still running: quitting stops it.")
	}
	a.t.Quit = ask
}

// quitToMenu chooses where the shell goes, then leaves: the directory filu
// started in, then each tab's.
// TODO(lib FL1): this is what q would open if the quit flow could be a menu.
func (a *App) quitToMenu() tea.Cmd {
	var acts []terminu.Action
	seen := map[string]bool{}
	add := func(dir, from string) {
		if dir == "" || seen[dir] {
			return
		}
		seen[dir] = true
		acts = append(acts, terminu.Action{Key: strconv.Itoa(len(acts) + 1), Label: safeName(shortPath(dir)), Desc: from,
			Do: func() tea.Cmd {
				a.quitDir = dir
				return tea.Quit
			}})
	}
	add(a.launchDir, "where filu started")
	for i, t := range a.tabs {
		add(t.dir, "tab "+strconv.Itoa(i+1))
	}
	sections := []terminu.MenuSection{{Title: "your shell goes to", Actions: acts}}
	if a.anyRunning() {
		return a.t.Confirm(terminu.Ask{Title: "Quit to", Question: "A task is still running. Quit anyway?", Verb: "choose",
			Details: []string{"Quitting stops it."}, Warn: []int{0}}, func() tea.Cmd { return a.t.Menu("Quit to", sections...) })
	}
	return a.t.Menu("Quit to", sections...)
}
