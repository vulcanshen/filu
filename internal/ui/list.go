package ui

import (
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
	dir    string
	items  []fileItem
	cursor int
	offset int
}

func newList(dir string) listModel {
	m := listModel{dir: dir}
	m.reload()
	return m
}

func (m *listModel) reload() {
	m.items = nil
	entries, err := os.ReadDir(m.dir)
	if err != nil {
		return // TODO: surface error (red '!' / toast) once those land
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".") {
			continue
		}
		m.items = append(m.items, fileItem{name: e.Name(), isDir: e.IsDir()})
	}
	sort.Slice(m.items, func(i, j int) bool {
		if m.items[i].isDir != m.items[j].isDir {
			return m.items[i].isDir // directories first
		}
		return m.items[i].name < m.items[j].name
	})
	m.clampCursor()
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
	if len(m.items) == 0 {
		return lipgloss.NewStyle().Foreground(dimColor).Render("(空目錄)")
	}
	cursorBg := focusColor
	if !focused {
		cursorBg = dimColor
	}
	cursorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(baseHex)).Background(cursorBg).Width(w)
	dirStyle := lipgloss.NewStyle().Foreground(focusColor)

	var b strings.Builder
	end := min(m.offset+rows, len(m.items))
	for i := m.offset; i < end; i++ {
		it := m.items[i]
		icon := iconFile
		if it.isDir {
			icon = iconDir
		}
		line := truncate(icon+" "+it.name, w)
		switch {
		case i == m.cursor:
			line = cursorStyle.Render(line)
		case it.isDir:
			line = dirStyle.Render(line)
		}
		b.WriteString(line)
		if i < end-1 {
			b.WriteByte('\n')
		}
	}
	return b.String()
}
