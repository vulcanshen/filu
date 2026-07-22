package ui

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

const baseHex = "#1e1e2e" // catppuccin base (cursor fg on highlight)

// Nerd Font icons, built from rune values so no PUA glyph sits in source.
var (
	iconDir  = string(rune(0xf07b)) // nf-fa-folder
	iconFile = string(rune(0xf15b)) // nf-fa-file
)

type fileItem struct {
	name  string
	isDir bool
}

// listModel is panel [2]: the CWD file list. Hidden files are dropped by
// default (the '.' toggle comes later).
type listModel struct {
	dir        string
	items      []fileItem
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
	m.items, m.err = readEntries(m.dir, m.showHidden)
	m.clampCursor()
}

// readEntries lists a directory: directories first, alphabetical, dotfiles
// hidden unless showHidden. The error (e.g. permission denied) is returned so
// callers can distinguish "empty" from "unreadable".
func readEntries(dir string, showHidden bool) ([]fileItem, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var items []fileItem
	for _, e := range entries {
		if !showHidden && strings.HasPrefix(e.Name(), ".") {
			continue
		}
		items = append(items, fileItem{name: e.Name(), isDir: e.IsDir()})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].isDir != items[j].isDir {
			return items[i].isDir // directories first
		}
		return items[i].name < items[j].name
	})
	return items, nil
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

func (m listModel) view(w, rows int, focused bool) string {
	hdr := lipgloss.NewStyle().Foreground(dimColor).Render("Files")
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
	cursorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(baseHex)).Background(cursorBg).Width(w)
	dimStyle := lipgloss.NewStyle().Foreground(dimColor)
	dirStyle := lipgloss.NewStyle().Foreground(dirColor)

	var b strings.Builder
	b.WriteString(hdr + "\n")
	end := min(m.offset+rows, len(m.items))
	for i := m.offset; i < end; i++ {
		it := m.items[i]
		icon := iconFile
		if it.isDir {
			icon = iconDir
		}
		line := truncate(" "+icon+" "+it.name, w)
		switch {
		case i == m.cursor:
			line = cursorStyle.Render(line)
		case !focused:
			line = dimStyle.Render(line) // unfocused panel: recede
		case it.isDir:
			line = dirStyle.Render(line) // focused dir: sky
		}
		b.WriteString(line)
		if i < end-1 {
			b.WriteByte('\n')
		}
	}
	return b.String()
}
