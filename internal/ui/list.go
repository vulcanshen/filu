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
	mtime  time.Time // for mtime sort
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
	// cached directory-own metadata for the top status bar, recomputed on reload
	// (never per frame). See loadDirStat.
	perm  string // dir mode string, e.g. drwxr-xr-x
	owner string // "owner:group"
	disk  string // "<free> / <total>" of the filesystem
}

func newList(dir string) listModel {
	m := listModel{dir: dir}
	m.reload()
	return m
}

func (m *listModel) reload() {
	m.items, m.hidden, m.err = readEntries(m.dir, m.showHidden)
	m.loadDirStat()
	m.clampCursor()
}

// loadDirStat caches the directory's own cheap metadata for the top status bar:
// perm + owner:group (one stat) and free/total disk (one statfs). Called from
// reload, so it refreshes on every cd / external change but never per frame; the
// item and hidden counts come live from the loaded list, costing nothing.
func (m *listModel) loadDirStat() {
	m.perm, m.owner, m.disk = "", "", ""
	fi, err := os.Stat(m.dir)
	if err != nil {
		return
	}
	m.perm = fi.Mode().String()
	if meta, ok := osStat(fi); ok {
		m.owner = userName(meta.uid) + ":" + groupName(meta.gid)
	}
	if free, total, ok := diskUsage(m.dir); ok {
		m.disk = humanSize(free) + " / " + humanSize(total)
	}
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
		if info, err := e.Info(); err == nil { // size/mtime for sorting, exec bit for colour
			it.size = info.Size()
			it.mtime = info.ModTime()
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

// view renders the file list. carried is the set of full paths sitting in the
// carries bucket; those rows get a green tick in a reserved left column, so a
// Pick reads the same as it does in the Carries tab (and doubles as
// multi-select).
func (m listModel) view(w, rows int, focused bool, carried map[string]bool) string {
	hdr := lipgloss.NewStyle().Foreground(dimColor).Render("Files" + sortHeaderSuffix())
	rows-- // reserve the section-header row
	if len(m.items) == 0 {
		msg := "(empty)"
		if m.err != nil {
			msg = "(" + friendlyErr(m.err) + ")"
		}
		return hdr + "\n" + lipgloss.NewStyle().Foreground(dimColor).Render(msg)
	}
	cursorBg := handColor // focused: current hand (subtext1)
	if !focused {
		cursorBg = userColor // unfocused: remembered position (lavender)
	}
	cursorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(baseHex)).Background(cursorBg)
	dimStyle := lipgloss.NewStyle().Foreground(dimColor)
	checkStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#a6e3a1")) // carried = green tick

	var b strings.Builder
	b.WriteString(hdr + "\n")
	end := min(m.offset+rows, len(m.items))
	// The leftmost cell is the carry mark: a tick when the file is in the bucket,
	// otherwise blank. Reserving exactly the tick's display width keeps ticked and
	// un-ticked rows aligned, and on a normal font it is one cell — so a plain row
	// reads " <icon> <name>", the original layout, with the tick sitting where
	// that leading space was.
	markW := dispWidth(pickGlyph)
	blank := strings.Repeat(" ", markW)
	for i := m.offset; i < end; i++ {
		it := m.items[i]
		inBucket := carried[filepath.Join(m.dir, it.name)]
		body := fileIcon(it) + " " + safeName(it.name)
		var line string
		switch {
		case i == m.cursor: // full-width highlight bar; tick inherits the bar fg
			lead := blank
			if inBucket {
				lead = pickGlyph
			}
			line = cursorStyle.Render(padDisp(lead+body, w))
		default:
			lead := blank
			if inBucket {
				lead = checkStyle.Render(pickGlyph)
			}
			if focused {
				body = lipgloss.NewStyle().Foreground(fileColor(it)).Render(body) // eza type colour
			} else {
				body = dimStyle.Render(body) // unfocused panel: recede
			}
			line = truncate(lead+body, w)
		}
		b.WriteString(line)
		if i < end-1 {
			b.WriteByte('\n')
		}
	}
	return b.String()
}
