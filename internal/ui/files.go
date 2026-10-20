package ui

import (
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	terminu "github.com/vulcanshen/terminu-go"
)

// tab is one of [1]'s directory tabs: a directory, its entries and the list
// that shows them.
type tab struct {
	dir        string
	items      []fileItem
	byPath     map[string]fileItem
	hidden     int   // dotfiles in the directory, shown or not
	err        error // why the directory could not be read
	showHidden bool
	list       *terminu.List
	fav        bool   // the directory is a favorite, for the location row's star
	logged     string // the read error last put in the App Log, so it goes in once
}

func newTab(dir string) *tab {
	t := &tab{dir: dir}
	t.list = &terminu.List{Marks: true}
	t.reload()
	return t
}

// reload reads the directory again; the list keeps its cursor on the same
// entry when it is still there (fillTab sets the rows).
func (t *tab) reload() {
	t.items, t.hidden, t.err = readEntries(t.dir, t.showHidden)
	t.byPath = make(map[string]fileItem, len(t.items))
	for _, it := range t.items {
		t.byPath[filepath.Join(t.dir, it.name)] = it
	}
}

// focus puts the cursor on the entry called name, reporting whether it is
// listed. The rows must be set first: it fills them when they are not.
func (t *tab) focus(name string) bool {
	p := filepath.Join(t.dir, name)
	if _, ok := t.byPath[p]; !ok {
		return false
	}
	if len(t.list.Rows()) != len(t.items) {
		t.list.SetRows(fileRows(t, nil))
	}
	t.list.Select(p)
	return true
}

// current is the entry under the cursor.
func (t *tab) current() (fileItem, string, bool) {
	r, ok := t.list.Current()
	if !ok {
		return fileItem{}, "", false
	}
	it, ok := t.byPath[r.ID]
	return it, r.ID, ok
}

// cursorIndex is where the cursor is among the entries, for the session file.
func (t *tab) cursorIndex() int {
	r, ok := t.list.Current()
	if !ok {
		return 0
	}
	return max(slices.IndexFunc(t.list.Rows(), func(x terminu.Row) bool { return x.ID == r.ID }), 0)
}

// selectIndex puts the cursor on the n-th entry.
func (t *tab) selectIndex(n int) {
	rows := t.list.Rows()
	if n >= 0 && n < len(rows) {
		t.list.Select(rows[n].ID)
	}
}

// fileCol is a column of the file list.
type fileCol int

const (
	colModified fileCol = iota
	colOwner
	colPerms
	colSize
	colName
)

// allFileCols are the columns in the order they are drawn; the name is last.
var allFileCols = []fileCol{colModified, colOwner, colPerms, colSize, colName}

// fileColumns are the list's columns, as allFileCols draws them. The name takes
// what the others leave, cut short down to 12 cells; past that the others go
// whole — the owner, then the size, the modified time and the permissions —
// and the name stays (components/layout/panel).
var fileColumns = []terminu.Column{
	{Title: "Modified", Drop: 3},
	{Title: "Owner", Drop: 1},
	{Title: "Perms", Drop: 4},
	{Title: "Size", Number: true, Drop: 2},
	{Title: "Name", Shrink: true, MinWidth: 12},
}

// colOfSort is the column a sort column sorts.
var colOfSort = map[sortCol]int{sortMtime: 0, sortOwner: 1, sortPerm: 2, sortSize: 3, sortName: 4}

// sortKeys are a directory's sort chain as the list sorts it.
func sortKeys(rules []sortRule) []terminu.SortKey {
	keys := make([]terminu.SortKey, len(rules))
	for i, r := range rules {
		keys[i] = terminu.SortKey{Col: colOfSort[r.col], Desc: !r.asc}
	}
	return keys
}

// fileRows are the list's rows for the tab's entries, the directories a group
// before the files, so that a sort keeps them first.
func fileRows(t *tab, favs map[string]bool) []terminu.Row {
	rows := make([]terminu.Row, len(t.items))
	for i, it := range t.items {
		path := filepath.Join(t.dir, it.name)
		cells := make([]string, len(allFileCols))
		styled := make([][]terminu.Span, len(allFileCols))
		for j, c := range allFileCols {
			styled[j] = fileCell(it, c, favs[path])
			cells[j] = spansText(styled[j])
		}
		group := 1
		if it.isDir {
			group = 0
		}
		rows[i] = terminu.Row{ID: path, Cells: cells, Styled: styled, SortGroup: group,
			Sort: []string{it.mtime.UTC().Format("2006-01-02T15:04:05.000000000"), it.owner, it.perm, strconv.FormatInt(it.size, 10), it.name}}
	}
	return rows
}

