package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

// TestListTitleShowsNumerals checks the tab bar shows a per-tab Roman numeral
// (Ⅰ, Ⅱ) and no directory name (the path lives in the header bar).
func TestListTitleShowsNumerals(t *testing.T) {
	m := AppModel{focus: panelList}
	m.tabs = []listModel{{dir: "/home/me/projects"}, {dir: "/etc"}}
	bar := m.listTitle(40)
	if !strings.Contains(bar, tabMark(0)) || !strings.Contains(bar, tabMark(1)) {
		t.Errorf("tab bar should show a numeral per tab, bar=%q", bar)
	}
	if strings.Contains(bar, "projects") || strings.Contains(bar, "etc") {
		t.Errorf("tab bar must not show a dir name, bar=%q", bar)
	}
}

// TestZoomTabNumeralPadded guards a padding cell after each zoomed tab's Roman
// numeral: singleChip renders flush against its round cap, so without the pad a
// wide numeral glyph (Ⅱ/Ⅲ/Ⅳ) is clipped by the cap. tabBar already pads its
// labels; the zoom chips must too.
func TestZoomTabNumeralPadded(t *testing.T) {
	m := minModel()
	m.tabs = nil
	for range 5 {
		m.tabs = append(m.tabs, listModel{dir: "/tmp"})
	}
	m.zoom, m.focus = panelList, panelList
	top := strings.Split(m.expandedListTabs(120, 8), "\n")[0]
	plain := ansi.Strip(top)
	for i := range m.tabs {
		if !strings.Contains(plain, tabMark(i)+" ") {
			t.Errorf("zoom tab %d numeral %q not padded before the cap: %q", i, tabMark(i), plain)
		}
	}
}

// TestTabMenuNewTab: the new-tab menu's Search arms the Goto finder with the
// new-tab intent, and confirming a directory opens it as a new active tab (tab 0
// stays put). A plain Goto (move the active tab) carries no intent.
func TestTabMenuNewTab(t *testing.T) {
	d1, d2 := t.TempDir(), t.TempDir()
	m := minModel()
	m.width, m.height = 80, 24
	m.search, m.toast = newSearch(), newToast()
	m.tabs = []listModel{newList(d1)}

	// New tab → Search arms the Goto finder with the new-tab intent.
	m.openTabMenu()
	if !m.gotoNewTab {
		t.Fatal("openTabMenu should set new-tab mode")
	}
	m.advanceGotoFlow("/")
	if !m.search.isActive() || !m.search.newTab {
		t.Fatal("New tab → Search should open the finder with the new-tab intent")
	}

	// Confirming a directory appends it as a new active tab; tab 0 stays put.
	model, _ := m.Update(searchConfirmMsg{path: d2, newTab: true})
	m = model.(AppModel)
	if len(m.tabs) != 2 || m.tab != 1 || m.tabs[1].dir != d2 {
		t.Fatalf("new-tab search: len=%d tab=%d dir=%q, want 2/1/%q", len(m.tabs), m.tab, m.tabs[1].dir, d2)
	}
	if m.tabs[0].dir != d1 {
		t.Errorf("tab 0 must stay at %q, got %q", d1, m.tabs[0].dir)
	}

	// A plain Goto (Search) moves the active tab — no new-tab intent.
	m2 := minModel()
	m2.width, m2.height = 80, 24
	m2.search = newSearch()
	m2.tabs = []listModel{newList(d1)}
	m2.openGotoMenu()
	m2.advanceGotoFlow("/")
	if m2.search.newTab {
		t.Error("plain Goto must not carry the new-tab intent")
	}
}

// TestTabLimitToast: at maxTabs, t is blocked with a toast — the new-tab menu
// does not open and no tab is added.
func TestTabLimitToast(t *testing.T) {
	m := minModel()
	m.search, m.toast = newSearch(), newToast()
	m.tabs = []listModel{newList(t.TempDir())}
	for len(m.tabs) < maxTabs {
		m.addTab(m.cur().dir) // fill directly — t now opens a menu, not a tab
	}
	if len(m.tabs) != maxTabs {
		t.Fatalf("setup: want %d tabs, got %d", maxTabs, len(m.tabs))
	}

	if cmd := m.handleListKey("t"); cmd == nil { // t at the cap → toast
		t.Error("t at the cap should return a toast cmd")
	}
	if m.gotoMenu.isActive() {
		t.Error("t at the cap must not open the new-tab menu")
	}
	if len(m.tabs) != maxTabs {
		t.Errorf("t at the cap must not add a tab, got %d", len(m.tabs))
	}
}

// TestNewAndCloseTab covers panel [1]'s dynamic tabs: `t` opens the new-tab menu,
// whose Same opens a tab in the current dir and makes it active; the count caps at
// maxTabs; `w` closes the active tab and clamps the cursor; the last tab can't be
// closed.
func TestNewAndCloseTab(t *testing.T) {
	d1 := t.TempDir()
	m := AppModel{focus: panelList, gotoMenu: newGotoMenu()}
	m.tabs = []listModel{newList(d1)}

	m.handleListKey("t") // opens the new-tab picker
	if !m.gotoMenu.isActive() || !m.gotoNewTab {
		t.Fatal("t should open the new-tab picker")
	}
	m.advanceGotoFlow("s") // Same → a tab in the current dir, made active
	if len(m.tabs) != 2 || m.tab != 1 {
		t.Fatalf("Same: len=%d tab=%d, want 2/1", len(m.tabs), m.tab)
	}
	if m.tabs[1].dir != d1 {
		t.Errorf("Same tab dir = %q, want current tab's %q", m.tabs[1].dir, d1)
	}

	for len(m.tabs) < maxTabs { // fill to the cap directly
		m.addTab(m.cur().dir)
	}
	m.handleListKey("t") // at the cap → toast, no tab added
	if len(m.tabs) != maxTabs {
		t.Errorf("tab count should cap at %d, got %d", maxTabs, len(m.tabs))
	}

	m.tab = len(m.tabs) - 1
	m.handleListKey("w") // close the active (last) tab → cursor clamps down
	if len(m.tabs) != maxTabs-1 || m.tab != maxTabs-2 {
		t.Fatalf("close: len=%d tab=%d", len(m.tabs), m.tab)
	}

	m.tabs, m.tab = m.tabs[:1], 0
	m.handleListKey("w") // the last remaining tab must stay
	if len(m.tabs) != 1 {
		t.Errorf("closing the last tab should be a no-op, got len=%d", len(m.tabs))
	}
}
