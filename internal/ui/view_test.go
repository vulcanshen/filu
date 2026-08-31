package ui

import (
	"fmt"
	"os"
	"path/filepath"
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

// TestListRowsMatchesRender: the cursor's row budget must equal the file rows
// the app really puts on screen. It renders through middleView with View's own
// midH and counts them, rather than re-deriving a height — an earlier version
// measured a body drawn at listPanelHeight(), which was self-consistent and so
// stayed green while midHeight() quietly disagreed with View by two rows.
func TestListRowsMatchesRender(t *testing.T) {
	dir := t.TempDir()
	for i := range 80 { // comfortably more files than any test panel can show
		if err := os.WriteFile(filepath.Join(dir, fmt.Sprintf("f%02d.txt", i)), nil, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	for _, tc := range []struct {
		w, h int
		zoom panelID
	}{
		{120, 40, 0},         // wide grid
		{100, 24, 0},         // a height where the 2/3 split doesn't divide evenly
		{60, 30, 0},          // narrow fallback: the list takes the whole middle
		{120, 40, panelList}, // [1] zoomed: the list gets the whole panel region
	} {
		m := minModel()
		m.width, m.height, m.zoom = tc.w, tc.h, tc.zoom
		m.tabs = []listModel{{dir: dir}}
		m.tabs[0].reload()

		frame := ansi.Strip(m.middleView(m.width, m.height-1)) // exactly what View passes
		drawn := strings.Count(frame, ".txt")                  // one per rendered file row
		if drawn != m.listRows() {
			t.Errorf("%dx%d zoom=%d: %d file rows on screen but listRows says %d",
				tc.w, tc.h, tc.zoom, drawn, m.listRows())
		}
	}
}

// TestDetailRowsMatchesRender: the preview's scroll budget must equal the
// content rows panel [2] draws, both in the grid and zoomed.
func TestDetailRowsMatchesRender(t *testing.T) {
	lines := make([]string, 200)
	for i := range lines {
		lines[i] = fmt.Sprintf("line %d", i)
	}
	for _, tc := range []struct {
		w, h int
		zoom panelID
	}{
		{120, 40, 0},
		{100, 24, 0},
		{120, 40, panelDetail},
	} {
		m := minModel()
		m.width, m.height, m.zoom = tc.w, tc.h, tc.zoom
		m.tabs = []listModel{{dir: t.TempDir()}}
		m.preview = previewModel{kind: previewText, lines: lines, body: lines}

		frame := ansi.Strip(m.middleView(m.width, m.height-1))
		drawn := strings.Count(frame, "line ")
		if drawn != m.detailRows() {
			t.Errorf("%dx%d zoom=%d: %d preview rows on screen but detailRows says %d",
				tc.w, tc.h, tc.zoom, drawn, m.detailRows())
		}
	}
}
