package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
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

	if out := m.view(40, 10, false, nil, nil); strings.Contains(out, pickGlyph) {
		t.Error("no carried files → view must not show a tick")
	}

	carried := map[string]bool{filepath.Join(dir, "b.txt"): true}
	out := m.view(40, 10, false, carried, nil)
	if !strings.Contains(out, pickGlyph) {
		t.Error("b.txt is carried → view should tick it")
	}
	if strings.Count(out, pickGlyph) != 1 {
		t.Errorf("exactly one file carried → want 1 tick, got %d", strings.Count(out, pickGlyph))
	}
}

// TestListColumns checks the multi-column rows: the wide layout shows the
// Modified / Perms / Name headers, a mode string and the pin glyph; the columns
// drop in the order perms → mtime as the panel narrows, leaving Name last.
func TestListColumns(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "f.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	m := newList(dir)
	pinned := map[string]bool{filepath.Join(dir, "sub"): true}

	// Wide: all columns + header labels + a mode string, and the pin glyph shows.
	wide := m.view(60, 8, true, nil, pinned)
	for _, want := range []string{"Modified", "Perms", "Name", "drwxr-xr-x"} {
		if !strings.Contains(ansi.Strip(wide), want) {
			t.Errorf("wide view missing %q:\n%s", want, ansi.Strip(wide))
		}
	}
	if !strings.Contains(wide, iconPin) {
		t.Error("pinned dir should show the pin glyph")
	}

	// Narrow: perms drop first (no mode string), Modified survives.
	narrow := ansi.Strip(m.view(40, 8, true, nil, nil))
	if strings.Contains(narrow, "drwx") {
		t.Errorf("perms should drop at w=40:\n%s", narrow)
	}
	if !strings.Contains(narrow, "Modified") {
		t.Errorf("Modified should survive at w=40:\n%s", narrow)
	}

	// Very narrow: only Name survives.
	tiny := ansi.Strip(m.view(24, 8, true, nil, nil))
	if strings.Contains(tiny, "Modified") || strings.Contains(tiny, "drwx") {
		t.Errorf("only Name should survive at w=24:\n%s", tiny)
	}
}
