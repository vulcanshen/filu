package ui

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

const baseHex = "#1e1e2e" // catppuccin base (cursor fg on highlight)

// Nerd Font icons, built from rune values so no PUA glyph sits in source.
var (
	iconDir  = string(rune(0xe5ff)) // eza FOLDER (nf-custom-folder)
	iconFile = string(rune(0xf15b)) // eza FILE (nf-fa-file)
)

type fileItem struct {
	name   string
	isDir  bool
	isLink bool
	isExec bool
	size   int64     // for size sort
	mtime  time.Time // modified time (Modified column + mtime sort)
	perm   string    // mode string drwxr-xr-x (Permissions column + perm sort)
	owner  string    // "user:group" (Owner column + owner sort)
}

// listModel is panel [2]: the CWD file list. Hidden files are dropped by
// default (the '.' toggle comes later).
type listModel struct {
	dir        string
	items      []fileItem
	hidden     int   // count of dotfile entries (whether shown or not) — for the status bar
	err        error // last read error (permission denied, etc.)
	cursor     int
	offset     int
	showHidden bool
}

func newList(dir string) listModel {
	m := listModel{dir: dir}
	m.reload()
	return m
}

func (m *listModel) reload() {
	m.items, m.hidden, m.err = readEntries(m.dir, m.showHidden)
	m.clampCursor()
}

// reloadPreserving re-reads the directory but keeps the cursor on the same named
// entry when it survives, so an external add/remove (live refresh) doesn't make
// the selection jump. Falls back to a clamp when the entry is gone.
func (m *listModel) reloadPreserving() {
	name := m.cursorItem().name
	m.reload()
	if name == "" {
		return
	}
	for i, it := range m.items {
		if it.name == name {
			m.cursor = i
			return
		}
	}
}

// readEntries lists a directory: directories first, alphabetical, dotfiles
// hidden unless showHidden. The error (e.g. permission denied) is returned so
// callers can distinguish "empty" from "unreadable".
func readEntries(dir string, showHidden bool) ([]fileItem, int, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, 0, err
	}
	var items []fileItem
	hidden := 0
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".") {
			hidden++
			if !showHidden {
				continue
			}
		}
		it := fileItem{
			name:   e.Name(),
			isDir:  e.IsDir(),
			isLink: e.Type()&os.ModeSymlink != 0,
		}
		if info, err := e.Info(); err == nil { // size/mtime/perm/owner for the columns + sort, exec bit for colour
			it.size = info.Size()
			it.mtime = info.ModTime()
			it.perm = info.Mode().String()
			if meta, ok := osStat(info); ok {
				it.owner = userName(meta.uid) + ":" + groupName(meta.gid)
			}
			if !it.isDir && !it.isLink {
				it.isExec = info.Mode()&0o111 != 0
			}
		}
		items = append(items, it)
	}
	sortItems(items) // directories first, then the active sort chain
	return items, hidden, nil
}

