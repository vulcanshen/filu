package ui

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	terminu "github.com/vulcanshen/terminu-go"
)

// Enter goes into a directory; Esc goes back up, the cursor on the directory
// it came from (tdp K3, K4).
func TestEnterIntoAndEscUp(t *testing.T) {
	hs := newHarness(t, 120, 40, map[string]string{"a/inner.txt": "x", "b/": "", "c.txt": ""})
	hs.to("b")
	hs.keys("Enter")
	if got := hs.a.cur().dir; got != hs.path("b") {
		t.Fatalf("Enter on b: the tab is on %s", got)
	}
	hs.keys("Esc")
	if got := hs.a.cur().dir; got != hs.root {
		t.Fatalf("Esc: the tab is on %s", got)
	}
	if hs.cursor() != "b" {
		t.Fatalf("Esc: the cursor is on %q, not the directory it came from", hs.cursor())
	}
}

// Enter on a file shows its facts in a note; nothing is opened outside filu.
func TestEnterOnAFileShowsItsFacts(t *testing.T) {
	hs := newHarness(t, 120, 40, map[string]string{"notes.md": "hello"})
	hs.to("notes.md")
	hs.keys("Enter")
	hs.has("INFO — notes.md", "PATH", "TYPE", "Text", "PERMISSIONS", "OWNER")
	hs.hasNot("1 PATH") // a table of facts, without line numbers (tdp S66)
	hs.keys("Esc")
	hs.hasNot("INFO — notes.md")
}

// The sort glyphs of components/layout/panel, as the list's header draws them.
const (
	glyphSortAsc  = "\U0000F160" // nf-fa-sort_amount_asc
	glyphSortDesc = "\U0000F161" // nf-fa-sort_amount_desc
)

// tabLines is [1]'s list as the current tab draws it, without its styles.
func tabLines(hs *harness, w int) []string {
	var out []string
	for _, l := range hs.a.cur().list.Draw(w, 20, true) {
		out = append(out, terminu.StripStyles(l))
	}
	return out
}

// The list is directories first, then files; a sort keeps the directories
// first, the header carrying the sort's glyph and place (components/layout/panel).
// Choosing a direction goes back to the column menu, to sort by another column
// too (tdp T1).
func TestSortKeepsDirectoriesFirst(t *testing.T) {
	hs := newHarness(t, 120, 40, map[string]string{"big.txt": strings.Repeat("x", 3000), "small.txt": "x", "zdir/": "", "adir/": ""})
	names := func() []string {
		var out []string
		for _, l := range tabLines(hs, 100) {
			for _, n := range []string{"adir", "zdir", "big.txt", "small.txt"} {
				if strings.Contains(l, " "+n) {
					out = append(out, n)
				}
			}
		}
		return out
	}
	if got := names(); !slices.Equal(got, []string{"adir", "zdir", "big.txt", "small.txt"}) {
		t.Fatalf("default order: %v", got)
	}
	hs.keys("S", "s", "D") // size, descending
	if got := names(); !slices.Equal(got, []string{"adir", "zdir", "big.txt", "small.txt"}) {
		t.Fatalf("size descending: %v", got)
	}
	hs.has("SORT", "1st, descending")
	hs.keys("n", "D", "Esc") // then the name, descending
	if got := names(); !slices.Equal(got, []string{"zdir", "adir", "big.txt", "small.txt"}) {
		t.Fatalf("size then name descending: %v", got)
	}
	hs.has("SIZE (1) "+glyphSortDesc, "NAME (2) "+glyphSortDesc)
	hs.hasNot("1st, descending")
	hs.keys("S", "r")
	if got := names(); !slices.Equal(got, []string{"adir", "zdir", "big.txt", "small.txt"}) {
		t.Fatalf("after Reset: %v", got)
	}
	hs.hasNot(glyphSortDesc)
}

