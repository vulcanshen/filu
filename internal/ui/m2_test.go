package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

// spaceMenus builds the Space menu for every panel (and every panel [3] tab),
// with and without a cursor item.
func spaceMenus(t *testing.T) map[string]struct {
	items []menuItem
	title string
} {
	t.Helper()
	out := map[string]struct {
		items []menuItem
		title string
	}{}
	add := func(name string, m AppModel) {
		items, title := m.buildSpaceMenu()
		out[name] = struct {
			items []menuItem
			title string
		}{items, title}
	}
	withItem := AppModel{focus: panelList}
	withItem.tabs = []listModel{{dir: "/tmp", items: []fileItem{{name: "foo.txt"}}}}
	add("[1] with item", withItem)
	empty := AppModel{focus: panelList}
	empty.tabs = []listModel{{dir: "/tmp"}}
	add("[1] empty", empty)
	add("[2]", AppModel{focus: panelDetail, tabs: empty.tabs})
	for tab, name := range []string{"[3] Marks", "[3] Tasks", "[3] Favorites"} {
		add(name, AppModel{focus: panelMarks, marksTab: tab, tabs: empty.tabs})
	}
	return out
}

// tdp M2 / M7: every panel's Space menu ends with the global operation region —
// its header and one Global operation row — so it is never empty.
func TestM2EveryMenuEndsWithGlobalRow(t *testing.T) {
	for name, sm := range spaceMenus(t) {
		n := len(sm.items)
		if n < 2 {
			t.Errorf("%s: menu too short: %+v", name, sm.items)
			continue
		}
		if row := sm.items[n-1]; row.key != globalOpKey || row.label != "Global operation" {
			t.Errorf("%s: last row should be Global operation, got %+v", name, row)
		}
		if hdr := sm.items[n-2]; !hdr.header || hdr.label != "global operation" {
			t.Errorf("%s: the global row needs its region header, got %+v", name, hdr)
		}
		if !sm.items[0].header {
			t.Errorf("%s: a panel Space menu always opens with a region header, got %+v", name, sm.items[0])
		}
	}
}

// tdp D4 (family default): the menu title is the focused panel's [N] label;
// panel [1] uses the cursor item as the label (2026-09-28 decision).
func TestD4MenuTitles(t *testing.T) {
	menus := spaceMenus(t)
	for name, want := range map[string]string{
		"[1] with item": "[1] foo.txt",
		"[1] empty":     "[1] CWD",
		"[2]":           "[2] Preview",
		"[3] Marks":     "[3] Marks",
		"[3] Tasks":     "[3] Tasks",
		"[3] Favorites": "[3] Favorites",
	} {
		if got := menus[name].title; got != want {
			t.Errorf("%s: title %q, want %q", name, got, want)
		}
	}
}

// The Global operation row has no hotkey: it renders as its plain label, and no
// keypress commits it.
func TestM2GlobalRowHasNoHotkey(t *testing.T) {
	m := newSpaceMenu()
	m.setSize(100)
	m.setItems(groupedMenu(nil, []menuItem{{label: "Zoom", key: "z"}}), "[2] Preview")
	plain := ansi.Strip(m.renderFull())
	if !strings.Contains(plain, "Global operation") || strings.Contains(plain, "[Global") || strings.Contains(plain, "[\x00") {
		t.Errorf("the global row should render as a plain label:\n%s", plain)
	}
}

