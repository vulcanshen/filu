package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	terminu "github.com/vulcanshen/terminu-go"
)

// Title glyphs of filu's own popups (an input's glyph is the app's choice,
// components/layout/popup).
const (
	glyphRename = "\U000F03EB" // nf-md-pencil
	glyphAdd    = "\U000F0752" // nf-md-file_plus
	glyphZip    = "\U000F05C4" // nf-md-zip_box
	glyphSearch = "\U000F0349" // nf-md-magnify
	glyphGoto   = "\U000F024B" // nf-md-folder
	glyphView   = "\U000F0208" // nf-md-eye
	glyphInfo   = "\U000F02FD" // nf-md-information_outline
)

func itoa(n int) string { return strconv.Itoa(n) }

// filesActions are [1]'s: what to do with the entry under the cursor, then
// with the directory and its tabs.
func (a *App) filesActions() []terminu.Action {
	item, panel := terminu.ItemScope, terminu.PanelScope
	return []terminu.Action{
		{Key: "Enter", Label: "Into", Desc: "this directory", Scope: item, Hint: "into",
			Hidden: func() bool { return !a.isDirItem() },
			Do: func() tea.Cmd {
				_, p, _ := a.item()
				return a.cd(p, "")
			}},
		{Key: "Enter", Label: "Info", Desc: "the file's path, type, size, times, permissions and owner", Scope: item, Hint: "info",
			Hidden: a.isDirItem, Do: a.info},
		{Key: "o", Label: "open", Desc: "it with its default app", Scope: item, Ask: a.openAsk,
			Disabled: asking(a.openAsk, func(k *terminu.Ask) {
				name := a.itemName()
				k.Subject, k.Question = name, "Open "+name+" with its default app?"
			}, nil),
			Do: a.open},
		{Key: "O", Label: "Open with", Desc: "an app you choose", Scope: item, Do: a.openWith},
		{Key: "m", Label: "mark", Desc: "it to copy or move it later, or take the mark off", Scope: item, Hint: "mark", Do: a.toggleMark},
		{Key: "y", Label: "yank", Desc: "its full path, to the clipboard", Scope: item, Do: a.yankItem},
		{Key: "r", Label: "rename", Desc: "it, in place", Scope: item, Do: a.rename},
		{Key: "D", Label: "Delete", Desc: "it: into the trash", Scope: item, Ask: a.deleteAsk,
			Disabled: asking(a.deleteAsk, func(k *terminu.Ask) {
				name := a.itemName()
				k.Subject, k.Question = name, "Move "+name+" to the trash?"
			}, nil),
			Do: a.delete},
		{Key: "f", Label: "favorite", Desc: "this directory, or take its star off", Scope: item,
			Hidden: func() bool { return !a.isDirItem() },
			Do: func() tea.Cmd {
				_, p, _ := a.item()
				return a.toggleFavorite(p)
			}},
		{Key: "c", Label: "copy here", Desc: "the marked files, into this directory", Scope: panel, Hint: "copy",
			Disabled: a.nothingMarked, Do: func() tea.Cmd { return a.land(false) }},
		{Key: "v", Label: "move here", Desc: "the marked files, into this directory", Scope: panel, Hint: "move",
			Disabled: a.nothingMarked, Do: func() tea.Cmd { return a.land(true) }},
		{Key: "/", Label: "Search", Desc: "file names or contents, under this directory", Scope: panel, Do: a.searchMenu},
		{Key: "go", Label: "goto", Desc: "a favorite, or any directory under your home", Scope: panel, Do: a.gotoMenu},
		{Key: "F", Label: "Favorite here", Desc: "the directory you are in, or take its star off", Scope: panel,
			Do: func() tea.Cmd { return a.toggleFavorite(a.cur().dir) }},
		{Key: "b", Label: "breadcrumb", Desc: "up to one of the directories this one is in", Scope: panel,
			Hidden: func() bool { d := a.cur().dir; return filepath.Dir(d) == d }, // at / there is none (tdp M6)
			Do:     a.breadcrumb},
		{Key: "t", Label: "new tab", Desc: "the same directory, a favorite, or a search", Scope: panel,
			Disabled: func() bool { return len(a.tabs) >= maxTabs }, Do: a.newTabMenu},
		{Key: "w", Label: "Close tab", Desc: "this tab", Scope: panel,
			Disabled: func() bool { return len(a.tabs) <= 1 }, Do: a.closeTab},
		{Key: "a", Label: "add", Desc: "a file here; end its name with / for a directory", Scope: panel, Do: a.add},
		{Key: "S", Label: "Sort", Desc: "by a column, then a direction; this directory keeps it", Scope: panel, Do: a.sortMenu},
		{Key: "s", Label: "shell", Desc: "your shell, in this directory", Scope: panel, Ask: a.shellAsk,
			Disabled: asking(a.shellAsk, func(k *terminu.Ask) {
				where := safeName(shortPath(a.cur().dir))
				k.Subject, k.Question = where, "Open your shell in "+where+"?"
			}, nil),
			Do: a.shell},
		{Key: ".", Label: "Hidden files", Desc: "show them, or hide them again", Scope: panel, Do: a.toggleHidden},
		a.zoomAction(),
	}
}

