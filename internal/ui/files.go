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
	cols       []fileCol // the columns that fit the last time it was drawn
	fav        bool      // the directory is a favorite, for the path row's star
}

func newTab(dir string) *tab {
	t := &tab{dir: dir, cols: allFileCols}
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

// allFileCols are the columns in the order they are drawn; the name is last
// and always there.
var allFileCols = []fileCol{colModified, colOwner, colPerms, colSize, colName}

// fileColWidth is how wide a column's values are at most; the name has no
// most, only a least it keeps before a column is left out.
var fileColWidth = map[fileCol]int{colModified: 16, colOwner: 13, colPerms: 10, colSize: 5, colName: 12}

// fitCols are the columns that fit in w cells: as the panel narrows the owner
// goes first, then the size, the modified time and the permissions; the name
// stays.
func fitCols(w int) []fileCol {
	cols := slices.Clone(allFileCols)
	for _, drop := range []fileCol{colOwner, colSize, colModified, colPerms} {
		need := 2 * (len(cols) - 1)
		for _, c := range cols {
			need += fileColWidth[c]
		}
		if need <= w {
			break
		}
		cols = slices.DeleteFunc(cols, func(c fileCol) bool { return c == drop })
	}
	return cols
}

// sortColOf is the sort column a list column sorts by.
var sortColOf = map[fileCol]sortCol{colModified: sortMtime, colOwner: sortOwner, colPerms: sortPerm, colSize: sortSize, colName: sortName}

// fileColumns are the list's columns for cols. The sort chain of the directory
// is written into the titles the way terminu writes its own (the place in a
// chain of more than one, then the direction's glyph, components/layout/panel):
// filu sorts the entries itself, directories first, which a List's sort keeps
// no hold of.
// TODO(lib FL6): drop this once a List can keep directories first under a
// sort and leave out the columns that do not fit.
func fileColumns(cols []fileCol, rules []sortRule) []terminu.Column {
	names := map[fileCol]string{colModified: "Modified", colOwner: "Owner", colPerms: "Perms", colSize: "Size", colName: "Name"}
	out := make([]terminu.Column, len(cols))
	for i, c := range cols {
		title := names[c]
		if k := sortRuleIndex(rules, sortColOf[c]); k >= 0 {
			if len(rules) > 1 {
				title += " (" + strconv.Itoa(k+1) + ")"
			}
			if rules[k].asc {
				title += " " + glyphSortAsc
			} else {
				title += " " + glyphSortDesc
			}
		}
		out[i] = terminu.Column{Title: title, Number: c == colSize, Flex: c == colName}
	}
	return out
}

// The sort glyphs of components/layout/panel.
const (
	glyphSortAsc  = "\U0000F160" // nf-fa-sort_amount_asc
	glyphSortDesc = "\U0000F161" // nf-fa-sort_amount_desc
)

// fileRows are the list's rows for the tab's entries, in the tab's columns.
func fileRows(t *tab, favs map[string]bool) []terminu.Row {
	rows := make([]terminu.Row, len(t.items))
	for i, it := range t.items {
		path := filepath.Join(t.dir, it.name)
		cells := make([]string, len(t.cols))
		for j, c := range t.cols {
			cells[j] = fileCell(it, c, favs[path])
		}
		rows[i] = terminu.Row{ID: path, Cells: cells}
	}
	return rows
}

// fileCell is one entry's value in a column, in its colours: the type's for
// the name (as eza and ls colour it), eza's for the owner, the permissions and
// the size.
func fileCell(it fileItem, c fileCol, fav bool) string {
	switch c {
	case colModified:
		return terminu.Fg(terminu.Overlay0).Render(fmtMtime(it.mtime))
	case colOwner:
		return colorOwner(terminu.Truncate(it.owner, fileColWidth[colOwner]))
	case colPerms:
		return colorPerm(clipMode(it.perm))
	case colSize:
		return colorSize(it)
	}
	color := terminu.Hex(string(fileColor(it)))
	name := terminu.Fg(color).Render(fileIcon(it) + " " + safeName(it.name))
	if fav {
		name += " " + terminu.Fg(terminu.Yellow).Render(iconPin) // a favorite directory: content, the app's (components/layout/panel)
	}
	return name
}

// fillTab sets a tab's rows from its entries: the columns, the marks from the
// bucket, the stars from the favorites.
func (a *App) fillTab(t *tab) {
	favs := a.favs.pinnedSet()
	t.fav = favs[t.dir]
	t.list.Columns = fileColumns(t.cols, sortRulesFor(t.dir))
	t.list.OnMove = func(terminu.Row) tea.Cmd { a.refreshPreview(); return nil }
	t.list.SetRows(fileRows(t, favs))
	t.list.EmptyFact, t.list.EmptyHint = emptyDir(t)
	syncMarks(t.list, a.marks.inBucket())
}

// emptyDir is what an empty directory says (components/layout/screen).
func emptyDir(t *tab) (fact, hint string) {
	if t.hidden > 0 && !t.showHidden {
		n := strconv.Itoa(t.hidden) + " hidden"
		return "Nothing here but " + n, "Press [.] to show hidden files, or [Space] to see what you can do here"
	}
	return "Nothing here", "Press [a] to add a file, or [Space] to see what you can do here"
}

// syncMarks marks the rows whose ID is in want and unmarks the rest, keeping
// the cursor where it is.
// TODO(lib FL4): a List cannot set its marks by ID; this walks the cursor over
// every row that has to change.
func syncMarks(l *terminu.List, want map[string]bool) {
	cur, hasCur := l.Current()
	have := map[string]bool{}
	for _, id := range l.Marked() {
		have[id] = true
	}
	for _, r := range l.Rows() {
		if have[r.ID] != want[r.ID] {
			l.Select(r.ID)
			l.ToggleMark()
		}
	}
	if hasCur {
		l.Select(cur.ID)
	}
}

// filesBody is [1]'s inside: the path of the tab's directory over its list.
// Zoomed with several tabs it shows every tab side by side, the one the keys go
// to lit. It embeds the current tab's list, so moving its cursor updates the
// preview the way a list's OnMove does.
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

// Empty: [1] draws its own empty state under the path, so the panel does not
// put one in its middle.
func (b *filesBody) Empty() bool { return false }

// drawTab is one tab in w × h: its path, then the list (or why the directory
// could not be read). Nothing is drawn between them: a panel draws no line
// under its header or its typing row either (components/layout/panel).
func (a *App) drawTab(t *tab, w, h int, focused bool) []string {
	out := []string{pathRow(t, w)}
	rest := h - len(out)
	if rest <= 0 {
		return terminu.Pad2(out, w, h)
	}
	if t.err != nil {
		return append(out, failRows(w, rest, "Cannot read this directory: "+friendlyErr(t.err), "Press [Esc] to go up")...)
	}
	cols := fitCols(w - 1 - terminu.PowerlineWidth() - (terminu.IconWidth() + 1))
	if !slices.Equal(cols, t.cols) {
		t.cols = cols
		a.fillTab(t)
	}
	return append(out, t.list.Draw(w, rest, focused)...)
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

// pathRow is a tab's directory: Lavender, the user's footprint, the slashes
// quiet; a star after it when the directory is a favorite.
func pathRow(t *tab, w int) string {
	star := ""
	if t.fav {
		star = " " + terminu.Fg(terminu.Yellow).Render(iconPin)
	}
	return terminu.Pad(crumbRow(t.dir, w-terminu.Width(star))+star, w)
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
		return a.t.Toast(terminu.Error, opFailedText("open "+safeName(filepath.Base(path)), err))
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
	return safeName(it.name)
}

// noTrailingSlash keeps a path's last element whole.
func noTrailingSlash(p string) string {
	if len(p) > 1 {
		return strings.TrimSuffix(p, "/")
	}
	return p
}
