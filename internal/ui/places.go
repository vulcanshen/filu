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
	iconHome  = string(rune(0xf015)) // nf-fa-home
	iconTrash = string(rune(0xf1f8)) // nf-fa-trash
	iconDisk  = string(rune(0xf0a0)) // nf-fa-hdd
)

type place struct {
	label string
	path  string
	icon  string
}

// placesModel is panel [1]: system Places (+ a Pinned region, empty for now).
// Rows are nav targets — Enter jumps panel [2] there.
type placesModel struct {
	places []place
	cursor int
}

func newPlaces() placesModel {
	var ps []place
	if home, err := os.UserHomeDir(); err == nil {
		ps = append(ps, place{"Home", home, iconHome})
		if t := trashDir(); t != "" && dirExists(t) {
			ps = append(ps, place{"Recycle Bin", t, iconTrash})
		}
	}
	ps = append(ps, place{"Root", "/", iconDisk})
	if dirExists("/Volumes") {
		ps = append(ps, place{"Volumes", "/Volumes", iconDisk})
	}
	return placesModel{places: ps}
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

func (m *placesModel) move(delta int) {
	m.cursor += delta
	if m.cursor < 0 {
		m.cursor = 0
	}
	if m.cursor > len(m.places)-1 {
		m.cursor = len(m.places) - 1
	}
}

func (m placesModel) current() (place, bool) {
	if m.cursor >= 0 && m.cursor < len(m.places) {
		return m.places[m.cursor], true
	}
	return place{}, false
}

func (m placesModel) view(w, rows int, focused bool) string {
	hdr := lipgloss.NewStyle().Foreground(dimColor)
	cursorBg := focusColor
	if !focused {
		cursorBg = dimColor
	}
	cur := lipgloss.NewStyle().Foreground(lipgloss.Color(baseHex)).Background(cursorBg).Width(w)

	lines := []string{hdr.Render("PLACES")}
	for i, p := range m.places {
		line := truncate(" "+p.icon+" "+p.label, w)
		if i == m.cursor {
			line = cur.Render(line)
		}
		lines = append(lines, line)
	}
	lines = append(lines, hdr.Render("PINNED"), hdr.Render(" (按 P 加入)"))
	if len(lines) > rows {
		lines = lines[:rows]
	}
	return strings.Join(lines, "\n")
}
