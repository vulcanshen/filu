package ui

import (
	"errors"
	"io/fs"
	"os"
	"strings"
	"time"
)

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

// readEntries lists a directory: alphabetical (the list puts the directories first), dotfiles
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
	sortItems(items) // by name: the list puts the directories first and sorts by the dir's chain
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

// Column widths for the list rows; the display-width layer counts the glyphs.
const (
	colMtimeW  = 16 // "2006-01-02 15:04"
	colPermW   = 11 // a padded mode string, and room for the "Perms" header + arrow
	colOwnerW  = 13 // "user:group" (e.g. vulcan:staff), truncated when longer
	colSizeW   = 6  // compact size (e.g. 1023K), and room for the "Size" header + arrow
	colNameMin = 12 // keep at least this much name before dropping a column
)

// fmtSize is a file's compact size for the Size column; a directory shows a dash
// placeholder (filu never recurses to size a directory).
func fmtSize(it fileItem) string {
	if it.isDir {
		return "-"
	}
	return compactSize(it.size)
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
