package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

// TestListNavHintFocusGated: the list's key legend appears only while focused.
func TestListNavHintFocusGated(t *testing.T) {
	if got := listNavHint(false); got != "" {
		t.Errorf("unfocused list should have no hint, got %q", got)
	}
	plain := ansi.Strip(listNavHint(true))
	for _, want := range []string{"enter into", "esc back", "jkud move", "hl switch tab"} {
		if !strings.Contains(plain, want) {
			t.Errorf("focused list hint missing %q, got %q", want, plain)
		}
	}
}

// TestMarksHint: the Marks panel legend names the marks-workflow keys.
func TestMarksHint(t *testing.T) {
	plain := ansi.Strip(marksHint())
	for _, want := range []string{"m mark", "c copy", "v move"} {
		if !strings.Contains(plain, want) {
			t.Errorf("marks hint missing %q, got %q", want, plain)
		}
	}
}

// TestPanelBoxHintBottomBorder: the hint lands in the bottom border, the box
// still measures exactly its width on every line, and "" keeps the edge clean.
func TestPanelBoxHintBottomBorder(t *testing.T) {
	var m AppModel
	const w, h = 60, 6
	title := singleChip("[1]", true)

	box := m.panelBoxHint(true, title, listNavHint(true), w, h, "body")
	lines := strings.Split(box, "\n")
	for i, ln := range lines {
		if got := dispWidth(ln); got != w {
			t.Errorf("row %d dispWidth = %d, want %d", i, got, w)
		}
	}
	bottom := ansi.Strip(lines[len(lines)-1])
	if !strings.Contains(bottom, "enter into") {
		t.Errorf("bottom border should carry the hint, got %q", bottom)
	}

	plainBox := m.panelBoxHint(true, title, "", w, h, "body")
	plainBottom := ansi.Strip(strings.Split(plainBox, "\n")[h-1])
	if strings.Contains(plainBottom, "enter") {
		t.Errorf("no-hint box should have a clean bottom border, got %q", plainBottom)
	}
}
