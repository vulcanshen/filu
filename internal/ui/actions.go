package ui

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	terminu "github.com/vulcanshen/terminu-go"
)

// Title glyphs of filu's own popups (an input's glyph is the app's choice,
// tdp popup.1).
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
				return a.goTo(p, "")
			}},
		{Key: "Enter", Label: "Info", Desc: "the file's path, type, size, times, permissions and owner", Scope: item, Hint: "info",
			Hidden: a.isDirItem, Do: a.info},
		{Key: "o", Label: "Open", Desc: "it with its default app", Scope: item,
			AskFor: func() *terminu.Ask {
				name := a.itemName()
				return &terminu.Ask{Title: "Open", Subject: name, Question: "Open " + name + " with its default app?", Verb: "open"}
			},
			Do: a.open},
		{Key: "O", Label: "Open with", Desc: "an app you choose", Scope: item, Do: a.openWith},
		{Key: "m", Label: "Mark", Desc: "it to copy or move it later, or take the mark off", Scope: item, Hint: "mark", Do: a.toggleMark},
		{Key: "y", Label: "Yank", Desc: "its full path, to the clipboard", Scope: item, Do: a.yankItem},
		{Key: "r", Label: "Rename", Desc: "it, in place", Scope: item, Do: a.rename},
		{Key: "D", Label: "Delete", Desc: "it: into the trash", Scope: item,
			AskFor: func() *terminu.Ask {
				name := a.itemName()
				return &terminu.Ask{Title: "Delete", Subject: name, Question: "Move " + name + " to the trash?", Verb: "delete"}
			},
			Do: a.delete},
		{Key: "f", Label: "Favorite", Desc: "this directory, or take its star off", Scope: item,
			Hidden: func() bool { return !a.isDirItem() },
			Do: func() tea.Cmd {
				_, p, _ := a.item()
				return a.toggleFavorite(p)
			}},
		{Key: "c", Label: "Copy here", Desc: "the marked files, into this directory", Scope: panel, Hint: "copy",
			Disabled: a.nothingMarked, Do: func() tea.Cmd { return a.land(false) }},
		{Key: "v", Label: "Move here", Desc: "the marked files, into this directory", Scope: panel, Hint: "move",
			Disabled: a.nothingMarked, Do: func() tea.Cmd { return a.land(true) }},
		{Key: "/", Label: "Search", Desc: "file names or contents, under this directory", Scope: panel, Do: a.searchMenu},
		{Key: "go", Label: "Goto", Desc: "a favorite, or any directory under your home", Scope: panel, Do: a.gotoMenu},
		{Key: "F", Label: "Favorite here", Desc: "the directory you are in, or take its star off", Scope: panel,
			Do: func() tea.Cmd { return a.toggleFavorite(a.cur().dir) }},
		{Key: "b", Label: "Breadcrumb", Desc: "up to one of the directories this one is in", Scope: panel,
			Hidden: func() bool { d := a.cur().dir; return filepath.Dir(d) == d }, // at / there is none (tdp M6)
			Do:     a.breadcrumb},
		{Key: "t", Label: "New tab", Desc: "the same directory, a favorite, or a search", Scope: panel,
			Disabled: func() bool { return len(a.tabs) >= maxTabs }, Do: a.newTabMenu},
		{Key: "w", Label: "Close tab", Desc: "this tab", Scope: panel,
			Disabled: func() bool { return len(a.tabs) <= 1 }, Do: a.closeTab},
		{Key: "a", Label: "Add", Desc: "a file here; end its name with / for a directory", Scope: panel, Do: a.add},
		{Key: "S", Label: "Sort", Desc: "by a column, then a direction; this directory keeps it", Scope: panel, Do: a.sortMenu},
		// A shell clears the menu it is opened from (tdp T1).
		{Key: "s", Label: "Shell", Desc: "your shell, in this directory", Scope: panel, Clear: true,
			AskFor: func() *terminu.Ask {
				where := shortPath(a.cur().dir)
				return &terminu.Ask{Title: "Shell", Subject: where, Question: "Open your shell in " + where + "?", Verb: "open"}
			},
			Do: a.shell},
		{Key: ".", Label: "Hidden files", Desc: "show them, or hide them again", Scope: panel, Do: a.toggleHidden},
		a.zoomAction(),
	}
}

// zoomAction is z on any panel: it takes the whole screen, or gives it back
// ([1] leaving [3] in sight, layout). Zoom is a change of layout, not a mode:
// no key changes its meaning and Esc does not end it (tdp terms.8).
func (a *App) zoomAction() terminu.Action {
	return terminu.Action{Key: "z", Label: "Zoom", Desc: "this panel to the whole screen, or back", Scope: terminu.PanelScope, Hint: "zoom",
		Do: func() tea.Cmd {
			a.zoomed = !a.zoomed
			n := map[*terminu.Panel]string{a.files: "[1]", a.preview: "[2]", a.drawer: "[3]"}[a.screen.Focused()]
			if a.zoomed {
				a.t.Log(terminu.Info, "Zoomed in on "+n)
			} else {
				a.t.Log(terminu.Info, "Zoomed out of "+n)
			}
			return nil
		}}
}

