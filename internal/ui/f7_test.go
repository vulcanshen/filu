package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
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
