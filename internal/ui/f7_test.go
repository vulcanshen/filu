package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// boxWidth is the widest line of a rendered popup: its outer width.
func boxWidth(s string) int {
	w := 0
	for _, l := range strings.Split(s, "\n") {
		w = max(w, lipgloss.Width(l))
	}
	return w
}

// f7Popups renders every popup kind at screen size w×h, each with content far
// narrower than the screen, so a box that sized itself to its content would
// show up as too narrow.
func f7Popups(t *testing.T, w, h int) map[string]string {
	t.Helper()
	out := map[string]string{}

	menu := newSpaceMenu()
	menu.setSize(w, h)
	menu.setItems([]menuItem{{label: "Copy", key: "c"}}, "[1] a")
	out["menu"] = menu.renderFull()

	c := newConfirmPopup()
	c.setSize(w)
	c.open("Delete a?", "delete")
	out["confirm"] = c.renderFull()

	in := newInputPopup()
	in.setSize(w)
	in.open(inputAdd, "New", "", fileItem{})
	out["input"] = in.renderFull()

	help := newHelpPopup()
	help.setSize(w, h)
	help.open("Keys", []helpRow{{key: "j", desc: "down"}})
	out["key reference"] = help.renderFull()

	meta := newMetaPopup()
	meta.setSize(w, h)
	meta.title, meta.rows = "a", []metaRow{{label: "Size", value: "1 B"}}
	out["metadata"] = meta.renderFull()

	bc := newBreadcrumbPopup()
	bc.setSize(w)
	bc.open(t.TempDir())
	out["breadcrumb"] = bc.renderFull()

	toast := newToast()
	toast.setSize(w)
	toast.message = "Copied"
	out["toast"] = toast.renderFull()

	vp := newDetailYank()
	vp.setSize(w, h)
	vp.open("a", []string{"x"}, false, nil)
	out["viewport"] = vp.renderFull()

	fs := newSearch()
	fs.open(t.TempDir(), w, h, false, false, make(chan fileBatchMsg, 1))
	out["finder"] = fs.renderFull()

	return out
}

// tdp F7: the toast sits at the bottom of the screen — its bottom border two
// rows up, clear of the footer (as in webu and locku) — not in the middle.
func TestF7ToastAtTheBottom(t *testing.T) {
	const w, h = 100, 40
	m := f4Model(t)
	m.toast = newToast()
	model, _ := m.Update(tea.WindowSizeMsg{Width: w, Height: h})
	m = model.(AppModel)
	m.toast.show("filu-toast-probe")
	m.toast.anim.state = popupOpen

	lines := strings.Split(ansi.Strip(m.View()), "\n")
	row := -1
	for i, l := range lines {
		if strings.Contains(l, "filu-toast-probe") {
			row = i
		}
	}
	// message, then the pad row, then the bottom border at h-3
	if want := h - 3 - 2; row != want {
		t.Errorf("toast message on row %d, want %d (bottom border at row %d)", row, want, h-3)
	}
	if !strings.Contains(lines[h-1], "menu") {
		t.Errorf("the footer should stay visible under the toast, last row = %q", lines[h-1])
	}
}

// tdp F7: every popup is min(terminal width − 2, 120) wide, whatever its
// content — on a narrow screen (one column spare each side) and on a wide one
// (the 120 cap). The finder's two boxes and their gap together take that width.
func TestF7EveryPopupHasTheFamilyWidth(t *testing.T) {
	for _, sz := range []struct{ w, h int }{{100, 40}, {80, 30}, {200, 50}} {
		want := min(sz.w-2, 120) // the rule literal, not the constant under test
		for name, box := range f7Popups(t, sz.w, sz.h) {
			if got := boxWidth(box); got != want {
				t.Errorf("screen %dx%d: %s is %d wide, want %d", sz.w, sz.h, name, got, want)
			}
		}
	}
}