func (a *App) nothingMarked() bool { return len(a.marks.landItems()) == 0 }

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
	// (tdp file-picker.4); the viewer opens as the first layer. A
	// table of facts has no line numbers (tdp note.2).
	label := terminu.Fg(terminu.LayerColor(1))
	for _, r := range rows {
		lines = append(lines, label.Render(fmt.Sprintf("%-*s", w, strings.ToUpper(r.label)))+"  "+r.value)
	}
	a.t.Log(terminu.Info, "Viewed the info of "+shortPath(p)) // only looking is an operation too (tdp note.7)
	return a.t.Viewer(&terminu.Viewer{Title: "Info", Subject: it.name, Glyph: glyphInfo, Lines: lines, NoNumbers: true})
}

// open hands the entry to its default app, once asked (openAsk): filu cannot
// follow it there (tdp F6, the app's choice).
func (a *App) open() tea.Cmd {
	_, p, ok := a.item()
	if !ok {
		return nil
	}
	a.t.Log(terminu.Info, "Opened "+shortPath(p)+" with its default app")
	return openFileCmd(p)
}

// openWith lists the default app and the ones open_with.yaml names; choosing one
// is the confirmation (tdp F6: a choice made in a picker).
func (a *App) openWith() tea.Cmd {
	_, p, ok := a.item()
	if !ok {
		return nil
	}
	acts := []terminu.Action{{Label: "Default", Desc: "the app the system opens it with", Do: func() tea.Cmd {
		a.t.Log(terminu.Info, "Opened "+shortPath(p)+" with its default app")
		return openFileCmd(p)
	}}}
	for _, app := range openWithApps {
		app := app
		acts = append(acts, terminu.Action{Label: app.Name, Desc: "runs " + app.Cmd, Do: func() tea.Cmd {
			a.t.Log(terminu.Info, "Opened "+shortPath(p)+" with "+app.Name+" ("+app.Cmd+")")
			return openWithCmd(app.Cmd, p)
		}})
	}
	return a.t.MenuAbout("Open with", a.itemName(), terminu.MenuSection{Actions: acts})
}

// toggleMark puts the entry in the marks, or takes it out.
func (a *App) toggleMark() tea.Cmd {
	_, p, ok := a.item()
	if !ok {
		return nil
	}
	a.marks.toggle(p)
	if indexOf(a.marks.items, p) >= 0 {
		a.t.Log(terminu.Info, "Marked "+shortPath(p))
	} else {
		a.t.Log(terminu.Info, "Took the mark off "+shortPath(p))
	}
	a.marksChanged()
	return a.persist()
}

// marksChanged redraws whatever shows the marks.
func (a *App) marksChanged() {
	for _, t := range a.tabs {
		t.list.SetMarked(a.marks.items)
	}
	a.refreshDrawer()
}

func (a *App) yankItem() tea.Cmd {
	_, p, ok := a.item()
	if !ok {
		return nil
	}
	return copyToClipboardCmd(p, "Copied "+p)
}