// zoomAction is z on any panel: it takes the whole screen, or gives it back.
// Zoom is a change of layout, not a mode: no key changes its meaning and Esc
// does not end it (the tdp term "mode").
func (a *App) zoomAction() terminu.Action {
	return terminu.Action{Key: "z", Label: "zoom", Desc: "this panel to the whole screen, or back", Scope: terminu.PanelScope, Hint: "zoom",
		Do: func() tea.Cmd {
			a.zoomed = !a.zoomed
			a.t.ForceNarrow = a.zoomed
			return nil
		}}
}

// globalActions are what belong to no panel. Quitting is terminu's own row.
func (a *App) globalActions() []terminu.Action {
	return []terminu.Action{
		// TODO(lib FL1): the quit flow cannot be a menu; choosing where the
		// shell goes is this row instead.
		{Key: "Q", Label: "Quit to", Desc: "leave filu, your shell in the directory you choose", Scope: terminu.GlobalScope, Do: a.quitToMenu},
	}
}

func (a *App) nothingMarked() bool { return len(a.marks.landItems()) == 0 }

// asking fills in an action's Ask from where filu is now — the question names
// the item under the cursor — and reports whether the action cannot run.
// terminu reads an action's Ask when it runs it, after it asks Disabled, so the
// question is always the one for now; and an action with an Ask opens its
// confirm over the menu it ran from (tdp F4, T1).
// TODO(lib FL15): an Ask that depends on the item.
func asking(ask *terminu.Ask, fill func(*terminu.Ask), disabled func() bool) func() bool {
	return func() bool {
		if disabled != nil && disabled() {
			return true
		}
		fill(ask)
		return false
	}
}

// info shows the file's facts (tdp K3: Enter on a file).
func (a *App) info() tea.Cmd {
	it, p, ok := a.item()
	if !ok {
		return nil
	}
	var lines []string
	rows := fileFacts(p)
	w := 0
	for _, r := range rows {
		w = max(w, len(r.label))
	}
	// The labels in the layer colour, as a file-picker's facts are
	// (components/input/file-picker); the viewer opens as the first layer.
	label := terminu.Fg(terminu.LayerColor(1))
	for _, r := range rows {
		lines = append(lines, label.Render(fmt.Sprintf("%-*s", w, strings.ToUpper(r.label)))+"  "+r.value)
	}
	// TODO(lib FL12): without line numbers, once a viewer can leave them out.
	return a.t.Viewer(&terminu.Viewer{Title: "Info", Subject: safeName(it.name), Glyph: glyphInfo, Lines: lines})
}

// open hands the entry to its default app, once asked (openAsk): filu cannot
// follow it there (tdp F6, the app's choice).
func (a *App) open() tea.Cmd {
	_, p, ok := a.item()
	if !ok {
		return nil
	}
	return openFileCmd(p)
}