// fileCell is one entry's value in a column, in its colours: the type's for
// the name (as eza and ls colour it), eza's for the owner, the permissions and
// the size. The name is as it is: terminu draws a control character in it by
// its name, in Red (components/README, control characters).
func fileCell(it fileItem, c fileCol, fav bool) []terminu.Span {
	switch c {
	case colModified:
		return []terminu.Span{{Text: fmtMtime(it.mtime), Style: terminu.Fg(terminu.Overlay0)}}
	case colOwner:
		return colorOwner(terminu.Truncate(it.owner, colOwnerW))
	case colPerms:
		return colorPerm(clipMode(it.perm))
	case colSize:
		return colorSize(it)
	}
	name := []terminu.Span{{Text: fileIcon(it) + " " + it.name, Style: terminu.Fg(terminu.Hex(string(fileColor(it))))}}
	if fav {
		name = append(name, terminu.Span{Text: " "}, terminu.Span{Text: iconPin, Style: terminu.Fg(terminu.Yellow)}) // a favorite directory: content, the app's (components/layout/panel)
	}
	return name
}

// fillTab sets a tab's rows from its entries: the directory's sort, the marks
// from the bucket, the stars from the favorites.
func (a *App) fillTab(t *tab) {
	// What could not be read is the panel's error; its details go in the App
	// Log, once (tdp E3, F5).
	if t.err == nil {
		t.logged = ""
	} else if why := t.dir + ": " + t.err.Error(); why != t.logged {
		t.logged = why
		a.t.Log(terminu.Error, "Cannot read the directory "+why)
	}
	favs := a.favs.pinnedSet()
	t.fav = favs[t.dir]
	t.list.Columns = fileColumns
	t.list.Location = location(t)
	t.list.OnMove = func(terminu.Row) tea.Cmd { a.refreshPreview(); return nil }
	t.list.SetRows(fileRows(t, favs))
	t.list.SortBy(sortKeys(sortRulesFor(t.dir))...)
	t.list.EmptyFact, t.list.EmptyHint = emptyDir(t)
	t.list.SetMarked(a.marks.items)
}

// location is a tab's location row: where it is, with a star after it when
// the directory is a favorite (content, the app's; components/layout/panel).
// TODO(tdp FL25): the star in Yellow, as after a starred name in the list,
// once a location row can take spans.
func location(t *tab) string {
	loc := shortPath(t.dir)
	if t.fav {
		loc += " " + iconPin
	}
	return loc
}

// emptyDir is what an empty directory says (components/layout/screen).
func emptyDir(t *tab) (fact, hint string) {
	if t.hidden > 0 && !t.showHidden {
		n := strconv.Itoa(t.hidden) + " hidden"
		return "Nothing here but " + n, "Press [.] to show hidden files, or [Space] to see what you can do here"
	}
	return "Nothing here", "Press [a] to add a file, or [Space] to see what you can do here"
}

// filesBody is [1]'s inside: the current tab's list. Zoomed with several tabs
// it shows every tab side by side, the one the keys go to lit. It embeds the
// current tab's list, so moving its cursor updates the preview the way a
// list's OnMove does.
type filesBody struct {
	*terminu.List
	app *App
}

func (b *filesBody) Draw(w, h int, focused bool) []string {
	a := b.app
	if a.zoomed && len(a.tabs) > 1 {
		return b.drawColumns(w, h, focused)
	}
	return a.drawTab(a.cur(), w, h, focused)
}

// Empty: [1]'s list draws its own empty state under the location row, so the
// panel does not put one in its middle.
func (b *filesBody) Empty() bool { return false }

// drawTab is one tab in w × h: its list, or under its location row why the
// directory could not be read.
func (a *App) drawTab(t *tab, w, h int, focused bool) []string {
	if t.err == nil {
		return t.list.Draw(w, h, focused)
	}
	out := t.list.Draw(w, 1, focused) // the location row: an unread directory lists nothing
	if h > 1 {
		out = append(out, failRows(w, h-1, "Cannot read this directory: "+friendlyErr(t.err), "Press [Esc] to go up")...)
	}
	return terminu.Pad2(out, w, h)
}