// tdp M4 / F4: Enter on the Global operation row opens the global operation
// popup over the Space menu; Esc returns to the menu.
func TestM4GlobalRowOpensGlobalPopup(t *testing.T) {
	m := f4Model(t)
	m.globalMenu = newGlobalMenu()
	m = press(t, m, runes(" "))
	m.spaceMenu.cursor = m.spaceMenu.lastSelectable() // the Global operation row
	m = press(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if !m.globalMenu.owns() || !m.spaceMenu.owns() {
		t.Fatalf("Global operation should open its popup over the Space menu: global %v, menu %v",
			m.globalMenu.owns(), m.spaceMenu.owns())
	}
	if len(m.globalMenu.items) != len(globalActions) || m.globalMenu.items[0].key != "q" {
		t.Errorf("the global popup should list globalActions: %+v", m.globalMenu.items)
	}
	back := press(t, m, escKey)
	if back.globalMenu.owns() || !back.spaceMenu.owns() {
		t.Errorf("Esc should return to the Space menu: global %v, menu %v", back.globalMenu.owns(), back.spaceMenu.owns())
	}
}

// tdp K9 / F4: [q]uit in the global operation popup opens the leave flow over
// it; Esc there returns to the global popup, and the Space menu is still below.
func TestM4QuitFromGlobalPopupStacks(t *testing.T) {
	m := f4Model(t)
	m.globalMenu = newGlobalMenu()
	m = press(t, m, runes(" "))
	m.spaceMenu.cursor = m.spaceMenu.lastSelectable()
	m = press(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	m = press(t, m, tea.KeyMsg{Type: tea.KeyEnter}) // Enter on [q]uit
	if !m.quitMenu.owns() || !m.globalMenu.owns() || !m.spaceMenu.owns() {
		t.Fatalf("Quit should stack the quit picker: quit %v, global %v, menu %v",
			m.quitMenu.owns(), m.globalMenu.owns(), m.spaceMenu.owns())
	}
	m = press(t, m, escKey)
	if m.quitMenu.owns() || !m.globalMenu.owns() {
		t.Errorf("Esc on the quit picker should return to the global popup")
	}
}

// tdp M6: a dimmed row can hold the cursor, but neither Enter nor its hotkey
// commits it.
func TestM6DisabledRowIsInert(t *testing.T) {
	m := newSpaceMenu()
	m.setItems([]menuItem{
		{label: "Open", key: "o"},
		{label: "Tab", key: "t", disabled: true},
	}, "x")
	m.anim.state = popupOpen

	moved, _, _ := m.update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	if moved.cursor != 1 {
		t.Fatalf("the cursor should be able to rest on a dimmed row, got %d", moved.cursor)
	}
	if _, key, _ := moved.update(tea.KeyMsg{Type: tea.KeyEnter}); key != "" {
		t.Errorf("Enter on a dimmed row committed %q", key)
	}
	if _, key, _ := m.update(runes("t")); key != "" {
		t.Errorf("the hotkey of a dimmed row committed %q", key)
	}
	if _, key, _ := m.update(runes("o")); key != "o" {
		t.Errorf("an enabled row's hotkey should still commit, got %q", key)
	}
}

// tdp M6 / M3: panel [1]'s tab rows are always listed (Switch tab included);
// the ones that can't run right now are dimmed, not hidden.
func TestM6TabRowsDimInsteadOfHiding(t *testing.T) {
	rows := func(n int) map[string]menuItem {
		m := AppModel{focus: panelList}
		for range n {
			m.tabs = append(m.tabs, listModel{dir: "/tmp"})
		}
		items, _ := m.buildSpaceMenu()
		out := map[string]menuItem{}
		for _, it := range items {
			out[it.label] = it
		}
		return out
	}
	for _, tc := range []struct {
		tabs                        int
		switchOff, newOff, closeOff bool
	}{
		{1, true, false, true},
		{2, false, false, false},
		{maxTabs, false, true, false},
	} {
		r := rows(tc.tabs)
		for label, off := range map[string]bool{"Switch tab": tc.switchOff, "Tab": tc.newOff, "Close tab": tc.closeOff} {
			it, ok := r[label]
			if !ok {
				t.Errorf("%d tabs: %s should always be listed", tc.tabs, label)
				continue
			}
			if it.disabled != off {
				t.Errorf("%d tabs: %s disabled = %v, want %v", tc.tabs, label, it.disabled, off)
			}
		}
	}
}

// tdp M3: Switch tab on panel [1]'s Space menu moves to the next tab.
func TestM3SwitchTabFromMenu(t *testing.T) {
	m := f4Model(t)
	m.tabs = append(m.tabs[:1], newList(m.tabs[0].dir))
	m.tab = 0
	m = press(t, m, runes(" "))
	m = press(t, m, runes("l"))
	if m.tab != 1 {
		t.Errorf("Switch tab should move to tab 1, at %d", m.tab)
	}
	if m.spaceMenu.owns() {
		t.Error("Switch tab opens nothing, so the menu should close")
	}
}

// The global operation popup is wired like every other popup: its open
// animation ticks through, the window size reaches it, and View draws it.
func TestM4GlobalPopupWiring(t *testing.T) {
	m := f4Model(t)
	m.globalMenu = newGlobalMenu()
	model, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	m = model.(AppModel)
	if m.globalMenu.screenW != 100 {
		t.Errorf("WindowSizeMsg should size the global popup, screenW = %d", m.globalMenu.screenW)
	}

	m.spaceMenu.anim.state = popupOpen
	m.openGlobalMenu()
	for i := 0; i < 20 && !m.globalMenu.isInteractive(); i++ {
		model, _ = m.Update(AnimTickMsg{Target: "globalmenu"})
		m = model.(AppModel)
	}
	if !m.globalMenu.isInteractive() {
		t.Fatal("the global popup never finished opening: AnimTickMsg is not reaching it")
	}
	if out := ansi.Strip(m.View()); !strings.Contains(out, "pick a dir to cd to") {
		t.Errorf("View should draw the global popup over the Space menu:\n%s", out)
	}
}
