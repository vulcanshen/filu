package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

// tdp F1: each step of a multi-step flow is its own popup, stacked over the
// step before it (F4): picking a sort column opens the direction picker as a
// second box while the column picker stays open and unchanged beneath; Esc on
// the direction goes back to the columns.
func TestF1SortStepsAreSeparatePopups(t *testing.T) {
	m := f4Model(t)
	model, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	m = model.(AppModel)
	m.openSortColumnPicker()
	m.sortMenu.anim.state = popupOpen
	cols := len(m.sortMenu.items)

	m = press(t, m, runes("m")) // Modified → the direction step
	if !m.sortDirMenu.owns() {
		t.Fatal("picking a column should open the direction picker")
	}
	if !m.sortMenu.owns() || m.sortMenu.title != "Sort by…" || len(m.sortMenu.items) != cols {
		t.Errorf("the column picker should stay open and unchanged beneath: open %v, title %q, %d rows (had %d)",
			m.sortMenu.owns(), m.sortMenu.title, len(m.sortMenu.items), cols)
	}
	if m.sortDirMenu.title != "Sort Modified…" {
		t.Errorf("the direction picker title = %q", m.sortDirMenu.title)
	}
	if out := ansi.Strip(m.View()); !strings.Contains(out, "Sort Modified…") || !strings.Contains(out, "Sort by…") {
		t.Errorf("both steps should be drawn, the direction over the columns:\n%s", out)
	}

	m = press(t, m, escKey)
	if m.sortDirMenu.owns() || !m.sortMenu.owns() {
		t.Errorf("Esc on the direction should return to the columns: dir %v, columns %v",
			m.sortDirMenu.owns(), m.sortMenu.owns())
	}
}

// tdp F1: Goto's Favorites list is its own popup over the Goto picker, not the
// picker's content swapped in place; Esc returns to the picker.
func TestF1GotoFavoritesIsASeparatePopup(t *testing.T) {
	m := f4Model(t)
	model, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	m = model.(AppModel)
	m.places.pinned = []place{{label: "x", path: t.TempDir(), icon: iconPin}}
	m.openGotoMenu()
	m.gotoMenu.anim.state = popupOpen
	title, rows := m.gotoMenu.title, len(m.gotoMenu.items)

	m = press(t, m, runes("f"))
	if !m.gotoFavMenu.owns() {
		t.Fatal("Favorites should open the Favorites list")
	}
	if !m.gotoMenu.owns() || m.gotoMenu.title != title || len(m.gotoMenu.items) != rows {
		t.Errorf("the Goto picker should stay open and unchanged beneath: open %v, title %q, %d rows",
			m.gotoMenu.owns(), m.gotoMenu.title, len(m.gotoMenu.items))
	}

	m = press(t, m, escKey)
	if m.gotoFavMenu.owns() || !m.gotoMenu.owns() {
		t.Errorf("Esc on the Favorites list should return to the Goto picker: list %v, picker %v",
			m.gotoFavMenu.owns(), m.gotoMenu.owns())
	}

	m = press(t, m, runes("f"))
	m = press(t, m, runes("1")) // pick the favorite: the whole stack closes (T1)
	if m.gotoFavMenu.owns() || m.gotoMenu.owns() {
		t.Errorf("picking a favorite should close both boxes: list %v, picker %v",
			m.gotoFavMenu.owns(), m.gotoMenu.owns())
	}
}