// rename renames the entry in place; the old name is offered in grey
// (tdp input.5).
func (a *App) rename() tea.Cmd {
	it, p, ok := a.item()
	if !ok {
		return nil
	}
	dir := a.cur().dir
	return a.t.Text(terminu.TextInput{
		Title: "Rename", Subject: it.name, Glyph: glyphRename, Name: "The name",
		Prompt: "A new name for " + it.name, Verb: "rename", Old: it.name,
		Check: func(v string) error {
			return checkName(v, func(name string) string { return renameProblem(dir, it.name, name) })
		},
		// A rename that fails keeps the input open with the name in it (tdp F4).
		Commit: func(v string) error {
			if err := os.Rename(p, filepath.Join(dir, v)); err != nil {
				return errors.New(opFailedText("rename "+it.name, err))
			}
			return nil
		},
		Done: func(v string) tea.Cmd {
			a.t.Log(terminu.Success, "Renamed "+shortPath(p)+" to "+v)
			a.reloadDir(dir)
			a.cur().list.Select(filepath.Join(dir, v))
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
		// Adding that fails keeps the input open with the name in it (tdp F4).
		Commit: func(v string) error {
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
				return errors.New(opFailedText("add "+v, err))
			}
			return nil
		},
		Done: func(v string) tea.Cmd {
			a.t.Log(terminu.Success, "Added "+shortPath(filepath.Join(dir, v)))
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
		return a.t.ErrorAbout("Delete failed", name, opFailedText("delete "+name, err))
	}
	a.marks.removeItem(p)
	a.reloadDir(dir)
	a.marksChanged()
	a.refreshPreview()
	return tea.Batch(a.t.Toast(terminu.Success, "Moved "+name+" to the trash"), a.persist())
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
	return tea.Batch(a.t.Toast(terminu.Success, what+shortPath(dir)), a.persist())
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
		acts = append(acts, terminu.Action{Label: shortPath(d), Desc: desc, Do: func() tea.Cmd { return a.goTo(d, "") }})
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
		{Key: "f", Label: "Favorite", Desc: "a tab on one of your favorites",
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
	a.t.Log(terminu.Info, "Opened a tab on "+shortPath(dir))
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
	a.t.Log(terminu.Info, "Closed the tab on "+shortPath(a.cur().dir))
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
	if t.showHidden {
		a.t.Log(terminu.Info, "Showed the hidden files in "+shortPath(t.dir))
	} else {
		a.t.Log(terminu.Info, "Hid the hidden files in "+shortPath(t.dir))
	}
	t.reload()
	a.fillTab(t)
	a.refreshPreview()
	return nil
}

// sortMenu is the sort, one popup a step (tdp F1): a column, then a
// direction. The directory keeps it; the directories stay first. Each column
// says where it is in the sort as the menu is drawn, so a direction chosen
// shows here at once.
func (a *App) sortMenu() tea.Cmd {
	dir := a.cur().dir
	var cols []terminu.Action
	for _, d := range sortCols {
		d := d
		cols = append(cols, terminu.Action{Key: d.key, Label: d.title, Desc: sortDesc(dir, d.col),
			Live: func() (string, string) { return d.title, sortDesc(dir, d.col) },
			Do:   func() tea.Cmd { return a.sortDirMenu(dir, d) }})
	}
	reset := terminu.Action{Key: "r", Label: "Reset", Desc: "back to names, A to Z", Disabled: func() bool { return len(sortRulesFor(dir)) == 0 },
		Do: func() tea.Cmd {
			resetSortFor(dir)
			a.t.Log(terminu.Info, "Reset the sort of "+shortPath(dir))
			return a.sorted(dir)
		}}
	return a.t.Menu("Sort", terminu.MenuSection{Title: "column", Actions: cols}, terminu.MenuSection{Title: "the whole sort", Actions: []terminu.Action{reset}})
}

// sortDesc is where a column is in a directory's sort.
func sortDesc(dir string, c sortCol) string {
	rules := sortRulesFor(dir)
	if i := sortRuleIndex(rules, c); i >= 0 {
		return ordinal(i+1) + ", " + direction(rules[i].asc)
	}
	return "not in the sort"
}

// sortDirMenu is the second step: a value written back to the column menu,
// which stays to sort by another column (tdp T1).
func (a *App) sortDirMenu(dir string, d sortColDef) tea.Cmd {
	in := sortRuleIndex(sortRulesFor(dir), d.col) >= 0
	// Each pick is the sort's result, so it goes in the App Log (tdp note.7).
	pick := func(done string, set func()) func() tea.Cmd {
		return func() tea.Cmd {
			set()
			a.t.Log(terminu.Info, done)
			return tea.Batch(a.t.CloseTop(1), a.sorted(dir))
		}
	}
	by := "Sorted " + shortPath(dir) + " by " + d.title
	acts := []terminu.Action{
		{Key: "a", Label: "Ascending", Desc: "smallest, oldest or A first", Stay: true,
			Do: pick(by+", ascending", func() { setSortFor(dir, d.col, true) })},
		{Key: "D", Label: "Descending", Desc: "largest, newest or Z first", Stay: true,
			Do: pick(by+", descending", func() { setSortFor(dir, d.col, false) })},
		{Key: "x", Label: "Remove", Desc: "this column from the sort", Stay: true, Hidden: func() bool { return !in },
			Do: pick("Took "+d.title+" out of the sort of "+shortPath(dir), func() { unsetSortFor(dir, d.col) })},
	}
	return a.t.Menu("Sort by "+d.title, terminu.MenuSection{Actions: acts})
}

// sorted shows a directory's new sort in every tab on it.
func (a *App) sorted(dir string) tea.Cmd {
	for _, t := range a.tabs {
		if t.dir == dir {
			a.fillTab(t)
		}
	}
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
// ends, the directory is read again. The user types commands of their own in
// it: an interactive terminal, with History on PgUp and the selection on Alt-v
// (tdp terminal), copying as filu copies.
func (a *App) shell() tea.Cmd {
	dir := a.cur().dir
	a.t.Log(terminu.Info, "Opened your shell in "+shortPath(dir))
	a.term = &terminu.Terminal{Title: "Shell", Subject: shortPath(dir), Dir: dir, Ends: true, Interactive: true,
		Copy: func(text string) tea.Cmd { return copyToClipboardCmd(text, "Copied the selection") },
		OnExit: func(error) tea.Cmd {
			a.t.Log(terminu.Info, "Your shell in "+shortPath(dir)+" ended")
			a.reloadDir(dir)
			a.refreshPreview()
			return nil
		}}
	return a.t.OpenTerminal(a.term)
}