// failRows are a panel's error, centred: the fact in Red, the hint in Overlay0
// (components/layout/panel, loading and errors).
func failRows(w, h int, fact, hint string) []string {
	out := terminu.EmptyState(w, h, "", hint)
	top := max((h-3)/2, 0)
	if top < len(out) {
		out[top] = terminu.Center(terminu.Fg(terminu.Red).Render(terminu.Truncate(fact, w)), w)
	}
	return out
}

// drawColumns is every tab side by side, a quiet line between (Overlay0, as
// the menu's separators); the tab the keys go to is lit and the others dimmed
// (tdp P6). No component lays tabs out like this: a deviation, written in
// dev-remarks.
func (b *filesBody) drawColumns(w, h int, focused bool) []string {
	a := b.app
	n := len(a.tabs)
	cw := (w - (n - 1)) / n
	out := terminu.Pad2(nil, w, h)
	x := 0
	for i, t := range a.tabs {
		width := cw
		if i == n-1 {
			width = w - x
		}
		col := a.drawTab(t, width, h, focused && i == a.active)
		if i != a.active {
			col = terminu.DimLines(col)
		}
		terminu.Composite(out, col, x, 0, w)
		x += width
		if i < n-1 {
			sep := make([]string, h)
			for r := range sep {
				sep[r] = terminu.Fg(terminu.Overlay0).Render("│")
			}
			terminu.Composite(out, sep, x, 0, w)
			x++
		}
	}
	return out
}

// switchTab is h|l on [1]: another tab.
func (a *App) switchTab(i int) tea.Cmd {
	if i < 0 || i >= len(a.tabs) {
		return nil
	}
	a.active = i
	a.fb.List = a.cur().list
	a.refreshPreview()
	a.syncQuit()
	return nil
}

// syncTabs gives [1] its tabs: one mark each, the first one's being the
// label (components/layout/panel, tabs).
func (a *App) syncTabs() {
	a.active = min(a.active, len(a.tabs)-1)
	a.files.Label = tabMark(0)
	a.files.Tabs = nil
	for i := 1; i < len(a.tabs); i++ {
		a.files.Tabs = append(a.files.Tabs, tabMark(i))
	}
	a.files.Tab = a.active
	a.fb.List = a.cur().list
}

// cd points the current tab at dir, the cursor on the entry at focus (a path)
// or on the first one.
func (a *App) cd(dir, focus string) tea.Cmd {
	t := a.cur()
	t.dir = dir
	t.reload()
	a.fillTab(t)
	if focus != "" {
		t.list.Select(focus)
	} else if rows := t.list.Rows(); len(rows) > 0 {
		t.list.Select(rows[0].ID)
	}
	a.syncWatches()
	a.refreshPreview()
	a.syncQuit()
	return nil
}

// up is Esc on [1]: the parent directory, the cursor on the one it came from
// (tdp K4, one level up).
func (a *App) up() tea.Cmd {
	t := a.cur()
	parent := filepath.Dir(t.dir)
	if parent == t.dir {
		return nil
	}
	return a.cd(parent, t.dir)
}

// reveal points the current tab at path: into it when it is a directory, else
// into its parent with the cursor on it.
func (a *App) reveal(path string) tea.Cmd {
	info, err := os.Stat(path)
	if err != nil {
		return a.t.Toast(terminu.Error, opFailedText("open "+filepath.Base(path), err))
	}
	if info.IsDir() {
		return a.cd(path, "")
	}
	return a.cd(filepath.Dir(path), path)
}

// reloadDir reads every tab on dir again.
func (a *App) reloadDir(dir string) {
	for _, t := range a.tabs {
		if t.dir == dir {
			t.reload()
			a.fillTab(t)
		}
	}
}

// reloadAll reads every tab again (a sort or the marks changed).
func (a *App) reloadAll() {
	for _, t := range a.tabs {
		t.reload()
		a.fillTab(t)
	}
	a.refreshPreview()
}

// item is the entry under [1]'s cursor and its path.
func (a *App) item() (fileItem, string, bool) { return a.cur().current() }

// isDirItem reports a directory under [1]'s cursor.
func (a *App) isDirItem() bool {
	it, _, ok := a.item()
	return ok && it.isDir
}

// itemName is the entry under [1]'s cursor as it is shown.
func (a *App) itemName() string {
	it, _, _ := a.item()
	return it.name
}

// noTrailingSlash keeps a path's last element whole.
func noTrailingSlash(p string) string {
	if len(p) > 1 {
		return strings.TrimSuffix(p, "/")
	}
	return p
}
