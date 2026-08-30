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

// TestMarkCellStates: the single mark cell shows one glyph per state (mark /
// favorite / both), blank when neither, and every state is the same width.
func TestMarkCellStates(t *testing.T) {
	cases := []struct {
		carried, pinned bool
		want            string
	}{
		{false, false, ""},         // blank
		{true, false, markGlyph},   // marked
		{false, true, iconPin},     // favorited
		{true, true, markFavGlyph}, // both → the combined glyph
	}
	for _, c := range cases {
		if got := strings.TrimRight(ansi.Strip(markCell(c.carried, c.pinned, false)), " "); got != c.want {
			t.Errorf("markCell(carried=%v,pinned=%v) = %q, want %q", c.carried, c.pinned, got, c.want)
		}
	}
	// all four states occupy the same display width, so nothing shifts on toggle.
	w := dispWidth(markCell(false, false, true))
	for _, c := range cases[1:] {
		if got := dispWidth(markCell(c.carried, c.pinned, true)); got != w {
			t.Errorf("mark cell width for (%v,%v) = %d, want %d", c.carried, c.pinned, got, w)
		}
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
	if got := fmtSize(fileItem{isDir: true, size: 4096}); got != "-" {
		t.Errorf("a directory's size should be a dash placeholder, got %q", got)
	}

	// colorSize: a dash for a dir, and warmer buckets differ across magnitudes.
	if got := ansi.Strip(colorSize(fileItem{isDir: true, size: 1 << 30})); got != "-" {
		t.Errorf("colorSize of a directory should be a dash, got %q", got)
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

// TestListHeaderStarsFavoriteDir: when the browsed directory is itself a
// favorite, the header's mark column shows the star (flagging the whole dir);
// otherwise the header carries no star.
func TestListHeaderStarsFavoriteDir(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	m := newList(dir)

	favHeader := strings.SplitN(ansi.Strip(m.view(76, 8, true, nil, map[string]bool{dir: true})), "\n", 2)[0]
	if !strings.Contains(favHeader, iconPin) {
		t.Errorf("a favorite dir should star the header mark column: %q", favHeader)
	}
	plainHeader := strings.SplitN(ansi.Strip(m.view(76, 8, true, nil, nil)), "\n", 2)[0]
	if strings.Contains(plainHeader, iconPin) {
		t.Errorf("a non-favorite dir header should carry no star: %q", plainHeader)
	}
}

// TestReadEntriesHiddenCount: dotfiles are excluded from the listing when
// showHidden is off but still counted; showing them keeps the count.
func TestReadEntriesHiddenCount(t *testing.T) {
	dir := t.TempDir()
	for _, n := range []string{"a", "b", ".dot1", ".dot2"} {
		mustWrite(t, filepath.Join(dir, n))
	}
	// showHidden=false: dotfiles are excluded from items but still counted.
	vis, hidden, err := readEntries(dir, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(vis) != 2 || hidden != 2 {
		t.Errorf("showHidden=false: got %d items, %d hidden; want 2, 2", len(vis), hidden)
	}
	// showHidden=true: dotfiles included, count unchanged.
	all, hidden2, _ := readEntries(dir, true)
	if len(all) != 4 || hidden2 != 2 {
		t.Errorf("showHidden=true: got %d items, %d hidden; want 4, 2", len(all), hidden2)
	}
}
