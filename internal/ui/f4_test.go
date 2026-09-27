package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

func runes(s string) tea.KeyMsg { return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)} }

// f4Model is minModel with the active tab on a real directory holding one file
// (so the Space menu has item rows) and every stackable popup constructed.
func f4Model(t *testing.T) AppModel {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("a"), 0o644); err != nil {
		t.Fatal(err)
	}
	m := minModel()
	m.toast = newToast()
	m.spaceMenu = newSpaceMenu()
	m.searchMenu = newSearchMenu()
	m.search = newSearch()
	m.tabs[0] = newList(dir)
	return m
}

// press sends key and returns the model, finishing any open animation so the
// popup on top can take the next key.
func press(t *testing.T, m AppModel, key tea.KeyMsg) AppModel {
	t.Helper()
	model, _ := m.Update(key)
	got := model.(AppModel)
	for _, a := range got.stackOrder() {
		if a.state == popupOpeningLine || a.state == popupOpeningExpand {
			a.state = popupOpen
		}
	}
	return got
}

// tdp F4: a Space-menu row that opens a popup leaves the menu beneath it;
// cancelling that popup with Esc comes back to the menu.
func TestF4SpaceMenuStaysUnderItsPopup(t *testing.T) {
	m := press(t, f4Model(t), runes(" "))
	m = press(t, m, runes("r")) // Rename → input popup
	if !m.inputPopup.owns() {
		t.Fatal("Rename from the Space menu should open the input popup")
	}
	if !m.spaceMenu.owns() {
		t.Fatal("the Space menu should stay beneath the input popup")
	}
	m = press(t, m, escKey)
	if m.inputPopup.owns() {
		t.Error("Esc should close the input popup")
	}
	if !m.spaceMenu.owns() {
		t.Error("Esc on the input popup should come back to the Space menu")
	}
}

// A row that just runs (no popup) still closes the menu.
func TestF4SpaceMenuClosesAfterPlainAction(t *testing.T) {
	m := press(t, f4Model(t), runes(" "))
	m = press(t, m, runes(".")) // Hidden: toggles, opens nothing
	if m.spaceMenu.owns() {
		t.Error("a row that opens no popup should close the Space menu")
	}
}

// tdp T1: finishing the action a popup was for clears the whole stack,
// including the Space menu it came from.
func TestF4CompletingClearsTheStack(t *testing.T) {
	m := f4Model(t)
	m.spaceMenu.anim.state = popupOpen
	m.confirm.open("Go ahead?", "go")
	m.confirm.anim.state = popupOpen
	m.confirmAction = confirmNone

	m = press(t, m, runes("y"))
	if m.confirm.owns() || m.spaceMenu.owns() {
		t.Errorf("accepting the confirm should clear the stack: confirm %v, Space menu %v",
			m.confirm.owns(), m.spaceMenu.owns())
	}
}

// tdp F4: the Search chooser stays beneath the finder it opened; Esc on the
// finder comes back to the chooser.
func TestF4SearchChooserStaysUnderFinder(t *testing.T) {
	m := f4Model(t)
	m.openSearchMenu()
	m.searchMenu.anim.state = popupOpen

	m = press(t, m, runes("f"))
	if !m.search.owns() {
		t.Fatal("f on the chooser should open the finder")
	}
	if !m.searchMenu.owns() {
		t.Fatal("the chooser should stay beneath the finder")
	}
	m = press(t, m, escKey)
	if m.search.owns() || !m.searchMenu.owns() {
		t.Errorf("Esc on the finder should return to the chooser: finder %v, chooser %v",
			m.search.owns(), m.searchMenu.owns())
	}
}

// tdp F4: likewise the Goto picker's Search row keeps the picker beneath the
// finder.
func TestF4GotoPickerStaysUnderFinder(t *testing.T) {
	m := f4Model(t)
	m.openGotoMenu()
	m.gotoMenu.anim.state = popupOpen

	m = press(t, m, runes("/"))
	if !m.search.owns() || !m.gotoMenu.owns() {
		t.Errorf("Search on the Goto picker should open the finder over it: finder %v, picker %v",
			m.search.owns(), m.gotoMenu.owns())
	}
}

// tdp T1: the finder's pick clears whatever led to it.
func TestF4FinderPickClearsTheStack(t *testing.T) {
	m := f4Model(t)
	m.spaceMenu.anim.state = popupOpen
	m.searchMenu.anim.state = popupOpen
	model, _ := m.Update(searchConfirmMsg{path: m.tabs[0].dir})
	got := model.(AppModel)
	if got.spaceMenu.owns() || got.searchMenu.owns() {
		t.Errorf("a finder pick should clear the stack: Space menu %v, chooser %v",
			got.spaceMenu.owns(), got.searchMenu.owns())
	}
}

// tdp D2: each open popup takes the colour of its depth in the stack.
func TestF4LayerColourFollowsDepth(t *testing.T) {
	m := f4Model(t)
	m.confirm.anim.state = popupOpen
	m.assignLayers()
	if m.confirm.anim.layer != 1 {
		t.Errorf("a lone confirm should be layer 1, got %d", m.confirm.anim.layer)
	}

	m.spaceMenu.anim.state = popupOpen
	m.assignLayers()
	if m.spaceMenu.anim.layer != 1 || m.confirm.anim.layer != 2 {
		t.Errorf("Space menu + confirm: layers %d, %d, want 1, 2", m.spaceMenu.anim.layer, m.confirm.anim.layer)
	}
	if m.confirm.anim.color != popupLayerColor(2) {
		t.Errorf("the confirm on layer 2 should take layer 2's colour, got %s", m.confirm.anim.color)
	}
}

// tdp T1: each popup's own completion clears the stack too — here with the
// Space menu left beneath it. (The open-with picker's run launches a real app,
// so it is not driven here; it goes through the same clearStack.)
func TestF4EveryCompletionClearsTheStack(t *testing.T) {
	for _, tc := range []struct {
		name string
		up   func(m *AppModel)
		key  tea.KeyMsg
	}{
		{"breadcrumb jump", func(m *AppModel) {
			m.breadcrumb.open(m.tabs[0].dir)
			m.breadcrumb.anim.state = popupOpen
		}, tea.KeyMsg{Type: tea.KeyEnter}},
		{"input submit", func(m *AppModel) {
			m.inputPopup.open(inputRename, "Rename", "a.txt", fileItem{name: "a.txt"})
			m.inputPopup.anim.state = popupOpen
		}, tea.KeyMsg{Type: tea.KeyEnter}},
		{"open-in pick", func(m *AppModel) {
			m.openInPath = m.tabs[0].dir
			m.openInMenu.setItems([]menuItem{{label: "New tab", key: "n"}}, "Open dir in…")
			m.openInMenu.anim.state = popupOpen
		}, runes("n")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := f4Model(t)
			m.tabs = m.tabs[:1]
			m.spaceMenu.anim.state = popupOpen
			tc.up(&m)
			m = press(t, m, tc.key)
			if m.spaceMenu.owns() {
				t.Errorf("%s should clear the Space menu beneath it", tc.name)
			}
		})
	}
}

// tdp D3: the popup a Space-menu row opened is drawn over the menu.
func TestF4ChildDrawnOverSpaceMenu(t *testing.T) {
	m := f4Model(t)
	m.width, m.height = 100, 40
	m = press(t, m, runes(" "))
	m = press(t, m, runes("D")) // Delete → confirm over the menu
	if out := ansi.Strip(m.View()); !strings.Contains(out, "a.txt to the trash?") {
		t.Errorf("the confirm should be drawn over the Space menu:\n%s", out)
	}
}
