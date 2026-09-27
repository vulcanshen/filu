package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// TestOpenInMenuItems: the picker offers New tab (dimmed at maxTabs) plus one
// entry per open tab; a tab not at the favorite's dir carries no flag.
func TestOpenInMenuItems(t *testing.T) {
	m := minModel() // 3 tabs at /tmp
	m.places.pinned = []place{{path: "/a", icon: iconPin}}
	m.places.cursor = 0

	m.openOpenInMenu()
	items := m.openInMenu.items
	if len(items) != 4 { // New tab + 3 tab entries
		t.Fatalf("want New tab + 3 tab entries = 4, got %d", len(items))
	}
	if items[0].key != "n" {
		t.Errorf("first item should be New tab (key n), got %q", items[0].key)
	}
	for _, it := range items[1:] {
		if strings.Contains(it.label, iconTabHere) {
			t.Errorf("no tab is at /a, so none should be flagged: %q", it.label)
		}
	}
}

// TestOpenInMenuFlagsOpenTab: a tab already sitting at the favorite's dir is
// flagged with iconTabHere.
func TestOpenInMenuFlagsOpenTab(t *testing.T) {
	m := minModel() // 3 tabs at /tmp
	m.places.pinned = []place{{path: "/tmp", icon: iconPin}}
	m.places.cursor = 0

	m.openOpenInMenu()
	flagged := 0
	for _, it := range m.openInMenu.items {
		if strings.Contains(it.label, iconTabHere) {
			flagged++
		}
	}
	if flagged != 3 { // all 3 tabs are at /tmp
		t.Errorf("all 3 tabs at /tmp should be flagged, got %d", flagged)
	}
}

// TestOpenInMenuNewTabDimmedWhenFull: at maxTabs New tab is still listed, dimmed,
// and neither Enter nor n on it does anything (tdp M6).
func TestOpenInMenuNewTabDimmedWhenFull(t *testing.T) {
	m := minModel()
	m.tabs = []listModel{{dir: "/a"}, {dir: "/b"}, {dir: "/c"}, {dir: "/d"}, {dir: "/e"}} // == maxTabs
	m.places.pinned = []place{{path: "/x", icon: iconPin}}
	m.places.cursor = 0

	m.openOpenInMenu()
	if len(m.openInMenu.items) != 6 {
		t.Fatalf("want New tab + 5 tab entries = 6, got %d", len(m.openInMenu.items))
	}
	if it := m.openInMenu.items[0]; it.key != "n" || !it.disabled {
		t.Errorf("at maxTabs New tab should be listed and dimmed: %+v", it)
	}
	m.openInMenu.anim.state = popupOpen
	for _, k := range []tea.KeyMsg{{Type: tea.KeyEnter}, {Type: tea.KeyRunes, Runes: []rune("n")}} {
		if _, key, _ := m.openInMenu.update(k); key != "" {
			t.Errorf("%q on the dimmed New tab committed %q", k.String(), key)
		}
	}
}

// TestOpenInNewTab: choosing New tab appends a tab at the favorite's dir and
// moves focus to panel [1].
func TestOpenInNewTab(t *testing.T) {
	m := minModel() // 3 tabs
	fav := t.TempDir()
	m.openInPath = fav
	m.openInMenu.anim.state = popupOpen
	m.focus = panelMarks

	m.advanceOpenIn("n")
	if len(m.tabs) != 4 {
		t.Fatalf("New tab should append a 4th tab, got %d", len(m.tabs))
	}
	if m.tabs[3].dir != fav {
		t.Errorf("new tab dir = %q, want %q", m.tabs[3].dir, fav)
	}
	if m.focus != panelList {
		t.Error("focus should move to panel [1] after opening")
	}
}

// TestOpenInExistingTab: choosing a numeral routes that existing tab to the
// favorite's dir (no new tab) and moves focus to panel [1].
func TestOpenInExistingTab(t *testing.T) {
	m := minModel() // 3 tabs at /tmp
	fav := t.TempDir()
	m.openInPath = fav
	m.openInMenu.anim.state = popupOpen
	m.focus = panelMarks

	m.advanceOpenIn("2") // route tab index 1 (Ⅱ)
	if len(m.tabs) != 3 {
		t.Errorf("routing an existing tab must not add tabs, got %d", len(m.tabs))
	}
	if m.tabs[1].dir != fav {
		t.Errorf("tab 2 (index 1) should navigate to %q, got %q", fav, m.tabs[1].dir)
	}
	if m.focus != panelList {
		t.Error("focus should move to panel [1]")
	}
}
