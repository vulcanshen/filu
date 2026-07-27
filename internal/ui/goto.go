package ui

import (
	"strconv"

	tea "github.com/charmbracelet/bubbletea"
)

// gotoStep tracks where the Goto picker is: the root {Pinned, Search} choice, or
// the drilled-in Pinned list. Mirrors the sort picker's column→direction chain.
type gotoStep int

const (
	gotoStepRoot gotoStep = iota
	gotoStepPinned
)

// openGotoMenu opens the Goto picker (moves the active tab): the `go` chord and
// the Space menu's Goto route here. openTabMenu opens the same picker in new-tab
// mode — Same / Pinned / Search open the result in a new tab. Callers guard the
// tab count before openTabMenu.
func (m *AppModel) openGotoMenu() tea.Cmd { return m.openNavMenu(false, "Goto…") }
func (m *AppModel) openTabMenu() tea.Cmd  { return m.openNavMenu(true, "New tab…") }

func (m *AppModel) openNavMenu(newTab bool, title string) tea.Cmd {
	m.gotoNewTab = newTab
	m.gotoStep = gotoStepRoot
	m.setGotoRootItems(title)
	m.gotoMenu.setSize(m.width)
	return m.gotoMenu.open()
}

// setGotoRootItems builds the root choice: Pinned (a drill-down list) or Search
// (the $HOME finder). In new-tab mode a leading Same opens a tab in the current
// directory.
func (m *AppModel) setGotoRootItems(title string) {
	var items []menuItem
	if m.gotoNewTab {
		items = append(items, menuItem{label: "Same", key: "s", hint: "open a tab in this directory"})
	}
	items = append(items,
		menuItem{label: "Pinned", key: "p", hint: "a pinned directory"},
		menuItem{label: "Search", key: "/", hint: "search any directory under home"})
	m.gotoMenu.setItems(items, title)
}

// setGotoPinnedItems (re)populates the drilled-in Pinned list: one row per pinned
// dir (Enter or its number opens that dir — jump or new tab, per the mode; P
// unpins), or a hint line when nothing is pinned. The menu stays open on the swap.
func (m *AppModel) setGotoPinnedItems() {
	if len(m.places.pinned) == 0 {
		m.gotoMenu.setItems([]menuItem{
			{header: true, label: "Nothing pinned — press P on a directory to pin it"},
		}, "Pinned")
		return
	}
	budget := maxInnerWidth(m.width) - 8 // room for the "[N] " prefix + box chrome
	items := make([]menuItem, 0, len(m.places.pinned))
	for i, p := range m.places.pinned {
		items = append(items, menuItem{label: fitPath(p.path, budget), key: strconv.Itoa(i + 1)})
	}
	m.gotoMenu.setItems(items, "Pinned · P unpin")
}

// advanceGotoFlow handles a committed picker key. At root: Same opens a new tab
// here (new-tab mode), Search opens the finder, Pinned drills in. At the pinned
// step a number picks that dir — jumping the active tab or opening a new one per
// the mode. It closes the menu on a terminal action, stays open on a drill.
func (m *AppModel) advanceGotoFlow(key string) tea.Cmd {
	switch m.gotoStep {
	case gotoStepRoot:
		switch key {
		case "s": // Same → a new tab in the current dir (new-tab mode only)
			if m.gotoNewTab {
				cmd := m.gotoMenu.close()
				m.addTab(m.cur().dir)
				return cmd
			}
		case "/": // Search → the $HOME dirs-only finder
			if m.gotoNewTab {
				return tea.Batch(m.gotoMenu.close(), m.openGotoNewTab())
			}
			return tea.Batch(m.gotoMenu.close(), m.openGoto())
		case "p": // Pinned → drill into the list
			m.gotoStep = gotoStepPinned
			m.setGotoPinnedItems()
		}
		return nil
	case gotoStepPinned:
		if idx, err := strconv.Atoi(key); err == nil && idx >= 1 && idx <= len(m.places.pinned) {
			dir := m.places.pinned[idx-1].path
			cmd := m.gotoMenu.close()
			if m.gotoNewTab {
				m.addTab(dir)
			} else {
				m.navigateTo(dir)
			}
			m.syncWatches()
			return cmd
		}
		return nil
	}
	return nil
}

// unpinAtGotoCursor removes the highlighted pinned dir while the Pinned list is
// open, then rebuilds the list in place. With the Places sidebar removed, this
// picker is the home for unpin (was panel [1]'s P).
func (m *AppModel) unpinAtGotoCursor() tea.Cmd {
	if idx := m.gotoMenu.cursor; idx >= 0 && idx < len(m.places.pinned) {
		m.places.unpin(m.places.pinned[idx].path)
	}
	m.setGotoPinnedItems()
	return nil
}