// As the panel narrows the columns go whole and in order — the owner, the
// size, the modified time, the permissions — and the name stays.
func TestColumnsLeaveInOrder(t *testing.T) {
	hs := newHarness(t, 120, 40, map[string]string{"notes.md": "hello"})
	steps := [][]string{
		{"MODIFIED", "OWNER", "PERMS", "SIZE", "NAME"},
		{"MODIFIED", "PERMS", "SIZE", "NAME"},
		{"MODIFIED", "PERMS", "NAME"},
		{"PERMS", "NAME"},
		{"NAME"},
	}
	at := 0
	for w := 120; w >= 16; w-- {
		header := tabLines(hs, w)[1]
		var shown []string
		for _, title := range steps[0] {
			if strings.Contains(header, title) {
				shown = append(shown, title)
			}
		}
		for at < len(steps) && !slices.Equal(shown, steps[at]) {
			at++
		}
		if at == len(steps) {
			t.Fatalf("at %d cells the header is %q", w, header)
		}
	}
	if at != len(steps)-1 {
		t.Fatalf("at 16 cells the columns are %v", steps[at])
	}
}

// The list's first row is where the tab is, a star after it when the
// directory is a favorite (components/layout/panel, location row).
func TestLocationRow(t *testing.T) {
	hs := newHarness(t, 120, 40, map[string]string{"proj/": ""})
	if got := tabLines(hs, 60)[0]; !strings.Contains(got, "~/work") || strings.Contains(got, iconPin) {
		t.Fatalf("the location row: %q", got)
	}
	hs.keys("F")
	if got := tabLines(hs, 60)[0]; !strings.Contains(got, "~/work "+iconPin) {
		t.Fatalf("a favorite's location row: %q", got)
	}
}

// m marks a file into the marks, which [3] lists; c copies the marks into
// the directory [1] is on, v moves them (and they leave the marks).
func TestMarkCopyMove(t *testing.T) {
	hs := newHarness(t, 120, 40, map[string]string{"a.txt": "A", "b.txt": "B", "dest/": ""})
	hs.to("a.txt")
	hs.keys("m")
	if !slices.Contains(hs.a.cur().list.Marked(), hs.path("a.txt")) {
		t.Fatal("m did not mark a.txt in the list")
	}
	hs.has("[3] MARKS", "a.txt")
	hs.to("dest")
	hs.keys("Enter", "c")
	hs.drain()
	if _, err := os.Stat(hs.path("dest/a.txt")); err != nil {
		t.Fatalf("c did not copy a.txt: %v", err)
	}
	if len(hs.a.marks.items) != 1 {
		t.Fatalf("a copy keeps the marks: %v", hs.a.marks.items)
	}
	hs.a.cd(hs.root, "") // Esc would close the toast first (tdp K4)
	hs.to("b.txt")
	hs.keys("m")
	hs.to("dest")
	hs.keys("Enter", "v")
	hs.drain()
	if hs.exists("b.txt") || !hs.exists("dest/b.txt") {
		t.Fatal("v did not move b.txt")
	}
	if slices.Contains(hs.a.marks.items, hs.path("b.txt")) {
		t.Fatal("a moved file stays in the marks")
	}
	if !strings.Contains(hs.screen(), "Moved") && len(hs.a.tasks) != 2 {
		t.Fatalf("the tasks: %v", hs.a.tasks)
	}
}

// Copy here and Move here cannot run with nothing marked (tdp M6).
func TestLandDisabledWithoutMarks(t *testing.T) {
	hs := newHarness(t, 120, 40, map[string]string{"a.txt": ""})
	hs.keys("c")
	if len(hs.a.tasks) != 0 {
		t.Fatal("c ran with nothing marked")
	}
}

// . shows the dotfiles, and hides them again.
func TestHiddenFiles(t *testing.T) {
	hs := newHarness(t, 120, 40, map[string]string{".env": "", "a.txt": ""})
	hs.hasNot(".env")
	hs.keys(".")
	hs.has(".env")
	hs.keys(".")
	hs.hasNot(".env")
}

// An empty directory says so and names the keys (components/layout/screen);
// one with only dotfiles says how to show them.
func TestEmptyDirectory(t *testing.T) {
	hs := newHarness(t, 120, 40, map[string]string{"e/": "", "h/.x": ""})
	hs.to("e")
	hs.keys("Enter")
	hs.has("Nothing here", "Press [a] to add a file")
	hs.keys("Esc")
	hs.to("h")
	hs.keys("Enter")
	hs.has("Nothing here but 1 hidden", "Press [.] to show hidden files")
}

