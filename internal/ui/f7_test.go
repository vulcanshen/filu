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
