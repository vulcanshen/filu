package ui

import (
	"path/filepath"
	"strconv"

	tea "github.com/charmbracelet/bubbletea"
)

// openDefault opens the cursor item straight away with the OS default app — the
// [o] fast path (no picker). [O] (openOpenWith) shows the app chooser instead.
// It works on a file or a directory.
func (m *AppModel) openDefault() tea.Cmd {
	l := m.cur()
	it := l.cursorItem()
	if it.name == "" {
		return nil
	}
	return openFileCmd(filepath.Join(l.dir, it.name))
}

// openOpenWith opens the [O]pen-with picker for the cursor item: a "Default"
// entry (the OS default app) followed by each app configured in config.yaml's
// open_with. A number or j/k + Enter runs the choice; Esc closes. It works on a
// file or a directory (e.g. open the current folder in your IDE).
func (m *AppModel) openOpenWith() tea.Cmd {
	l := m.cur()
	it := l.cursorItem()
	if it.name == "" {
		return nil
	}
	m.openWithPath = filepath.Join(l.dir, it.name)
	items := make([]menuItem, 0, len(openWithApps)+1)
	items = append(items, menuItem{label: "Default", key: "1", hint: "OS default app"})
	for i, a := range openWithApps {
		items = append(items, menuItem{label: a.Name, key: strconv.Itoa(i + 2), hint: a.Cmd})
	}
	m.openWithMenu.setItems(items, "Open with…")
	m.openWithMenu.setSize(m.width)
	return m.openWithMenu.open()
}

// runOpenWith launches the picked entry against openWithPath: index 1 is the OS
// default (openFileCmd), 2+ are the configured open_with apps in order.
func (m AppModel) runOpenWith(idx int) tea.Cmd {
	if m.openWithPath == "" {
		return nil
	}
	if idx == 1 {
		return openFileCmd(m.openWithPath)
	}
	if i := idx - 2; i >= 0 && i < len(openWithApps) {
		return openWithCmd(openWithApps[i].Cmd, m.openWithPath)
	}
	return nil
}