// safeName strips control characters from a name before it is displayed, so an
// embedded CR (macOS "Icon\r" custom-icon files), ESC, NUL, tab, etc. can't
// reset the cursor / inject ANSI and shatter the layout. Apply to the RAW name
// before any lipgloss styling — never to an already-styled string. The real
// name (with the control byte) is kept for file operations.
func safeName(s string) string {
	if !strings.ContainsFunc(s, func(r rune) bool { return r < 0x20 || r == 0x7f }) {
		return s // fast path: the common case has no control chars
	}
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if r < 0x20 || r == 0x7f {
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

// friendlyErr turns a read error into a short label.
func friendlyErr(err error) string {
	switch {
	case errors.Is(err, fs.ErrPermission):
		return "permission denied"
	case errors.Is(err, fs.ErrNotExist):
		return "not found"
	default:
		return "unreadable"
	}
}

func (m *listModel) clampCursor() {
	if m.cursor > len(m.items)-1 {
		m.cursor = len(m.items) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
}

func (m *listModel) move(delta int) {
	m.cursor += delta
	m.clampCursor()
}

func (m listModel) cursorItem() fileItem {
	if m.cursor >= 0 && m.cursor < len(m.items) {
		return m.items[m.cursor]
	}
	return fileItem{}
}

// enter descends into the cursor directory. Opening files goes through the OS
// later; for now a file is a no-op.
func (m *listModel) enter() {
	if m.cursor < len(m.items) && m.items[m.cursor].isDir {
		m.dir = filepath.Join(m.dir, m.items[m.cursor].name)
		m.cursor, m.offset = 0, 0
		m.reload()
	}
}

// parent goes up one directory, landing the cursor on the dir we came from.
func (m *listModel) parent() {
	up := filepath.Dir(m.dir)
	if up == m.dir {
		return // already at root
	}
	came := filepath.Base(m.dir)
	m.dir, m.cursor, m.offset = up, 0, 0
	m.reload()
	for i, it := range m.items {
		if it.name == came {
			m.cursor = i
			break
		}
	}
}

func (m *listModel) ensureVisible(rows int) {
	if rows <= 0 {
		return
	}
	switch {
	case m.cursor < m.offset:
		m.offset = m.cursor
	case m.cursor >= m.offset+rows:
		m.offset = m.cursor - rows + 1
	}
	if m.offset < 0 {
		m.offset = 0
	}
}

// Column widths for the list rows; the display-width layer counts the glyphs.
const (
	colMtimeW  = 16 // "2006-01-02 15:04"
	colPermW   = 11 // a padded mode string, and room for the "Perms" header + arrow
	colOwnerW  = 16 // "user:group", truncated when longer
	colNameMin = 12 // keep at least this much name before dropping a column
)

// markCellW is the combined width of the two mark slots (mark + favorite), fixed
// so toggling a mark or a favorite never shifts the columns that follow.
func markCellW() int { return dispWidth(markGlyph) + dispWidth(iconPin) }

// listCols is which optional columns fit at a given inner width. They drop in the
// order owner → perms → mtime → mark as the panel narrows; the name always stays.
type listCols struct {
	mark, mtime, perm, owner bool
	nameW                    int
}

func computeListCols(w int) listCols {
	mk := markCellW()
	// width before the name for each level of columns shown (each column has a
	// trailing space).
	all := mk + 1 + colMtimeW + 1 + colPermW + 1 + colOwnerW + 1
	noOwner := mk + 1 + colMtimeW + 1 + colPermW + 1
	noPerm := mk + 1 + colMtimeW + 1
	noMtime := mk + 1
	switch {
	case w >= all+colNameMin:
		return listCols{mark: true, mtime: true, perm: true, owner: true, nameW: w - all}
	case w >= noOwner+colNameMin:
		return listCols{mark: true, mtime: true, perm: true, nameW: w - noOwner}
	case w >= noPerm+colNameMin:
		return listCols{mark: true, mtime: true, nameW: w - noPerm}
	case w >= noMtime+colNameMin:
		return listCols{mark: true, nameW: w - noMtime}
	default:
		return listCols{nameW: w}
	}
}

// fmtMtime formats a modified time as "2006-01-02 15:04"; a zero time is blank.
func fmtMtime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format("2006-01-02 15:04")
}

// clipMode bounds a mode string to the perms column so special modes (setuid,
// sticky) can't overrun it.
func clipMode(perm string) string {
	if len(perm) > colPermW {
		return perm[:colPermW]
	}
	return perm
}

// markCell renders the two status slots — mark (green tick) then favorite
// (lavender star) — each a blank of the same width when not set, so toggling never
// shifts the columns. coloured=false leaves the glyphs plain so a highlighted
// cursor row can recolour them with the bar.
func markCell(carried, pinned, coloured bool) string {
	carrySlot := strings.Repeat(" ", dispWidth(markGlyph))
	if carried {
		carrySlot = markGlyph
		if coloured {
			carrySlot = lipgloss.NewStyle().Foreground(lipgloss.Color("#a6e3a1")).Render(markGlyph)
		}
	}
	pinSlot := strings.Repeat(" ", dispWidth(iconPin))
	if pinned {
		pinSlot = iconPin
		if coloured {
			pinSlot = lipgloss.NewStyle().Foreground(userColor).Render(iconPin)
		}
	}
	return carrySlot + pinSlot
}

// sortColHeader renders one column-header label: dim when the column is not in
// the sort chain, else brightened + bold with an asc/desc arrow — so the header
// row doubles as the sort indicator.
func sortColHeader(label string, col sortCol) string {
	i := sortChainIndex(col)
	if i < 0 {
		return lipgloss.NewStyle().Foreground(dimColor).Render(label)
	}
	arrow := sortAscGlyph
	if !sortChain[i].asc {
		arrow = sortDescGlyph
	}
	return lipgloss.NewStyle().Foreground(handColor).Bold(true).Render(label) + " " +
		lipgloss.NewStyle().Foreground(focusColor).Render(arrow)
}

// listHeaderRow is the column-header line above the file rows: the sortable
// column labels (Modified / Perms / Name) aligned to the row columns. The mark
// column carries no label.
func listHeaderRow(cols listCols, w int) string {
	var b strings.Builder
	if cols.mark {
		b.WriteString(strings.Repeat(" ", markCellW()+1))
	}
	if cols.mtime {
		b.WriteString(padDisp(sortColHeader("Modified", sortMtime), colMtimeW) + " ")
	}
	if cols.perm {
		b.WriteString(padDisp(sortColHeader("Perms", sortPerm), colPermW) + " ")
	}
	if cols.owner {
		b.WriteString(padDisp(sortColHeader("Owner", sortOwner), colOwnerW) + " ")
	}
	b.WriteString(sortColHeader("Name", sortName))
	return truncate(b.String(), w)
}

// renderListRow renders one file row: mark | modified | perms | owner | icon name,
// with whichever columns fit (cols). The cursor row is drawn plain on a full-width
// highlight bar; other rows colour each column (dim mtime, eza perms/owner, type-
// coloured name), receding to dim when the panel is unfocused.
func renderListRow(it fileItem, cols listCols, w int, cursor, focused, carried, pinned bool) string {
	name := truncate(fileIcon(it)+" "+safeName(it.name), cols.nameW)
	if cursor { // plain content on a full-width highlight bar
		var b strings.Builder
		if cols.mark {
			b.WriteString(markCell(carried, pinned, false) + " ")
		}
		if cols.mtime {
			b.WriteString(padDisp(fmtMtime(it.mtime), colMtimeW) + " ")
		}
		if cols.perm {
			b.WriteString(padDisp(clipMode(it.perm), colPermW) + " ")
		}
		if cols.owner {
			b.WriteString(padDisp(truncate(it.owner, colOwnerW), colOwnerW) + " ")
		}
		b.WriteString(name)
		cursorBg := handColor // focused: current hand (subtext1)
		if !focused {
			cursorBg = userColor // unfocused: remembered position (lavender)
		}
		return lipgloss.NewStyle().Foreground(lipgloss.Color(baseHex)).Background(cursorBg).Render(padDisp(b.String(), w))
	}
	var b strings.Builder
	if cols.mark {
		b.WriteString(markCell(carried, pinned, true) + " ")
	}
	if cols.mtime {
		b.WriteString(lipgloss.NewStyle().Foreground(dimColor).Render(padDisp(fmtMtime(it.mtime), colMtimeW)) + " ")
	}
	if cols.perm {
		b.WriteString(padDisp(colorPerm(clipMode(it.perm)), colPermW) + " ")
	}
	if cols.owner {
		b.WriteString(padDisp(colorOwner(truncate(it.owner, colOwnerW)), colOwnerW) + " ")
	}
	if focused {
		b.WriteString(lipgloss.NewStyle().Foreground(fileColor(it)).Render(name)) // eza type colour
	} else {
		b.WriteString(lipgloss.NewStyle().Foreground(dimColor).Render(name)) // unfocused: recede
	}
	return truncate(b.String(), w)
}

// view renders the file list: a column-header row, then one renderListRow per
// visible entry. carried / pinned are the sets of full paths in the carries
// bucket and the Pinned list, driving the two mark glyphs.
func (m listModel) view(w, rows int, focused bool, carried, pinned map[string]bool) string {
	cols := computeListCols(w)
	header := listHeaderRow(cols, w)
	rows-- // reserve the column-header row
	if len(m.items) == 0 {
		msg := "(empty)"
		if m.err != nil {
			msg = "(" + friendlyErr(m.err) + ")"
		}
		return header + "\n" + lipgloss.NewStyle().Foreground(dimColor).Render(msg)
	}
	var b strings.Builder
	b.WriteString(header + "\n")
	end := min(m.offset+rows, len(m.items))
	for i := m.offset; i < end; i++ {
		it := m.items[i]
		path := filepath.Join(m.dir, it.name)
		b.WriteString(renderListRow(it, cols, w, i == m.cursor, focused, carried[path], pinned[path]))
		if i < end-1 {
			b.WriteByte('\n')
		}
	}
	return b.String()
}
