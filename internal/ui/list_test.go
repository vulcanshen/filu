package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

// TestListViewTicksCarried checks that a file sitting in the marks bucket gets
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

	if out := m.view(40, 10, false, nil, nil); strings.Contains(out, markGlyph) {
		t.Error("no carried files → view must not show a tick")
	}

	carried := map[string]bool{filepath.Join(dir, "b.txt"): true}
	out := m.view(40, 10, false, carried, nil)
	if !strings.Contains(out, markGlyph) {
		t.Error("b.txt is carried → view should tick it")
	}
	if strings.Count(out, markGlyph) != 1 {
		t.Errorf("exactly one file carried → want 1 tick, got %d", strings.Count(out, markGlyph))
	}
}

// TestListSizeColumn: a file shows its compact size, a directory is blank (filu
// never recurses to size a directory), and the size colour tracks magnitude.
func TestListSizeColumn(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "adir"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "big.bin"), make([]byte, 4096), 0o644); err != nil {
		t.Fatal(err)
	}
	out := ansi.Strip(newList(dir).view(90, 8, true, nil, nil))
	if !strings.Contains(out, "Size") {
		t.Errorf("wide view should show the Size header:\n%s", out)
	}
	if !strings.Contains(out, "4.0K") { // 4096 bytes
		t.Errorf("big.bin (4096 B) should show 4.0K:\n%s", out)
	}

	// fmtSize: a file has a size, a directory is blank.
	if got := fmtSize(fileItem{size: 4096}); got != "4.0K" {
		t.Errorf("fmtSize(4096) = %q, want 4.0K", got)
	}
	if got := fmtSize(fileItem{isDir: true, size: 4096}); got != "" {
		t.Errorf("a directory's size must be blank, got %q", got)
	}

	// colorSize: blank for a dir, and warmer buckets differ across magnitudes.
	if colorSize(fileItem{isDir: true, size: 1 << 30}) != "" {
		t.Error("colorSize of a directory should be blank")
	}
	small := colorSize(fileItem{size: 500})   // < 1 MiB → green
	big := colorSize(fileItem{size: 5 << 30}) // > 1 GiB → peach
	if small == big {
		t.Error("size colour should differ across magnitude buckets")
	}
	if !strings.Contains(ansi.Strip(small), "500") {
		t.Errorf("colorSize should render the compact size, got %q", ansi.Strip(small))
	}
}

// TestListColumns checks the multi-column rows: the wide layout shows the
// Modified / Owner / Perms / Size / Name headers, a mode string and the pin glyph;
// the columns drop in the order owner → size → mtime → perms as the panel narrows,
// leaving Name last.
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

	// Wide: every column + header labels + a mode string, and the pin glyph shows.
	wide := m.view(76, 8, true, nil, pinned)
	for _, want := range []string{"Owner", "Modified", "Perms", "Size", "Name", "drwxr-xr-x"} {
		if !strings.Contains(ansi.Strip(wide), want) {
			t.Errorf("wide view missing %q:\n%s", want, ansi.Strip(wide))
		}
	}
	if !strings.Contains(wide, iconPin) {
		t.Error("pinned dir should show the pin glyph")
	}

	// Owner drops first, before perms.
	noOwner := ansi.Strip(m.view(56, 8, true, nil, nil))
	if strings.Contains(noOwner, "Owner") {
		t.Errorf("Owner should drop first:\n%s", noOwner)
	}
	if !strings.Contains(noOwner, "Perms") {
		t.Errorf("Perms should survive when only Owner drops:\n%s", noOwner)
	}

	// Narrower: Modified drops before Perms (perms is kept longest) — at w=40 the
	// mode string still shows but Modified is gone.
	narrow := ansi.Strip(m.view(40, 8, true, nil, nil))
	if !strings.Contains(narrow, "drwx") {
		t.Errorf("Perms should survive at w=40 (dropped last):\n%s", narrow)
	}
	if strings.Contains(narrow, "Modified") {
		t.Errorf("Modified should drop before Perms at w=40:\n%s", narrow)
	}

	// Very narrow: only Name survives (all metadata gone).
	tiny := ansi.Strip(m.view(24, 8, true, nil, nil))
	if strings.Contains(tiny, "Modified") || strings.Contains(tiny, "drwx") {
		t.Errorf("only Name should survive at w=24:\n%s", tiny)
	}
}
