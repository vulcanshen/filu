package ui

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Nerd Font icons (rune values so no PUA glyph sits in source).
var (
	iconCWD     = string(rune(0xf14de)) // launch dir (cd-on-quit picker)
	iconPin     = string(rune(0xf04ce)) // favorite dir — nf-md-star (matches markGlyph's MDI family)
	iconTabHere = string(rune(0xf02fa)) // "Open dir in…" picker: a tab already sits at that dir
)

type place struct {
	label string
	path  string
	icon  string
}

// placesModel holds the user's favorited directories (the field keeps its legacy
// "pinned" name). Favorites are created from the list ([f]avorite), managed in
// panel [3]'s Favorites tab (view/cursor/D-to-remove), and jumped to through the
// Goto / new-tab pickers — there is no longer a Places sidebar panel. Persisted in
// state.yaml (cursor is transient, not saved).
type placesModel struct {
	pinned []place
	cursor int // Favorites tab (panel [3]) selection
}

// moveCursor / clampCursor drive the Favorites tab selection.
func (m *placesModel) moveCursor(d int) {
	m.cursor += d
	m.clampCursor()
}

func (m *placesModel) clampCursor() {
	if m.cursor >= len(m.pinned) {
		m.cursor = len(m.pinned) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
}

// view renders the Favorites tab (panel [3]): one row per favorited directory
// with its full home-folded path (the panel is wide, so paths show in full and
// are only left-trimmed on overflow, keeping the tail visible), a yellow star,
// and the highlighted row following focus.
func (m placesModel) view(w, rows int, focused bool) string {
	if len(m.pinned) == 0 {
		return centeredNote(w, rows, "(no favorites — press f on a directory)")
	}
	star := lipgloss.NewStyle().Foreground(lipgloss.Color(ezaYellow)) // favorite = yellow star
	cursorBg := handColor
	if !focused {
		cursorBg = userColor
	}
	cur := lipgloss.NewStyle().Foreground(lipgloss.Color(baseHex)).Background(cursorBg)

	start := 0
	if focused && m.cursor >= rows {
		start = m.cursor - rows + 1
	}
	end := min(start+rows, len(m.pinned))
	prefixW := 1 + dispWidth(iconPin) + 1 // " <star> "
	var b strings.Builder
	for i := start; i < end; i++ {
		path := truncPathLeft(safeName(shortPath(m.pinned[i].path)), w-prefixW)
		if focused && i == m.cursor {
			b.WriteString(cur.Render(padDisp(" "+iconPin+" "+path, w)))
		} else {
			b.WriteString(truncate(" "+star.Render(iconPin)+" "+path, w))
		}
		if i < end-1 {
			b.WriteByte('\n')
		}
	}
	return b.String()
}

// unpin removes path from the pinned list if present.
func (m *placesModel) unpin(path string) {
	for i, p := range m.pinned {
		if p.path == path {
			m.pinned = append(m.pinned[:i], m.pinned[i+1:]...)
			return
		}
	}
}

// togglePin adds or removes a pinned directory.
func (m *placesModel) togglePin(path string) {
	for i, p := range m.pinned {
		if p.path == path {
			m.pinned = append(m.pinned[:i], m.pinned[i+1:]...)
			return
		}
	}
	m.pinned = append(m.pinned, place{label: filepath.Base(path), path: path, icon: iconPin})
}

// pinnedSet is the set of pinned directory paths, for the list's pin mark glyph
// (symmetry with marks.inBucket). nil when nothing is pinned.
func (m placesModel) pinnedSet() map[string]bool {
	if len(m.pinned) == 0 {
		return nil
	}
	s := make(map[string]bool, len(m.pinned))
	for _, p := range m.pinned {
		s[p.path] = true
	}
	return s
}

// trashDir is the system trash location. TODO: move behind the platform
// interface (unix-first: macOS ~/.Trash, Linux XDG).
func trashDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	if runtime.GOOS == "darwin" {
		return filepath.Join(home, ".Trash")
	}
	return filepath.Join(home, ".local", "share", "Trash", "files")
}
