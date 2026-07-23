package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestListViewTicksCarried checks that a file sitting in the carries bucket gets
// the pick glyph in panel [2], and that nothing is ticked when the bucket set is
// empty — this is the multi-select cue.
func TestListViewTicksCarried(t *testing.T) {
	dir := t.TempDir()
	for _, n := range []string{"a.txt", "b.txt"} {
		if err := os.WriteFile(filepath.Join(dir, n), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	m := newList(dir) // cursor defaults to the first item (a.txt)

	if out := m.view(40, 10, false, nil); strings.Contains(out, pickGlyph) {
		t.Error("no carried files → view must not show a tick")
	}

	carried := map[string]bool{filepath.Join(dir, "b.txt"): true}
	out := m.view(40, 10, false, carried)
	if !strings.Contains(out, pickGlyph) {
		t.Error("b.txt is carried → view should tick it")
	}
	if strings.Count(out, pickGlyph) != 1 {
		t.Errorf("exactly one file carried → want 1 tick, got %d", strings.Count(out, pickGlyph))
	}
}