// openWith lists the default app and the ones config.yaml names; choosing one
// is the confirmation (tdp F6: a choice made in a picker).
func (a *App) openWith() tea.Cmd {
	_, p, ok := a.item()
	if !ok {
		return nil
	}
	acts := []terminu.Action{{Label: "Default", Desc: "the app the system opens it with", Do: func() tea.Cmd { return openFileCmd(p) }}}
	for _, app := range openWithApps {
		app := app
		acts = append(acts, terminu.Action{Label: app.Name, Desc: "runs " + app.Cmd, Do: func() tea.Cmd { return openWithCmd(app.Cmd, p) }})
	}
	return a.t.Menu("Open with", // TODO(lib FL14): a menu title cannot keep a name as it is
		terminu.MenuSection{Actions: acts})
}

// toggleMark puts the entry in the marks, or takes it out.
func (a *App) toggleMark() tea.Cmd {
	_, p, ok := a.item()
	if !ok {
		return nil
	}
	a.marks.toggle(p)
	a.marksChanged()
	return a.persist()
}

// marksChanged redraws whatever shows the marks.
func (a *App) marksChanged() {
	for _, t := range a.tabs {
		syncMarks(t.list, a.marks.inBucket())
	}
	a.refreshDrawer()
}

func (a *App) yankItem() tea.Cmd {
	_, p, ok := a.item()
	if !ok {
		return nil
	}
	return copyToClipboardCmd(p, "Copied "+safeName(p))
}

// rename renames the entry in place; the old name is offered in grey
// (components/input/README).
func (a *App) rename() tea.Cmd {
	it, p, ok := a.item()
	if !ok {
		return nil
	}
	dir := a.cur().dir
	return a.t.Text(terminu.TextInput{
		Title: "Rename", Subject: safeName(it.name), Glyph: glyphRename, Name: "The name",
		Prompt: "A new name for " + safeName(it.name), Verb: "rename", Old: it.name,
		Check: func(v string) error {
			return checkName(v, func(name string) string { return renameProblem(dir, it.name, name) })
		},
		Done: func(v string) tea.Cmd {
			to := filepath.Join(dir, v)
			if err := os.Rename(p, to); err != nil {
				return a.t.Error("Rename failed", opFailedText("rename "+safeName(it.name), err))
			}
			a.reloadDir(dir)
			a.cur().list.Select(to)
			a.refreshPreview()
			return nil
		},
	})
}

// checkName turns a problem with a name into the input's error.
func checkName(v string, problem func(string) string) error {
	if v == "" {
		return fmt.Errorf("Type a name first")
	}
	if p := problem(v); p != "" {
		return fmt.Errorf("%s", p)
	}
	return nil
}

func renameProblem(dir, old, name string) string {
	switch {
	case strings.Contains(name, "/"):
		return "A name can't contain /"
	case name == old:
		return ""
	}
	if _, err := os.Lstat(filepath.Join(dir, name)); err == nil {
		return name + " already exists here"
	}
	return ""
}

// add makes a file, or a directory when the name ends with /, in the
// directory; the name may have directories in it, not leave the directory.
func (a *App) add() tea.Cmd {
	dir := a.cur().dir
	return a.t.Text(terminu.TextInput{
		Title: "Add", Glyph: glyphAdd, Name: "The name",
		Prompt: "A name for a new file; end it with / for a directory", Verb: "add",
		Check: func(v string) error { return checkName(v, func(name string) string { return addProblem(dir, name) }) },
		Done: func(v string) tea.Cmd {
			target := filepath.Join(dir, strings.TrimSuffix(v, "/"))
			var err error
			if strings.HasSuffix(v, "/") {
				err = os.MkdirAll(target, 0o755)
			} else if err = os.MkdirAll(filepath.Dir(target), 0o755); err == nil {
				var f *os.File
				if f, err = os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644); err == nil {
					f.Close()
				}
			}
			if err != nil {
				return a.t.Error("Add failed", opFailedText("add "+safeName(v), err))
			}
			a.reloadDir(dir)
			a.cur().list.Select(filepath.Join(dir, firstElem(strings.TrimSuffix(v, "/"))))
			a.refreshPreview()
			return nil
		},
	})
}

