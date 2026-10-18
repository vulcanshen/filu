package ui

import (
	"archive/zip"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// In [3] Marks, p picks some of the marks: only the picks land; m takes one
// out of the marks; C asks, then clears them all.
func TestMarksPickUnmarkClear(t *testing.T) {
	hs := newHarness(t, 120, 40, map[string]string{"a.txt": "A", "b.txt": "B", "c.txt": "C", "dest/": ""})
	for _, n := range []string{"a.txt", "b.txt", "c.txt"} {
		hs.to(n)
		hs.keys("m")
	}
	hs.keys("3")
	hs.a.marksL.Select(hs.path("b.txt"))
	hs.keys("p")
	if got := hs.a.marks.landItems(); !slices.Equal(got, []string{hs.path("b.txt")}) {
		t.Fatalf("the picks: %v", got)
	}
	hs.keys("1")
	hs.to("dest")
	hs.keys("Enter", "c")
	hs.drain()
	if !hs.exists("dest/b.txt") || hs.exists("dest/a.txt") {
		t.Fatal("c landed more than the pick")
	}
	hs.keys("3")
	hs.a.marksL.Select(hs.path("a.txt"))
	hs.keys("m")
	if slices.Contains(hs.a.marks.items, hs.path("a.txt")) {
		t.Fatal("m in Marks did not unmark a.txt")
	}
	if slices.Contains(hs.a.tabs[0].list.Marked(), hs.path("a.txt")) {
		t.Fatal("[1] still shows a.txt marked")
	}
	hs.keys("C")
	hs.has("Clear all 2 marks?")
	hs.keys("Enter")
	if len(hs.a.marks.items) != 0 {
		t.Fatalf("C left %v", hs.a.marks.items)
	}
}

// Enter on a mark shows it in [1]: the tab on its directory, or a new one;
// the focus stays in [3] (Rules K3) and a toast says where to look.
func TestMarkShowInTabs(t *testing.T) {
	hs := newHarness(t, 120, 40, map[string]string{"sub/deep.txt": "", "top.txt": ""})
	hs.a.cd(hs.path("sub"), "")
	hs.to("deep.txt")
	hs.keys("m")
	hs.a.cd(hs.root, "")
	hs.keys("3", "Enter")
	if len(hs.a.tabs) != 2 || hs.a.cur().dir != hs.path("sub") || hs.cursor() != "deep.txt" {
		t.Fatalf("Enter on the mark: %d tabs, on %s, cursor on %q", len(hs.a.tabs), hs.a.cur().dir, hs.cursor())
	}
	if hs.a.screen.Focused() != hs.a.drawer {
		t.Fatal("Enter moved the focus")
	}
	hs.has("In [1]")
}

// Z packs the picks into a zip that becomes the only pick, to land with c.
func TestZip(t *testing.T) {
	hs := newHarness(t, 120, 40, map[string]string{"a.txt": "A", "dir/b.txt": "B", "dest/": ""})
	hs.to("a.txt")
	hs.keys("m")
	hs.to("dir")
	hs.keys("m", "3", "Z")
	hs.has("Zip 2 items")
	hs.keys("pack", "Enter")
	hs.drain()
	if len(hs.a.marks.picked) != 1 {
		t.Fatalf("the picks after a zip: %v", hs.a.marks.picked)
	}
	var zipPath string
	for p := range hs.a.marks.picked {
		zipPath = p
	}
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	var names []string
	for _, f := range r.File {
		names = append(names, f.Name)
	}
	if !slices.Contains(names, "a.txt") || !slices.Contains(names, "dir/b.txt") {
		t.Fatalf("the zip holds %v", names)
	}
}

// Tasks: Enter takes [1] to where it landed; D takes it out of the log.
func TestTasks(t *testing.T) {
	hs := newHarness(t, 120, 40, map[string]string{"a.txt": "A", "dest/": ""})
	hs.to("a.txt")
	hs.keys("m")
	hs.to("dest")
	hs.keys("Enter", "c")
	hs.drain()
	hs.a.cd(hs.root, "")
	hs.keys("3", "l")
	hs.has("Copied a.txt → dest")
	hs.keys("Enter")
	if hs.a.cur().dir != hs.path("dest") {
		t.Fatalf("Enter on the task: [1] is on %s", hs.a.cur().dir)
	}
	hs.keys("D")
	if len(hs.a.tasks) != 0 {
		t.Fatal("D did not take the task out of the log")
	}
}

// Favorites: Enter goes there, o chooses the tab, D asks and takes the star
// off.
func TestFavoritesTab(t *testing.T) {
	hs := newHarness(t, 120, 40, map[string]string{"proj/": ""})
	hs.to("proj")
	hs.keys("f", "3", "l", "l", "Enter")
	if hs.a.cur().dir != hs.path("proj") {
		t.Fatalf("Enter on the favorite: [1] is on %s", hs.a.cur().dir)
	}
	hs.keys("o")
	hs.has("OPEN IN", "New tab", "already there")
	hs.keys("Esc", "Esc", "D") // the first Esc takes the toast (Rules K4)
	hs.has("Take the star off")
	hs.keys("Enter")
	if len(hs.a.favs.pinned) != 0 {
		t.Fatal("D did not take the star off")
	}
}

// The marks, the favorites, the tasks and each directory's sort come back
// next time.
func TestSessionComesBack(t *testing.T) {
	hs := newHarness(t, 120, 40, map[string]string{"a.txt": "", "proj/": ""})
	hs.to("a.txt")
	hs.keys("m")
	hs.to("proj")
	hs.keys("f", "S", "n", "D")
	hs.restart()
	if !slices.Equal(hs.a.marks.items, []string{hs.path("a.txt")}) {
		t.Fatalf("the marks: %v", hs.a.marks.items)
	}
	if !hs.a.favs.pinnedSet()[hs.path("proj")] {
		t.Fatal("the favorite is gone")
	}
	if r := sortRulesFor(hs.root); len(r) != 1 || r[0].asc {
		t.Fatalf("the sort: %v", r)
	}
	if !slices.Contains(hs.a.cur().list.Marked(), hs.path("a.txt")) {
		t.Fatal("[1] does not show the mark after a restart")
	}
}

// Quitting asks, says where the shell goes, and tells the shell function
// (cd-on-quit); Quit to chooses another directory.
func TestQuit(t *testing.T) {
	hs := newHarness(t, 120, 40, map[string]string{"sub/": ""})
	last := filepath.Join(hs.home, "lastdir")
	t.Setenv(envLastDirFile, last)
	hs.a.syncQuit()
	hs.to("sub")
	hs.keys("Enter", "q")
	hs.has("Quit filu?", "Your shell goes to ~/work/sub.")
	hs.keys("Enter")
	if !hs.quit {
		t.Fatal("Enter on the quit confirm did not quit")
	}
	hs.a.shutdown()
	if b, _ := os.ReadFile(last); string(b) != hs.path("sub") {
		t.Fatalf("the shell is told %q", b)
	}

	hs = newHarness(t, 120, 40, map[string]string{"sub/": ""})
	t.Setenv(envLastDirFile, last)
	hs.to("sub")
	hs.keys("Enter", "Q")
	hs.has("QUIT TO", "where filu started")
	hs.keys("1")
	if !hs.quit {
		t.Fatal("choosing a directory did not quit")
	}
	hs.a.shutdown()
	if b, _ := os.ReadFile(last); string(b) != hs.root {
		t.Fatalf("the shell is told %q", b)
	}
}

// A running task makes the quit confirm warn.
func TestQuitWarnsOfARunningTask(t *testing.T) {
	hs := newHarness(t, 120, 40, nil)
	hs.a.addTask("cp", "x", hs.root, []string{"a"})
	hs.keys("q")
	hs.has("A task is still running")
}

// The search finder lists what is under the directory; choosing a file
// takes [1] to it.
func TestNameFinder(t *testing.T) {
	hs := newHarness(t, 120, 40, map[string]string{"deep/er/needle.txt": "", "hay.txt": ""})
	hs.keys("/", "f")
	hs.drain()
	hs.keys("needle")
	hs.has("deep/er/needle.txt")
	hs.keys("Enter", "Enter")
	if hs.a.cur().dir != hs.path("deep/er") || hs.cursor() != "needle.txt" {
		t.Fatalf("[1] is on %s, the cursor on %q", hs.a.cur().dir, hs.cursor())
	}
}

// Goto lists only directories, hidden ones too, under the home.
func TestGotoFinder(t *testing.T) {
	hs := newHarness(t, 120, 40, map[string]string{"proj/.hidden/": "", "file.txt": ""})
	hs.keys("g", "o", "/")
	hs.drain()
	hs.keys("hidden")
	hs.has("work/proj/.hidden")
	hs.keys("Enter", "Enter")
	if hs.a.cur().dir != hs.path("proj/.hidden") {
		t.Fatalf("[1] is on %s", hs.a.cur().dir)
	}
}

// Content search needs ripgrep; without it, it says so (tdp F5).
func TestContentSearchWithoutRipgrep(t *testing.T) {
	hs := newHarness(t, 120, 40, nil)
	old := lookPath
	lookPath = func(string) (string, error) { return "", os.ErrNotExist }
	defer func() { lookPath = old }()
	hs.keys("/", "c")
	hs.has("ripgrep (rg) is not installed")
}

// Content search asks for the text, then lists the files that hold it.
func TestContentSearch(t *testing.T) {
	if _, err := lookPath("rg"); err != nil {
		t.Skip("ripgrep is not installed")
	}
	hs := newHarness(t, 120, 40, map[string]string{"a.txt": "nothing\n", "b.txt": "one\ntwo needle\n"})
	hs.keys("/", "c", "needle", "Enter")
	hs.drain()
	hs.has("b.txt")
	hs.hasNot("a.txt   ")
	if hs.a.hits[hs.path("b.txt")] != 2 {
		t.Fatalf("the hit is on line %d", hs.a.hits[hs.path("b.txt")])
	}
}

// [2] previews the entry under [1]'s cursor; Enter views it, y copies all of
// it.
func TestPreview(t *testing.T) {
	hs := newHarness(t, 120, 40, map[string]string{"notes.md": "# title\nbody line\n"})
	hs.to("notes.md")
	hs.has("body line")
	hs.keys("2", "y")
	if got := hs.copied(); !strings.Contains(got, "body line") || strings.Contains(got, "\x1b") {
		t.Fatalf("copied %q", got)
	}
	hs.keys("Enter")
	hs.has("PREVIEW — notes.md", "v:visual")
}

// V opens the splash; any key closes it (tdp S).
func TestSplash(t *testing.T) {
	hs := newHarness(t, 120, 40, nil)
	hs.keys("V")
	hs.send(nil)
	for range 40 {
		hs.run(nil)
	}
	hs.keys("q") // only closes it (Rules S3)
	if hs.quit {
		t.Fatal("q on the splash quit")
	}
	hs.has("[1]")
}

// `filu <dotfile>` shows the hidden files and puts the cursor on it.
func TestStartOnADotfile(t *testing.T) {
	hs := newHarness(t, 120, 40, map[string]string{".env": "", "a.txt": ""})
	a := NewApp(hs.root, ".env")
	if it, _, _ := a.item(); it.name != ".env" || !a.cur().showHidden {
		t.Fatalf("the cursor is on %q", it.name)
	}
}