// A directory that cannot be read says why, in Red, under its path.
func TestUnreadableDirectory(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("root reads everything")
	}
	hs := newHarness(t, 120, 40, map[string]string{"locked/a": ""})
	if err := os.Chmod(hs.path("locked"), 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(hs.path("locked"), 0o755) })
	hs.to("locked")
	hs.keys("Enter")
	hs.has("Cannot read this directory: permission denied", "Press [Esc] to go up")
}

// t opens a tab (here on the same directory), h|l switch tabs, w closes one;
// the tabs come back next time, the first on where filu starts.
func TestTabs(t *testing.T) {
	hs := newHarness(t, 120, 40, map[string]string{"sub/x": ""})
	hs.to("sub")
	hs.keys("t", "s")
	if len(hs.a.tabs) != 2 || hs.a.active != 1 {
		t.Fatalf("t then Same directory: %d tabs, on %d", len(hs.a.tabs), hs.a.active)
	}
	hs.keys("Enter") // the second tab into sub
	hs.keys("h")
	if hs.a.active != 0 || hs.a.cur().dir != hs.root {
		t.Fatalf("h: on tab %d at %s", hs.a.active, hs.a.cur().dir)
	}
	hs.restart()
	if len(hs.a.tabs) != 2 || hs.a.tabs[1].dir != hs.path("sub") {
		t.Fatalf("after a restart the tabs are %d, the second on %s", len(hs.a.tabs), hs.a.tabs[1].dir)
	}
	hs.keys("l", "w")
	if len(hs.a.tabs) != 1 {
		t.Fatalf("w left %d tabs", len(hs.a.tabs))
	}
	hs.keys("w") // the last tab stays (tdp M6: Close tab cannot run)
	if len(hs.a.tabs) != 1 {
		t.Fatal("w closed the last tab")
	}
}

// At five tabs a new tab cannot be opened (tdp M6).
func TestAtMostFiveTabs(t *testing.T) {
	hs := newHarness(t, 120, 40, nil)
	for range 6 {
		hs.keys("t", "s")
	}
	if len(hs.a.tabs) != maxTabs {
		t.Fatalf("%d tabs", len(hs.a.tabs))
	}
}

// r renames in place; a name already there, or with a /, is refused on the
// spot and the popup stays (tdp K3).
func TestRename(t *testing.T) {
	hs := newHarness(t, 120, 40, map[string]string{"old.txt": "", "taken.txt": ""})
	hs.to("old.txt")
	hs.keys("r", "taken.txt", "Enter")
	hs.has("taken.txt already exists here")
	for range len("taken.txt") {
		hs.keys("Backspace")
	}
	hs.keys("a/b", "Enter")
	hs.has("A name can't contain /")
	for range 3 {
		hs.keys("Backspace")
	}
	hs.keys("new.txt", "Enter")
	if hs.exists("old.txt") || !hs.exists("new.txt") {
		t.Fatal("old.txt was not renamed to new.txt")
	}
	if hs.cursor() != "new.txt" {
		t.Fatalf("the cursor is on %q, not the renamed file", hs.cursor())
	}
}

// A rename that fails says so over the input, which keeps the name typed
// (tdp F4).
func TestRenameThatFails(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root renames in a read-only directory too")
	}
	hs := newHarness(t, 120, 40, map[string]string{"old.txt": ""})
	hs.to("old.txt")
	if err := os.Chmod(hs.root, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(hs.root, 0o755) })
	hs.keys("r", "new.txt", "Enter")
	hs.has("RENAME FAILED: old.txt")
	hs.keys("Esc")
	hs.has("RENAME: old.txt", "new.txt")
	if !hs.exists("old.txt") {
		t.Fatal("old.txt is gone")
	}
}

// a adds a file, or a directory when the name ends with /; a name outside the
// directory is refused.
func TestAdd(t *testing.T) {
	hs := newHarness(t, 120, 40, nil)
	hs.keys("a", "../escape", "Enter")
	hs.has("../escape is outside this directory")
	hs.keys("Esc", "a", "made/", "Enter")
	if fi, err := os.Stat(hs.path("made")); err != nil || !fi.IsDir() {
		t.Fatalf("made/ is not a directory: %v", err)
	}
	hs.keys("a", "deep/file.txt", "Enter")
	if !hs.exists("deep/file.txt") {
		t.Fatal("deep/file.txt was not made")
	}
	if hs.cursor() != "deep" {
		t.Fatalf("the cursor is on %q", hs.cursor())
	}
}

