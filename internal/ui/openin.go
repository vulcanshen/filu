package ui

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// openOpenInMenu opens the Favorites tab's "Open dir in…" picker for the
// highlighted favorite: New tab (dimmed once the tab count is at maxTabs) plus
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
	// New tab is always offered; at maxTabs it is dimmed rather than hidden (tdp M6).
	items = append(items, menuItem{label: "New tab", key: "n", hint: "open in a new tab", disabled: len(m.tabs) >= maxTabs})
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
	m.openInMenu.setSize(m.width, m.height)
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

// showInTabs brings dir up in panel [1] — Enter on a mark or a favorite (tdp K3,
// 2026-09-28 decision): the tab already showing dir if there is one, else a new
// tab there; with every tab in use it says so instead. When name is set (a
// marked file) the cursor lands on it. Focus moves to [1] to show the result.
func (m *AppModel) showInTabs(dir, name string) tea.Cmd {
	at := -1
	for i := range m.tabs {
		if cleanDir(m.tabs[i].dir) == cleanDir(dir) {
			at = i
			break
		}
	}
	if at < 0 {
		if len(m.tabs) >= maxTabs {
			return m.toast.show(fmt.Sprintf("All %d tabs are in use — close one (w) to open %s", maxTabs, safeName(filepath.Base(dir))))
		}
		m.addTab(dir)
		at = m.tab
	}
	m.tab = at
	l := m.cur()
	if name != "" && !l.focusEntry(name) { // a dotfile the tab hides: reveal hidden and retry
		l.showHidden = true
		l.reload()
		l.focusEntry(name)
	}
	m.setFocus(panelList)
	m.syncWatches()
	l.ensureVisible(m.listRows())
	m.refreshPreview()
	return nil
}
