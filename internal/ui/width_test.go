package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestIsWideIcon(t *testing.T) {
	cases := []struct {
		r    rune
		want bool
		name string
	}{
		{0xf07b, true, "folder icon"},
		{0xf15b, true, "file icon"},
		{0xf450, true, "oct file-directory"},
		{0xf00c, true, "check"},
		{0xe0b0, false, "powerline hard cap"},
		{0xe0b6, false, "powerline round-left"},
		{0x2502, false, "box vertical"},
		{0x280b, false, "braille spinner frame"},
		{'a', false, "ascii"},
		{0x4e2d, false, "CJK 中 (already double-width, not a PUA icon)"},
	}
	for _, c := range cases {
		if got := isWideIcon(c.r); got != c.want {
			t.Errorf("isWideIcon(%s U+%X) = %v, want %v", c.name, c.r, got, c.want)
		}
	}
}

func TestDispWidthWideIcons(t *testing.T) {
	defer restoreIconCells(iconCells)
	icon := string(rune(0xf07b))

	iconCells = 1 // normal font: icon is 1 cell, no adjustment
	if got := dispWidth(" " + icon + " name"); got != 7 {
		t.Errorf("iconCells=1: dispWidth = %d, want 7", got)
	}
	iconCells = 2 // CJK icon font: the icon eats an extra cell
	if got := dispWidth(" " + icon + " name"); got != 8 {
		t.Errorf("iconCells=2: dispWidth = %d, want 8", got)
	}
}

func TestTruncateAndPadDispAreExactWidth(t *testing.T) {
	defer restoreIconCells(iconCells)
	iconCells = 2
	icon := string(rune(0xf07b))
	s := " " + icon + " a-fairly-long-directory-name"

	for w := 1; w <= 20; w++ {
		if got := dispWidth(truncate(s, w)); got > w {
			t.Errorf("truncate to %d: dispWidth = %d (must be <= %d)", w, got, w)
		}
		if got := dispWidth(padDisp(s, w)); got != w {
			t.Errorf("padDisp to %d: dispWidth = %d (must equal %d)", w, got, w)
		}
	}
}

func TestJoinHKeepsColumnsAligned(t *testing.T) {
	defer restoreIconCells(iconCells)
	iconCells = 2
	// two 1-row blocks: a icon-bearing left, a plain right.
	left := " " + string(rune(0xf07b)) + " x" // dispWidth 5
	right := "abc"                            // dispWidth 3
	got := dispWidth(joinH(padDisp(left, 5), padDisp(right, 3)))
	if got != 8 {
		t.Errorf("joinH total dispWidth = %d, want 8", got)
	}
}

// TestViewEveryLineIsTerminalWidth is the real proof: with icons rendering
// 2-wide (the CJK-font case that broke the borders), every line of the composed
// View must still be exactly the terminal width — otherwise a border is pushed.
func TestViewEveryLineIsTerminalWidth(t *testing.T) {
	defer restoreIconCells(iconCells)

	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "subdir"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "file.go"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	newModel := func() AppModel {
		m := AppModel{
			focus: panelList, places: newPlaces(), spaceMenu: newSpaceMenu(),
			confirm: newConfirmPopup(), inputPopup: newInputPopup(), help: newHelpPopup(),
			taskCh: make(chan landMsg, 8),
		}
		m.tabs = []listModel{newList(dir)}
		m.carry.items = []string{filepath.Join(dir, "file.go")}
		m.tasks = []landTask{{id: 1, action: "cp", dest: "d", total: 1, status: taskDone}}
		m.refreshPreview()
		m.width, m.height = 120, 30
		return m
	}

	for _, cells := range []int{1, 2} {
		iconCells = cells
		// exercise the normal grid plus each zoom mode and all four foci.
		zooms := []panelID{0, panelList, panelDetail, panelCarry}
		for _, z := range zooms {
			for _, f := range []panelID{panelPin, panelList, panelDetail, panelCarry} {
				m := newModel()
				m.zoom, m.focus = z, f
				for r, line := range strings.Split(m.View(), "\n") {
					if got := dispWidth(line); got != m.width {
						t.Errorf("iconCells=%d zoom=%d focus=%d: row %d dispWidth=%d, want %d\n  %q",
							cells, z, f, r, got, m.width, line)
					}
				}
			}
		}
	}
}

func restoreIconCells(v int) { iconCells = v }
