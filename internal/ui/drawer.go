package ui

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
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
		a.drawer.Body = a.tb
	case drawerFavs:
		a.drawer.Body = a.favsL
	default:
		a.drawer.Body = a.marksL
	}
	return nil
}

// tasksBody is the Tasks tab: its list, a running task's loading icon turning
// as it is drawn (App.Spinning keeps the frames coming).
type tasksBody struct {
	*terminu.List
	app *App
}

func (b *tasksBody) Draw(w, h int, focused bool) []string {
	if b.app.anyRunning() {
		b.app.fillTasks()
	}
	return b.List.Draw(w, h, focused)
}

// refreshDrawer fills [3]'s three lists from the model.
func (a *App) refreshDrawer() {
	var rows []terminu.Row
	for _, p := range a.marks.items {
		rows = append(rows, terminu.Row{ID: p, Cells: []string{pathIcon(p) + " " + filepath.Base(p), shortPath(filepath.Dir(p))}})
	}
	a.marksL.SetRows(rows)
	var picked []string
	for p := range a.marks.picked {
		picked = append(picked, p)
	}
	a.marksL.SetMarked(picked)

	a.fillTasks()

	rows = nil
	for _, f := range a.favs.pinned {
		name := []terminu.Span{{Text: iconPin, Style: terminu.Fg(terminu.Yellow)}, {Text: " " + filepath.Base(f.path)}}
		rows = append(rows, terminu.Row{ID: f.path, Cells: []string{spansText(name), shortPath(filepath.Dir(f.path))},
			Styled: [][]terminu.Span{name}})
	}
	a.favsL.SetRows(rows)
}

// fillTasks fills the Tasks tab from the task log.
func (a *App) fillTasks() {
	var rows []terminu.Row
	for _, t := range a.tasks {
		when := []terminu.Span{{Text: taskTime(t.at), Style: terminu.Fg(terminu.Overlay0)}}
		task := taskText(t)
		rows = append(rows, terminu.Row{ID: strconv.Itoa(t.id), Cells: []string{spansText(when), spansText(task)},
			Styled: [][]terminu.Span{when, task}})
	}
	a.tasksL.SetRows(rows)
}

// taskText is a task in plain words: what it did to what, and where; the
// state's glyph and the count in their colours.
func taskText(t landTask) []terminu.Span {
	subject := fmt.Sprintf("%d items", t.total)
	if len(t.srcs) == 1 {
		subject = filepath.Base(t.srcs[0])
	}
	verb, verbing, verbed := "Copy", "Copying", "Copied"
	switch t.action {
	case "mv":
		verb, verbing, verbed = "Move", "Moving", "Moved"
	case "zip":
		verb, verbing, verbed = "Zip", "Zipping", "Zipped"
	}
	to := " → " + t.dest
	grey := terminu.Fg(terminu.Overlay0)
	line := func(glyph string, c terminu.Color, words, tail string, tailStyle terminu.Style) []terminu.Span {
		out := []terminu.Span{{Text: glyph, Style: terminu.Fg(c)}, {Text: " " + words + " " + subject + to}}
		if tail != "" {
			out = append(out, terminu.Span{Text: "  "}, terminu.Span{Text: tail, Style: tailStyle})
		}
		return out
	}
	switch t.status {
	case taskRunning: // a row that is still going, the app's to show (tdp F7, screen.3)
		return line(terminu.LoadingFrame(), terminu.Green, verbing, fmt.Sprintf("%d/%d", t.done, t.total), grey)
	case taskDone:
		return line(glyphDone, terminu.Green, verbed, "", grey)
	case taskPending:
		return line(glyphPending, terminu.Peach, verb, "· interrupted", grey)
	case taskError:
		return line(glyphFailed, terminu.Red, verb, fmt.Sprintf("· %d/%d failed", t.failed, t.total), terminu.Fg(terminu.Red))
	}
	return []terminu.Span{{Text: verb + " " + subject + to}}
}

