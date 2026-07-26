package ui

import (
	"strings"
	"testing"
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