// firstElem is the first element of a relative path: what shows in the list.
func firstElem(p string) string {
	if i := strings.Index(p, "/"); i >= 0 {
		return p[:i]
	}
	return p
}

func addProblem(dir, name string) string {
	base := strings.TrimSuffix(name, "/")
	if base == "" || base == "." || base == ".." {
		return "Not a name for a file or directory"
	}
	// Under this directory only: filepath.Join would take a ../ out of it and
	// put a leading / back under it.
	if c := filepath.Clean(base); filepath.IsAbs(c) || c == ".." || strings.HasPrefix(c, "../") {
		return base + " is outside this directory"
	}
	if _, err := os.Lstat(filepath.Join(dir, base)); err == nil {
		return base + " already exists here"
	}
	return ""
}

// delete moves the entry to the trash, once asked (deleteAsk).
func (a *App) delete() tea.Cmd {
	_, p, ok := a.item()
	if !ok {
		return nil
	}
	name, dir := a.itemName(), a.cur().dir
	if err := moveToTrash(p); err != nil {
		return a.t.Error("Delete failed", opFailedText("delete "+name, err))
	}
	a.marks.removeItem(p)
	a.reloadDir(dir)
	a.marksChanged()
	a.refreshPreview()
	return tea.Batch(a.t.Toast(terminu.Info, "Moved "+name+" to the trash"), a.persist())
}

// toggleFavorite stars dir, or takes its star off.
func (a *App) toggleFavorite(dir string) tea.Cmd {
	starred := a.favs.pinnedSet()[dir]
	a.favs.togglePin(dir)
	for _, t := range a.tabs {
		a.fillTab(t)
	}
	a.refreshDrawer()
	what := "Starred "
	if starred {
		what = "Took the star off "
	}
	return tea.Batch(a.t.Toast(terminu.Info, what+safeName(shortPath(dir))), a.persist())
}

// land copies or moves the marked files (the picks, when there are some) into
// the directory, in the background (the Tasks tab).
func (a *App) land(move bool) tea.Cmd {
	return a.startLand(a.marks.landItems(), a.cur().dir, move)
}

// breadcrumb lists the directories this one is in, nearest first.
func (a *App) breadcrumb() tea.Cmd {
	var acts []terminu.Action
	dir := a.cur().dir
	for d, n := filepath.Dir(dir), 1; ; d, n = filepath.Dir(d), n+1 {
		d := d
		desc := "one level up"
		if n > 1 {
			desc = itoa(n) + " levels up"
		}
		acts = append(acts, terminu.Action{Label: safeName(shortPath(d)), Desc: desc, Do: func() tea.Cmd { return a.cd(d, "") }})
		if filepath.Dir(d) == d {
			break
		}
	}
	return a.t.Menu("Breadcrumb", terminu.MenuSection{Actions: acts})
}

// newTabMenu opens a tab: on the same directory, a favorite, or what a search
// finds.
func (a *App) newTabMenu() tea.Cmd {
	return a.t.Menu("New tab", terminu.MenuSection{Actions: []terminu.Action{
		{Key: "s", Label: "Same directory", Desc: "a second tab on this one", Do: func() tea.Cmd { return a.openTab(a.cur().dir, "") }},
		{Key: "f", Label: "Favorite", Desc: "a tab on one of your favorites", Stay: true,
			Disabled: func() bool { return len(a.favs.pinned) == 0 },
			Do:       func() tea.Cmd { return a.favoriteMenu(true) }},
		{Key: "/", Label: "Search", Desc: "a tab on any directory under your home", Do: func() tea.Cmd { return a.gotoFinder(true) }},
	}})
}

// openTab adds a tab on dir after the others and shows it.
func (a *App) openTab(dir, focus string) tea.Cmd {
	if len(a.tabs) >= maxTabs {
		return a.t.Toast(terminu.Warning, fmt.Sprintf("All %d tabs are in use: close one with [w] first", maxTabs))
	}
	t := newTab(dir)
	a.fillTab(t)
	a.tabs = append(a.tabs, t)
	a.active = len(a.tabs) - 1
	a.syncTabs()
	if focus != "" {
		t.list.Select(focus)
	}
	a.syncWatches()
	a.refreshPreview()
	a.syncQuit()
	return a.persist()
}

