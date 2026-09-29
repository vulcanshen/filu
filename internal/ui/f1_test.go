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

// tdp F1: while typing, the finder is an input — Enter submits, which picks the
// highlighted result at once (the first, unless the arrows moved); the arrows
// move among the results and j/k stay characters (K8); Tab hands focus to the
// list, where j/k move.
func TestF1FinderTypingEnterPicksTheHighlighted(t *testing.T) {
	m := openedSearch("/root", "a.go", "b.go", "c.go")

	if h := m.hint(200); !strings.Contains(h, "Enter:go") || !strings.Contains(h, "Tab:list") {
		t.Errorf("the hint while typing should say Enter picks and Tab goes to the list: %q", h)
	}
	m, _ = m.update(tea.KeyMsg{Type: tea.KeyDown})
	if m.mode != searchInput || m.selectedAbs() != "/root/b.go" {
		t.Fatalf("Down while typing should move to b.go and keep typing: mode %v, %q", m.mode, m.selectedAbs())
	}
	m, _ = m.update(tea.KeyMsg{Type: tea.KeyUp})
	if m.selectedAbs() != "/root/a.go" {
		t.Errorf("Up while typing should move back to a.go, got %q", m.selectedAbs())
	}
	m, _ = m.update(tea.KeyMsg{Type: tea.KeyDown})

	if typed, _ := m.update(runes("j")); typed.query != "j" {
		t.Errorf("j while typing is a character, query = %q", typed.query)
	}

	m, cmd := m.update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("Enter while typing should submit")
	}
	var picked *searchConfirmMsg
	for _, msg := range batchMsgs(cmd) {
		if c, ok := msg.(searchConfirmMsg); ok {
			picked = &c
		}
	}
	if picked == nil || picked.path != "/root/b.go" {
		t.Errorf("Enter while typing should pick the highlighted b.go, got %+v", picked)
	}
	if m.anim.owns() {
		t.Error("the finder should close on the pick")
	}

	n := openedSearch("/root", "a.go", "b.go")
	n, _ = n.update(tea.KeyMsg{Type: tea.KeyTab})
	if n.mode != searchNav {
		t.Fatal("Tab while typing should hand focus to the list")
	}
	if n, _ = n.update(runes("j")); n.selectedAbs() != "/root/b.go" {
		t.Errorf("j in the list should move, got %q", n.selectedAbs())
	}
}

// batchMsgs runs cmd and, for a batch, each command in it, returning the
// messages they produce.
func batchMsgs(cmd tea.Cmd) []tea.Msg {
	msg := cmd()
	b, ok := msg.(tea.BatchMsg)
	if !ok {
		return []tea.Msg{msg}
	}
	var out []tea.Msg
	for _, c := range b {
		if c != nil {
			out = append(out, c())
		}
	}
	return out
}