// The glyphs of a task's state; a running one has the turning loading icon.
const (
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
		{Key: "p", Label: "Pick", Desc: "it: when you pick some, only the picks land", Scope: item, Hint: "pick",
			Hidden: not(drawerMarks), Do: func() tea.Cmd {
				r, _ := a.marksL.Current()
				a.marks.cursor = max(indexOf(a.marks.items, r.ID), 0)
				a.marks.togglePick()
				a.marksChanged()
				return a.persist()
			}},
		{Key: "y", Label: "Yank", Desc: "its full path, to the clipboard", Scope: item,
			Hidden: not(drawerMarks), Do: func() tea.Cmd {
				r, _ := a.marksL.Current()
				return copyToClipboardCmd(r.ID, "Copied "+r.ID)
			}},
		{Key: "m", Label: "Unmark", Desc: "it: out of the marks", Scope: item, Hint: "unmark",
			Hidden: not(drawerMarks), Do: func() tea.Cmd {
				r, _ := a.marksL.Current()
				a.marks.removeItem(r.ID)
				a.marksChanged()
				return a.persist()
			}},
		{Key: "Z", Label: "Zip", Desc: "the picks, or every mark, into one zip to land", Scope: panel, Hint: "zip",
			Hidden: not(drawerMarks), Disabled: a.nothingMarked, Do: a.zip},
		{Key: "C", Label: "Clear", Desc: "every mark; the files stay where they are", Scope: panel, Hint: "clear",
			Hidden: not(drawerMarks), Disabled: func() bool { return len(a.marks.items) == 0 },
			AskFor: func() *terminu.Ask {
				return &terminu.Ask{Title: "Clear", Question: fmt.Sprintf("Clear all %d marks?", len(a.marks.items)), Verb: "clear"}
			},
			Do: func() tea.Cmd {
				a.t.Log(terminu.Success, fmt.Sprintf("Cleared %d marks", len(a.marks.items)))
				a.marks.clear()
				a.marksChanged()
				return a.persist()
			}},
		// Tasks
		{Key: "Enter", Label: "Go to", Desc: "where it landed, in the tab [1] shows", Scope: item, Hint: "go",
			Hidden: not(drawerTasks), Do: func() tea.Cmd {
				if t := a.currentTask(); t != nil {
					return tea.Batch(a.cd(t.destPath, ""), a.t.Focus(a.files))
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
		{Key: "o", Label: "Open in", Desc: "the tab you choose, or a new one", Scope: item, Hint: "open in",
			Hidden: not(drawerFavs), Do: func() tea.Cmd {
				r, _ := a.favsL.Current()
				return a.openInMenu(r.ID)
			}},
		{Key: "D", Label: "Unfavorite", Desc: "it: take its star off", Scope: item,
			Hidden: not(drawerFavs),
			AskFor: func() *terminu.Ask {
				r, _ := a.favsL.Current()
				where := shortPath(r.ID)
				return &terminu.Ask{Title: "Unfavorite", Subject: where, Question: "Take the star off " + where + "?", Verb: "unfavorite"}
			},
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
// cursor goes there, and the focus goes to [1], as the user going there
// expects (tdp K3).
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
			return a.t.Toast(terminu.Warning, fmt.Sprintf("All %d tabs are in use: close one with [w] to open %s", maxTabs, filepath.Base(dir)))
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
	return tea.Batch(a.t.Focus(a.files), a.persist())
}

// openInMenu puts a favorite in the tab chosen, or a new one; the focus goes
// to [1], to it.
func (a *App) openInMenu(dir string) tea.Cmd {
	acts := []terminu.Action{{Key: "n", Label: "New tab", Desc: "a tab of its own", Disabled: func() bool { return len(a.tabs) >= maxTabs },
		Do: func() tea.Cmd { return tea.Batch(a.openTab(dir, ""), a.t.Focus(a.files)) }}}
	for i, t := range a.tabs {
		i, t := i, t
		desc := "now on " + shortPath(t.dir)
		if cleanDir(t.dir) == cleanDir(dir) {
			desc = "already there"
		}
		acts = append(acts, terminu.Action{Key: strconv.Itoa(i + 1), Label: tabMark(i) + " Tab " + strconv.Itoa(i+1), Desc: desc,
			Do: func() tea.Cmd {
				a.switchTab(i)
				a.syncTabs()
				return tea.Batch(a.cd(dir, ""), a.t.Focus(a.files))
			}})
	}
	return a.t.MenuAbout("Open in", shortPath(dir), terminu.MenuSection{Actions: acts})
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
	subject, into := fmt.Sprintf("%d items", len(items)), destPath
	if len(items) == 1 {
		subject = filepath.Base(items[0])
	}
	if action == "zip" {
		into = filepath.Join(destPath, dest)
	}
	a.t.Log(terminu.Info, map[string]string{"cp": "Copying ", "mv": "Moving ", "zip": "Zipping "}[action]+subject+" to "+shortPath(into))
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
		// The row's words without its glyph: the toast's title says how it
		// ended, and its words start with a capital (tdp toast.2).
		// One toast: a second one would take the first one's place in the App
		// Log too, and its animation, the first one's command dropped.
		words := strings.TrimSpace(spansText(taskText(*t)[1:]))
		kind := terminu.Success
		if t.status == taskError {
			kind = terminu.Error
		}
		return tea.Batch(a.t.Toast(kind, words), a.persist())
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
	var dir string // the temporary directory the zip goes in
	return a.t.Text(terminu.TextInput{
		// The name filu would choose, in grey: Enter takes it (tdp input.6).
		Title: "Zip", Glyph: glyphZip, Name: "The name", Verb: "zip",
		Prompt: zipPrompt(len(items)) + ": a name for the zip", Suggest: suggestZipName(items),
		Check: func(v string) error {
			if zipFileName(v) == "" {
				return fmt.Errorf("Not a name for a zip file")
			}
			return nil
		},
		// No temporary directory keeps the input open with the name in it (tdp F4).
		Commit: func(string) error {
			d, err := os.MkdirTemp("", "filu-zip-")
			if err != nil {
				return errors.New(opFailedText("make a temporary directory", err))
			}
			dir = d
			return nil
		},
		Done: func(v string) tea.Cmd {
			name := zipFileName(v)
			id := a.addTask("zip", name, dir, items)
			go runZip(id, items, filepath.Join(dir, name), a.taskCh)
			return a.persist()
		},
	})
}

// --- quitting ------------------------------------------------------------------

// syncQuit sets the quit flow (tdp K9: the app's). With one tab q leaves at
// once, a shell that goes where filu leaves it (FILU__LAST_DIR_FILE) going
// where that tab is (user 2026-10-24). With more, such a shell makes q a menu
// of where it goes; otherwise q confirms. A task still running is warned of
// first, one tab or not.
func (a *App) syncQuit() {
	a.t.Quit, a.t.QuitFlow, a.t.QuitDesc = nil, nil, "leave filu"
	shell := os.Getenv(envLastDirFile) != ""
	if shell && len(a.tabs) > 1 {
		a.t.QuitFlow, a.t.QuitDesc = a.quitMenu, "leave filu, your shell in the directory you choose"
		return
	}
	if shell {
		a.t.QuitDesc = "leave filu, your shell where you are"
	}
	ask := &terminu.Ask{Title: "Quit", Question: "Quit filu?", Verb: "quit"}
	if a.anyRunning() {
		ask.Details, ask.Warn = []string{"A task is still running: quitting stops it."}, []int{0}
	} else if len(a.tabs) == 1 {
		return
	}
	a.t.Quit = ask
}

// quitMenu chooses where the shell goes, then leaves: the directory filu
// started in, then each tab's. While a task is still running, the one chosen
// asks first.
func (a *App) quitMenu() tea.Cmd {
	var acts []terminu.Action
	seen := map[string]bool{}
	add := func(dir, from string) {
		if dir == "" || seen[dir] {
			return
		}
		seen[dir] = true
		where := shortPath(dir)
		acts = append(acts, terminu.Action{Key: strconv.Itoa(len(acts) + 1), Label: where, Desc: from, EnterHint: "quit",
			AskFor: func() *terminu.Ask {
				if !a.anyRunning() {
					return nil
				}
				return &terminu.Ask{Title: "Quit", Subject: where, Question: "Quit filu, your shell in " + where + "?", Verb: "quit",
					Details: []string{"A task is still running: quitting stops it."}, Warn: []int{0}}
			},
			Do: func() tea.Cmd {
				a.quitDir = dir
				return tea.Quit
			}})
	}
	add(a.launchDir, "where filu started")
	for i, t := range a.tabs {
		add(t.dir, "tab "+strconv.Itoa(i+1))
	}
	return a.t.Menu("Quit", terminu.MenuSection{Title: "your shell goes to", Actions: acts})
}
