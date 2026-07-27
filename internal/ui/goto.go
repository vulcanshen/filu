package ui

import (
	"os"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// gotoStep tracks where the Goto picker is: the root {Pinned, Search} choice, or
// the drilled-in Pinned list. Mirrors the sort picker's column→direction chain.
type gotoStep int

const (
	gotoStepRoot gotoStep = iota
	gotoStepPinned
)

// openGotoMenu opens the Goto picker. Both the `go` chord and the panel [2] Space
// menu's Goto route here. Root offers Pinned (a drill-down list of pinned dirs) or
// Search (the $HOME dirs-only finder, unchanged).
func (m *AppModel) openGotoMenu() tea.Cmd {
	m.gotoStep = gotoStepRoot
	m.setGotoRootItems()
	m.gotoMenu.setSize(m.width)
	return m.gotoMenu.open()
}

func (m *AppModel) setGotoRootItems() {
	items := []menuItem{
		{label: "Pinned", key: "p", hint: "jump to a pinned directory"},
		{label: "Search", key: "/", hint: "fuzzy-jump to any directory under home"},
	}
	m.gotoMenu.setItems(items, "Goto…")
}

// setGotoPinnedItems (re)populates the drilled-in Pinned list: one row per pinned
// dir (Enter or its number jumps the active tab there, P unpins), or a hint line
// when nothing is pinned. The menu stays open across the swap.
func (m *AppModel) setGotoPinnedItems() {
	if len(m.places.pinned) == 0 {
		m.gotoMenu.setItems([]menuItem{
			{header: true, label: "Nothing pinned — press P on a directory to pin it"},
		}, "Pinned")
		return
	}
	items := make([]menuItem, 0, len(m.places.pinned))
	for i, p := range m.places.pinned {
		items = append(items, menuItem{label: collapseHome(p.path), key: strconv.Itoa(i + 1)})
	}
	m.gotoMenu.setItems(items, "Pinned · P unpin")
}

// advanceGotoFlow handles a committed Goto-picker key: at root, Search opens the
// finder and Pinned drills in; at the pinned step, a number jumps the active tab
// to that pinned dir. It closes the menu on a terminal action, keeps it open on a
// drill (mirrors advanceSortFlow).
func (m *AppModel) advanceGotoFlow(key string) tea.Cmd {
	switch m.gotoStep {
	case gotoStepRoot:
		switch key {
		case "/": // Search → the $HOME dirs-only finder (unchanged behaviour)
			return tea.Batch(m.gotoMenu.close(), m.openGoto())
		case "p": // Pinned → drill into the list
			m.gotoStep = gotoStepPinned
			m.setGotoPinnedItems()
		}
		return nil
	case gotoStepPinned:
		if idx, err := strconv.Atoi(key); err == nil && idx >= 1 && idx <= len(m.places.pinned) {
			cmd := m.gotoMenu.close()
			m.navigateTo(m.places.pinned[idx-1].path)
			m.syncWatches()
			return cmd
		}
		return nil
	}
	return nil
}

// unpinAtGotoCursor removes the highlighted pinned dir while the Pinned list is
// open, then rebuilds the list in place. With the Places sidebar going away, this
// picker is the new home for unpin (was panel [1]'s P).
func (m *AppModel) unpinAtGotoCursor() tea.Cmd {
	if idx := m.gotoMenu.cursor; idx >= 0 && idx < len(m.places.pinned) {
		m.places.unpin(m.places.pinned[idx].path)
	}
	m.setGotoPinnedItems()
	return nil
}

// collapseHome shortens a path for display by replacing the home dir with ~. The
// separator guard keeps /home/bob from swallowing /home/bobby.
func collapseHome(path string) string {
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		if path == home {
			return "~"
		}
		if strings.HasPrefix(path, home+string(os.PathSeparator)) {
			return "~" + path[len(home):]
		}
	}
	return path
}
