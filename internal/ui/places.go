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
	iconHome  = string(rune(0xf015))  // nf-fa-home
	iconCWD   = string(rune(0xf450))  // nf-oct-file-directory
	iconTrash = string(rune(0xf1f8))  // nf-fa-trash
	iconRoot  = string(rune(0xf0fdf)) // nf-md server/disk
	iconPin   = string(rune(0xf005))  // nf-fa-star
)

type place struct {
	label string
	path  string
	icon  string
}

// placesModel is panel [1]: system Places + user Pinned. Rows are nav targets —
// Enter jumps panel [2] there. The cursor runs over system then pinned.
type placesModel struct {
	system []place
	pinned []place
	cursor int
}

func newPlaces() placesModel {
	var ps []place
	if cwd, err := os.Getwd(); err == nil {
		ps = append(ps, place{"CWD", cwd, iconCWD}) // startup dir, first
	}
	if home, err := os.UserHomeDir(); err == nil {
		ps = append(ps, place{"Home", home, iconHome})
	}
	if t := trashDir(); t != "" && dirExists(t) {
		ps = append(ps, place{"Recycle Bin", t, iconTrash})
	}
	ps = append(ps, place{"Root Dir", "/", iconRoot})
	return placesModel{system: ps}
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

func dirExists(p string) bool {
	info, err := os.Stat(p)
	return err == nil && info.IsDir()
}

func (m placesModel) all() []place {
	out := make([]place, 0, len(m.pinned)+len(m.system))
	return append(append(out, m.pinned...), m.system...) // pinned first
}

func (m *placesModel) move(delta int) {
	m.cursor += delta
	if m.cursor < 0 {
		m.cursor = 0
	}
	if n := len(m.all()); m.cursor > n-1 {
		m.cursor = n - 1
	}
}

func (m placesModel) current() (place, bool) {
	all := m.all()
	if m.cursor >= 0 && m.cursor < len(all) {
		return all[m.cursor], true
	}
	return place{}, false
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

func (m placesModel) view(w, rows int, focused bool) string {
	hdr := lipgloss.NewStyle().Foreground(dimColor)
	cursorBg := focusColor
	if !focused {
		cursorBg = dimColor
	}
	cur := lipgloss.NewStyle().Foreground(lipgloss.Color(baseHex)).Background(cursorBg).Width(w)

	var lines []string
	idx := 0
	render := func(ps []place) {
		for _, p := range ps {
			line := truncate(" "+p.icon+" "+p.label, w)
			if idx == m.cursor {
				line = cur.Render(line)
			}
			lines = append(lines, line)
			idx++
		}
	}

	if len(m.pinned) > 0 { // pinned on top, only when it has items
		lines = append(lines, hdr.Render("Pinned"))
		render(m.pinned)
	}
	lines = append(lines, hdr.Render("Local"))
	render(m.system)

	if len(lines) > rows {
		lines = lines[:rows]
	}
	return strings.Join(lines, "\n")
}
