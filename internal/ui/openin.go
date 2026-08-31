package ui

import (
	"path/filepath"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// openOpenInMenu opens the Favorites tab's "Open dir in…" picker for the
// highlighted favorite: New tab (unless the tab count is already at maxTabs) plus
// one entry per open panel [1] tab, each labelled with its tab mark and current
// directory. A tab already sitting at this favorite's directory is
// flagged with iconTabHere. Choosing acts on panel [1] and moves focus there.
func (m *AppModel) openOpenInMenu() tea.Cmd {
	if m.places.cursor < 0 || m.places.cursor >= len(m.places.pinned) {
		return nil
	}
	path := m.places.pinned[m.places.cursor].path
	m.openInPath = path

	var items []menuItem
	if len(m.tabs) < maxTabs {
		items = append(items, menuItem{label: "New tab", key: "n", hint: "open in a new tab"})
	}
	blank := strings.Repeat(" ", dispWidth(iconTabHere)) // keep tab marks aligned when there's no flag
	for i := range m.tabs {
		mark := blank
		if cleanDir(m.tabs[i].dir) == cleanDir(path) {
			mark = iconTabHere // this tab is already at that dir
		}
		label := mark + " " + tabMark(i) + "  " + safeName(filepath.Base(m.tabs[i].dir))
		items = append(items, menuItem{label: label, key: strconv.Itoa(i + 1)})
	}
	m.openInMenu.setItems(items, "Open dir in…")
	m.openInMenu.setSize(m.width)
	return m.openInMenu.open()
}

// advanceOpenIn commits an openInMenu choice: "n" opens a fresh tab at the
// favorite's dir, a numeral routes that existing tab there; either way focus
// moves to panel [1].
func (m *AppModel) advanceOpenIn(key string) tea.Cmd {
	cmd := m.openInMenu.close()
	if key == "n" {
		m.addTab(m.openInPath)
	} else if idx, err := strconv.Atoi(key); err == nil && idx >= 1 && idx <= len(m.tabs) {
		m.tab = idx - 1
		m.navigateActive(m.openInPath)
	}
	m.setFocus(panelList)
	m.syncWatches()
	return cmd
}