// D asks, then moves the file to the trash (tdp F6).
func TestDeleteAsksFirst(t *testing.T) {
	hs := newHarness(t, 120, 40, map[string]string{"gone.txt": "x"})
	hs.to("gone.txt")
	hs.keys("D")
	hs.has("Move gone.txt to the trash?")
	hs.keys("Esc")
	if !hs.exists("gone.txt") {
		t.Fatal("Esc deleted it")
	}
	hs.keys("D", "Enter")
	if hs.exists("gone.txt") {
		t.Fatal("Enter did not delete it")
	}
	if _, err := os.Stat(filepath.Join(trashDir(), "gone.txt")); err != nil {
		t.Fatalf("it is not in the trash: %v", err)
	}
}

// f stars a directory (not a file: the row is not there), F stars the one
// [1] is on; [3] Favorites lists them, the list and the path row show a star.
func TestFavorites(t *testing.T) {
	hs := newHarness(t, 120, 40, map[string]string{"proj/": "", "a.txt": ""})
	hs.to("a.txt")
	hs.keys("f")
	if len(hs.a.favs.pinned) != 0 {
		t.Fatal("f starred a file")
	}
	hs.to("proj")
	hs.keys("f")
	if !hs.a.favs.pinnedSet()[hs.path("proj")] {
		t.Fatal("f did not star proj")
	}
	hs.has("proj " + iconPin)
	hs.keys("F")
	if !hs.a.favs.pinnedSet()[hs.root] {
		t.Fatal("F did not star the directory [1] is on")
	}
	hs.keys("3", "l", "l")
	hs.has("[3]", "FAVORITES", "proj", "work")
}

// b lists the directories this one is in; choosing one goes there.
func TestBreadcrumb(t *testing.T) {
	hs := newHarness(t, 120, 40, map[string]string{"a/b/c/": ""})
	hs.a.cd(hs.path("a/b/c"), "")
	hs.keys("b")
	hs.has("BREADCRUMB", "one level up", "2 levels up")
	hs.keys("j", "Enter") // the second row: two levels up
	if got := hs.a.cur().dir; got != hs.path("a") {
		t.Fatalf("the tab is on %s", got)
	}
}

// y copies the full path of the entry (OSC 52).
func TestYankPath(t *testing.T) {
	hs := newHarness(t, 120, 40, map[string]string{"a.txt": ""})
	hs.to("a.txt")
	hs.keys("y")
	if got := hs.copied(); got != hs.path("a.txt") {
		t.Fatalf("copied %q", got)
	}
	hs.has("Copied")
}

// z zooms the focused panel to the whole screen and back; zoomed, [1] shows
// every tab side by side.
func TestZoom(t *testing.T) {
	hs := newHarness(t, 160, 40, map[string]string{"sub/x.txt": ""})
	hs.keys("t", "s", "z")
	if !hs.a.t.ForceNarrow {
		t.Fatal("z did not zoom")
	}
	hs.hasNot("[2] PREVIEW")
	if n := strings.Count(hs.screen(), "/work"); n < 2 {
		t.Fatalf("zoomed, the two tabs are not side by side:\n%s", hs.screen())
	}
	hs.keys("z")
	if hs.a.t.ForceNarrow {
		t.Fatal("z again did not give the screen back")
	}
	hs.has("[2] PREVIEW")
}

// The first tab's mark is the launch glyph, then the animals (components:
// the label of a panel with tabs is its first tab's name).
func TestTabMarks(t *testing.T) {
	hs := newHarness(t, 120, 40, nil)
	hs.keys("t", "s")
	top := terminu.StripStyles(hs.a.t.Frame()[0])
	if !strings.Contains(top, "[1]") || !strings.Contains(top, tabMark(0)) || !strings.Contains(top, tabMark(1)) {
		t.Fatalf("the tab chain: %q", top)
	}
}
