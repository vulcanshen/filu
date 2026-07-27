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
	if !strings.Contains(bar, tabNumeral(0)) || !strings.Contains(bar, tabNumeral(1)) {
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
		if !strings.Contains(plain, tabNumeral(i)+" ") {
			t.Errorf("zoom tab %d numeral %q not padded before the cap: %q", i, tabNumeral(i), plain)
		}
	}
}

// TestTabGotoNewTabIntent: T opens the Goto finder with the new-tab intent, and
// confirming a directory opens it as a new active tab (tab 0 stays put); a plain
// Goto (go) carries no intent, so it reveals in place instead.
func TestTabGotoNewTabIntent(t *testing.T) {
	d1, d2 := t.TempDir(), t.TempDir()
	m := minModel()
	m.width, m.height = 80, 24
	m.search, m.toast = newSearch(), newToast()
	m.tabs = []listModel{newList(d1)}

	// T arms the Goto finder with the new-tab intent (the returned fd-stream cmd
	// is not executed here).
	if cmd := m.handleListKey("T"); cmd == nil {
		t.Fatal("T under the cap should open the Goto finder")
	}
	if !m.search.newTab {
		t.Error("T must open Goto with the new-tab intent")
	}

	// Confirming a directory appends it as a new active tab; tab 0 stays put.
	model, _ := m.Update(searchConfirmMsg{path: d2, newTab: true})
	m = model.(AppModel)
	if len(m.tabs) != 2 || m.tab != 1 || m.tabs[1].dir != d2 {
		t.Fatalf("goto-new-tab: len=%d tab=%d dir=%q, want 2/1/%q", len(m.tabs), m.tab, m.tabs[1].dir, d2)
	}
	if m.tabs[0].dir != d1 {
		t.Errorf("tab 0 must stay at %q, got %q", d1, m.tabs[0].dir)
	}

	// A plain Goto (go) opens with no intent — a reveal, not a new tab.
	m.handleListKey("go")
	if m.search.newTab {
		t.Error("plain Goto must not carry the new-tab intent")
	}
}

// TestTabLimitToast: at maxTabs, both t and T are blocked with a toast — no tab
// is added and T does not open the Goto finder.
func TestTabLimitToast(t *testing.T) {
	m := minModel()
	m.search, m.toast = newSearch(), newToast()
	m.tabs = []listModel{newList(t.TempDir())}
	for len(m.tabs) < maxTabs {
		m.handleListKey("t")
	}
	if len(m.tabs) != maxTabs {
		t.Fatalf("setup: want %d tabs, got %d", maxTabs, len(m.tabs))
	}

	if cmd := m.handleListKey("t"); cmd == nil { // t at the cap → toast, no new tab
		t.Error("t at the cap should return a toast cmd")
	}
	if len(m.tabs) != maxTabs {
		t.Errorf("t at the cap must not add a tab, got %d", len(m.tabs))
	}

	if cmd := m.handleListKey("T"); cmd == nil { // T at the cap → toast, no Goto
		t.Error("T at the cap should return a toast cmd")
	}
	if m.search.newTab {
		t.Error("T at the cap must not open Goto")
	}
	if len(m.tabs) != maxTabs {
		t.Errorf("T at the cap must not add a tab, got %d", len(m.tabs))
	}
}

// TestNewAndCloseTab covers panel [2]'s dynamic tabs: `t` opens a tab in the
// current tab's directory and makes it active, the count caps at maxTabs, `w`
// closes the active tab and clamps the cursor, and the last tab can't be closed.
func TestNewAndCloseTab(t *testing.T) {
	d1 := t.TempDir()
	m := AppModel{focus: panelList}
	m.tabs = []listModel{newList(d1)}

	m.handleListKey("t") // new tab duplicates the current tab's dir, made active
	if len(m.tabs) != 2 || m.tab != 1 {
		t.Fatalf("new tab: len=%d tab=%d, want 2/1", len(m.tabs), m.tab)
	}
	if m.tabs[1].dir != d1 {
		t.Errorf("new tab dir = %q, want current tab's %q", m.tabs[1].dir, d1)
	}

	for i := len(m.tabs); i <= maxTabs; i++ { // fill to the cap, then one more that must no-op
		m.handleListKey("t")
	}
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