// tdp F7 / K3: an input whose submit can fail opens with its error row already
// there, blank; a refused Enter writes the reason on that row and the box keeps
// its height.
func TestF7InputReservesTheErrorRow(t *testing.T) {
	m := f4Model(t)
	model, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	m = model.(AppModel)
	m.handleListKey("a") // Add: an empty name is refused
	m.inputPopup.anim.state = popupOpen

	before := strings.Split(ansi.Strip(m.inputPopup.renderFull()), "\n")
	m = press(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.inputPopup.errMsg == "" {
		t.Fatal("Enter on an empty name should be refused")
	}
	after := strings.Split(ansi.Strip(m.inputPopup.renderFull()), "\n")

	if len(after) != len(before) {
		t.Errorf("the box went from %d to %d rows when the error appeared", len(before), len(after))
	}
	errRow := len(after) - 2 // the row just above the bottom border
	if strings.TrimSpace(strings.Trim(before[errRow], "│")) != "" {
		t.Errorf("the error row should be blank before a refused Enter, got %q", before[errRow])
	}
	if !strings.Contains(after[errRow], m.inputPopup.errMsg) {
		t.Errorf("the reason should be on the reserved row, got %q", after[errRow])
	}
}

// boxRows is a rendered popup height.
func boxRows(s string) int { return strings.Count(s, "\n") + 1 }

// tdp F7: a menu keeps the height it opened with. Unfavoriting in Goto's
// Favorites list rebuilds the list shorter, and the box stays as tall.
func TestF7FavoritesListKeepsItsHeight(t *testing.T) {
	m := f4Model(t)
	model, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	m = model.(AppModel)
	m.places.pinned = []place{
		{label: "a", path: t.TempDir(), icon: iconPin},
		{label: "b", path: t.TempDir(), icon: iconPin},
		{label: "c", path: t.TempDir(), icon: iconPin},
	}
	m.openGotoMenu()
	m.gotoMenu.anim.state = popupOpen
	m = press(t, m, runes("f"))
	before := boxRows(m.gotoFavMenu.renderFull())

	m = press(t, m, runes("f")) // unfavorite the highlighted one
	if len(m.places.pinned) != 2 {
		t.Fatalf("f should unfavorite one, %d left", len(m.places.pinned))
	}
	if after := boxRows(m.gotoFavMenu.renderFull()); after != before {
		t.Errorf("the Favorites list went from %d to %d rows after unfavoriting", before, after)
	}
}

// tdp F7: the sort column picker keeps its height as a sort is added: Reset is
// there from the start, dimmed until there is a sort to reset (M6).
func TestF7SortColumnsKeepTheirHeight(t *testing.T) {
	sortByDir = map[string][]sortRule{}
	defer func() { sortByDir = map[string][]sortRule{} }()
	statePathOverride = t.TempDir() + "/state.yaml"
	defer func() { statePathOverride = "" }()

	m := f4Model(t)
	model, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	m = model.(AppModel)
	m.openSortColumnPicker()
	m.sortMenu.anim.state = popupOpen
	before := boxRows(m.sortMenu.renderFull())
	reset := m.sortMenu.items[len(m.sortMenu.items)-1]
	if reset.label != "Reset" || !reset.disabled {
		t.Errorf("Reset should be there, dimmed, before any sort: %+v", reset)
	}

	m = press(t, m, runes("m"))
	m = press(t, m, runes("d")) // Modified, descending: now there is a sort
	if after := boxRows(m.sortMenu.renderFull()); after != before {
		t.Errorf("the column picker went from %d to %d rows once a sort existed", before, after)
	}
	if reset := m.sortMenu.items[len(m.sortMenu.items)-1]; reset.disabled {
		t.Error("Reset should be live once there is a sort")
	}
}

// tdp F7: rows added to an open menu scroll inside the height it opened with
// rather than growing the box.
func TestF7MenuDoesNotGrowWhileOpen(t *testing.T) {
	menu := newSpaceMenu()
	menu.setSize(100, 40)
	menu.setItems([]menuItem{{label: "One", key: "1"}, {label: "Two", key: "2"}}, "t")
	menu.open()
	before := boxRows(menu.renderFull())
	if before != 2+4 { // two rows, two borders, two padding rows: the content sets the height
		t.Fatalf("a two-row menu opened %d rows tall, want 6", before)
	}
	menu.setItems([]menuItem{{label: "One", key: "1"}, {label: "Two", key: "2"},
		{label: "Three", key: "3"}, {label: "Four", key: "4"}}, "t")
	if after := boxRows(menu.renderFull()); after != before {
		t.Errorf("the menu went from %d to %d rows when rows were added", before, after)
	}
}

// tdp F7: the preview viewport is as tall as its content, up to the screen less
// its margins, where it scrolls — a three-line file is not a full-screen box.
func TestF7ViewportHeightFollowsContent(t *testing.T) {
	short := newDetailYank()
	short.setSize(100, 40)
	short.open("a", []string{"one", "two", "three"}, false, nil)
	if got := boxRows(short.renderFull()); got != 3+2 { // three lines and two borders
		t.Errorf("a three-line viewport is %d rows tall, want 5", got)
	}

	lines := make([]string, 100)
	for i := range lines {
		lines[i] = "x"
	}
	long := newDetailYank()
	long.setSize(100, 40)
	long.open("b", lines, false, nil)
	if got := boxRows(long.renderFull()); got != 40-6+2 {
		t.Errorf("a 100-line viewport is %d rows tall, want the screen cap %d", got, 40-6+2)
	}
}