// closeTab closes the tab [1] shows; the one before it takes its place.
func (a *App) closeTab() tea.Cmd {
	if len(a.tabs) <= 1 {
		return nil
	}
	a.tabs = append(a.tabs[:a.active], a.tabs[a.active+1:]...)
	a.active = max(a.active-1, 0)
	a.syncTabs()
	a.syncWatches()
	a.refreshPreview()
	a.syncQuit()
	return a.persist()
}

// toggleHidden shows the tab's dotfiles, or hides them again.
func (a *App) toggleHidden() tea.Cmd {
	t := a.cur()
	t.showHidden = !t.showHidden
	t.reload()
	a.fillTab(t)
	a.refreshPreview()
	return nil
}

// sortMenu is the sort, one popup a step (tdp F1): a column, then a
// direction. The directory keeps it; the directories stay first.
func (a *App) sortMenu() tea.Cmd {
	dir := a.cur().dir
	rules := sortRulesFor(dir)
	var cols []terminu.Action
	for _, d := range sortCols {
		d := d
		desc := "not in the sort"
		if i := sortRuleIndex(rules, d.col); i >= 0 {
			desc = ordinal(i+1) + ", " + direction(rules[i].asc)
		}
		cols = append(cols, terminu.Action{Key: d.key, Label: d.title, Desc: desc, Stay: true, Do: func() tea.Cmd { return a.sortDirMenu(dir, d) }})
	}
	reset := terminu.Action{Key: "r", Label: "Reset", Desc: "back to names, A to Z", Disabled: func() bool { return len(sortRulesFor(dir)) == 0 },
		Do: func() tea.Cmd {
			resetSortFor(dir)
			return a.sorted(dir)
		}}
	return a.t.Menu("Sort", terminu.MenuSection{Title: "column", Actions: cols}, terminu.MenuSection{Title: "the whole sort", Actions: []terminu.Action{reset}})
}

// TODO(lib FL13): choosing a direction should close this step only and leave
// the column menu showing the new sort (tdp T1, S4).
func (a *App) sortDirMenu(dir string, d sortColDef) tea.Cmd {
	in := sortRuleIndex(sortRulesFor(dir), d.col) >= 0
	acts := []terminu.Action{
		{Key: "a", Label: "Ascending", Desc: "smallest, oldest or A first", Do: func() tea.Cmd { setSortFor(dir, d.col, true); return a.sorted(dir) }},
		{Key: "D", Label: "Descending", Desc: "largest, newest or Z first", Do: func() tea.Cmd { setSortFor(dir, d.col, false); return a.sorted(dir) }},
		{Key: "x", Label: "Remove", Desc: "this column from the sort", Hidden: func() bool { return !in },
			Do: func() tea.Cmd { unsetSortFor(dir, d.col); return a.sorted(dir) }},
	}
	return a.t.Menu("Sort by "+d.title, terminu.MenuSection{Actions: acts})
}

// sorted shows a directory's new sort in every tab on it.
func (a *App) sorted(dir string) tea.Cmd {
	a.reloadDir(dir)
	a.refreshPreview()
	return a.persist()
}

func ordinal(n int) string {
	switch n {
	case 1:
		return "1st"
	case 2:
		return "2nd"
	case 3:
		return "3rd"
	}
	return itoa(n) + "th"
}

func direction(asc bool) string {
	if asc {
		return "ascending"
	}
	return "descending"
}

// shell opens the user's shell in the directory, once asked (shellAsk); when it
// ends, the directory is read again.
func (a *App) shell() tea.Cmd {
	dir := a.cur().dir
	a.term = &terminu.Terminal{Title: "Shell", Subject: safeName(shortPath(dir)), Dir: dir, Ends: true,
		OnExit: func(error) tea.Cmd {
			a.reloadDir(dir)
			a.refreshPreview()
			return nil
		}}
	return a.t.OpenTerminal(a.term)
}
