package ui

import (
	"archive/zip"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
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
	if got := hs.a.marksL.Marked(); !slices.Equal(got, []string{hs.path("b.txt")}) {
		t.Fatalf("[3] Marks ticks %v", got)
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
// the focus goes there, as the user going there expects (tdp K3).
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
	if hs.a.screen.Focused() != hs.a.files {
		t.Fatal("Enter left the focus in [3]")
	}
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

// Zip offers the name filu would give the zip in grey; Enter with nothing
// typed takes it (tdp S6).
func TestZipSuggestsAName(t *testing.T) {
	hs := newHarness(t, 120, 40, map[string]string{"a.txt": "A", "b.txt": "B"})
	for _, n := range []string{"a.txt", "b.txt"} {
		hs.to(n)
		hs.keys("m")
	}
	hs.keys("3", "Z")
	hs.has("work.zip")
	hs.keys("Enter")
	hs.drain()
	if len(hs.a.tasks) != 1 || hs.a.tasks[0].dest != "work.zip" {
		t.Fatalf("the tasks: %+v", hs.a.tasks)
	}
}

// A running task's row has the turning loading icon: filu keeps the frames
// coming while the Tasks tab shows one.
func TestRunningTaskSpins(t *testing.T) {
	hs := newHarness(t, 120, 40, nil)
	hs.a.addTask("cp", "x", hs.root, []string{"a"})
	if hs.a.t.Spinning() {
		t.Fatal("spinning with the Tasks tab hidden")
	}
	hs.keys("3", "l")
	if !hs.a.t.Spinning() {
		t.Fatal("not spinning with a running task shown")
	}
	hs.a.tasks[0].status = taskDone
	if hs.a.t.Spinning() {
		t.Fatal("spinning with no task running")
	}
}

// Tasks: Enter takes [1] to where it landed, and the focus with it; D takes
// it out of the log. The toast says the row's words, its glyph left to the
// row: the App Log entry starts with a capital (tdp note-78).
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
	if hs.a.cur().dir != hs.path("dest") || hs.a.screen.Focused() != hs.a.files {
		t.Fatalf("Enter on the task: [1] is on %s", hs.a.cur().dir)
	}
	hs.keys("3", "D")
	if len(hs.a.tasks) != 0 {
		t.Fatal("D did not take the task out of the log")
	}
	hs.keys("!")
	hs.has("OK   Copied a.txt → dest")
}

// A task that fails is one Error toast, with the row's words: in the App Log
// once, a capital first (tdp note-78), and a popup after it still opens.
func TestTaskFails(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("root writes everywhere")
	}
	hs := newHarness(t, 120, 40, map[string]string{"a.txt": "A", "shut/": ""})
	if err := os.Chmod(hs.path("shut"), 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(hs.path("shut"), 0o755) })
	hs.to("a.txt")
	hs.keys("m")
	hs.to("shut")
	hs.keys("Enter", "c")
	hs.drain()
	hs.keys("!")
	hs.has("ERR  Copy a.txt → shut")
	hs.hasNot("OK   Copy a.txt")
}

// Favorites: Enter goes there, and the focus with it; o chooses the tab, D
// asks and takes the star off.
func TestFavoritesTab(t *testing.T) {
	hs := newHarness(t, 120, 40, map[string]string{"proj/": ""})
	hs.to("proj")
	hs.keys("f", "3", "l", "l", "Enter")
	if hs.a.cur().dir != hs.path("proj") || hs.a.screen.Focused() != hs.a.files {
		t.Fatalf("Enter on the favorite: [1] is on %s", hs.a.cur().dir)
	}
	hs.keys("3", "o")
	hs.has("OPEN IN: ~/work/proj", "a tab of its own", "already there")
	hs.keys("Esc", "Esc", "D") // the first Esc takes the toast (tdp K4)
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

// The favorites and the task log are data (tdp E3): they are kept in
// data.yaml, not in the state; ones an older version left in state.yaml move
// there on the first start, and filu says so.
func TestFavoritesAndTasksAreData(t *testing.T) {
	hs := newHarness(t, 120, 40, map[string]string{"proj/": ""})
	if err := os.MkdirAll(filepath.Dir(statePathOverride), 0o755); err != nil {
		t.Fatal(err)
	}
	old := "pinned:\n  - " + hs.path("proj") + "\ntasks:\n  - id: 3\n    action: cp\n    dest: x\n    path: /tmp\n    total: 1\n    status: done\n"
	if err := os.WriteFile(statePathOverride, []byte(old), 0o644); err != nil {
		t.Fatal(err)
	}
	hs.a = NewApp(hs.root, "")
	hs.send(tea.WindowSizeMsg{Width: 120, Height: 40})
	hs.run(hs.a.start())
	hs.has("Moved the favorites and the task log")
	if !hs.a.favs.pinnedSet()[hs.path("proj")] || len(hs.a.tasks) != 1 {
		t.Fatalf("favorites %v, tasks %v", hs.a.favs.pinned, hs.a.tasks)
	}
	state, _ := os.ReadFile(statePathOverride)
	data, _ := os.ReadFile(dataPathOverride)
	if strings.Contains(string(state), "pinned") || strings.Contains(string(state), "tasks:") {
		t.Fatalf("state.yaml still has them:\n%s", state)
	}
	if !strings.Contains(string(data), hs.path("proj")) || !strings.Contains(string(data), "id: 3") {
		t.Fatalf("data.yaml:\n%s", data)
	}
}

// Favorites in state.yaml while data.yaml has its own stay in state.yaml, not
// deleted, and filu says so with both paths (tdp E3).
func TestFavoritesInTwoPlaces(t *testing.T) {
	hs := newHarness(t, 120, 40, map[string]string{"proj/": "", "other/": ""})
	for path, body := range map[string]string{
		statePathOverride: "pinned:\n  - " + hs.path("other") + "\n",
		dataPathOverride:  "favorites:\n  - " + hs.path("proj") + "\n",
	} {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	hs.a = NewApp(hs.root, "")
	hs.send(tea.WindowSizeMsg{Width: 120, Height: 40})
	hs.run(hs.a.start())
	hs.has("FOUND IN TWO PLACES", "state.yaml", "data.yaml")
	if set := hs.a.favs.pinnedSet(); !set[hs.path("proj")] || set[hs.path("other")] {
		t.Fatalf("favorites %v", hs.a.favs.pinned)
	}
	hs.a.shutdown()
	if state, _ := os.ReadFile(statePathOverride); !strings.Contains(string(state), hs.path("other")) {
		t.Fatalf("a save took the old favorites out of state.yaml:\n%s", state)
	}
}

// With one tab q leaves at once, a shell that goes where filu leaves it going
// where that tab is (user 2026-10-24). With more, quitting confirms; run by
// such a shell, q is a menu of where: the directory filu started in, then
// each tab's.
func TestQuit(t *testing.T) {
	for _, shell := range []bool{false, true} {
		hs := newHarness(t, 120, 40, map[string]string{"sub/": ""})
		last := filepath.Join(hs.home, "lastdir")
		if shell {
			t.Setenv(envLastDirFile, last)
			hs.a.syncQuit()
		}
		hs.to("sub")
		hs.keys("Enter", "q")
		if !hs.quit {
			t.Fatalf("q with one tab did not leave at once (shell %v)", shell)
		}
		hs.a.shutdown()
		if b, _ := os.ReadFile(last); shell && string(b) != hs.path("sub") {
			t.Fatalf("one tab: the shell is told %q", b)
		}
	}

	hs := newHarness(t, 120, 40, map[string]string{"sub/": ""})
	hs.keys("t", "s", "q")
	hs.has("Quit filu?")
	hs.keys("Enter")
	if !hs.quit {
		t.Fatal("Enter on the quit confirm did not quit")
	}

	for _, c := range []struct {
		key  string
		want func(*harness) string
	}{
		{"1", func(hs *harness) string { return hs.root }},
		{"2", func(hs *harness) string { return hs.path("sub") }},
	} {
		hs := newHarness(t, 120, 40, map[string]string{"sub/": ""})
		last := filepath.Join(hs.home, "lastdir")
		t.Setenv(envLastDirFile, last)
		hs.a.syncQuit()
		hs.to("sub")
		hs.keys("Enter", "t", "s", "q")
		hs.has("QUIT", "YOUR SHELL GOES TO", "~/work", "where filu started", "~/work/sub", "tab 1", "Enter:quit")
		hs.keys(c.key)
		if !hs.quit {
			t.Fatalf("%s on the quit menu did not quit", c.key)
		}
		hs.a.shutdown()
		if b, _ := os.ReadFile(last); string(b) != c.want(hs) {
			t.Fatalf("%s: the shell is told %q", c.key, b)
		}
	}
}

// With a popup open the first q or Ctrl-C never quits (tdp K9-6). With one
// tab filu would leave at once, so the lib asks Quit filu?; with two filu's
// flows ask anyway, the confirm or the menu, and ask only that. Esc goes back
// to the popup.
func TestQuitOverAPopup(t *testing.T) {
	for _, key := range []string{"q", "Ctrl-C"} {
		for _, c := range []struct {
			shell       bool
			more        []string // keys that open more tabs
			want, other string
		}{
			{false, nil, "Quit filu?", "YOUR SHELL GOES TO"},
			{true, nil, "Quit filu?", "YOUR SHELL GOES TO"},
			{false, []string{"t", "s"}, "Quit filu?", "YOUR SHELL GOES TO"},
			{true, []string{"t", "s"}, "YOUR SHELL GOES TO", "Quit filu?"},
		} {
			hs := newHarness(t, 120, 40, nil)
			if c.shell {
				t.Setenv(envLastDirFile, filepath.Join(hs.home, "lastdir"))
				hs.a.syncQuit()
			}
			hs.keys(c.more...)
			hs.keys("?", key)
			hs.has("Tab|Shift-Tab", c.want)
			hs.hasNot(c.other)
			if hs.quit {
				t.Fatalf("%s over the key reference quit (shell %v, %d tabs)", key, c.shell, len(hs.a.tabs))
			}
			hs.keys("Esc")
			hs.hasNot(c.want)
			hs.has("Tab|Shift-Tab")
		}
	}
}

// A second Ctrl-C during the quit menu leaves at once (tdp K9).
func TestQuitMenuSecondCtrlC(t *testing.T) {
	hs := newHarness(t, 120, 40, nil)
	t.Setenv(envLastDirFile, filepath.Join(hs.home, "lastdir"))
	hs.keys("t", "s")
	hs.keys("Ctrl-C")
	hs.has("YOUR SHELL GOES TO")
	hs.keys("Ctrl-C")
	if !hs.quit {
		t.Fatal("a second Ctrl-C did not quit")
	}
}

// A running task makes quitting warn, one tab too: the confirm, or choosing
// in the menu.
func TestQuitWarnsOfARunningTask(t *testing.T) {
	for _, shell := range []bool{false, true} {
		hs := newHarness(t, 120, 40, nil)
		if shell {
			t.Setenv(envLastDirFile, filepath.Join(hs.home, "lastdir"))
		}
		hs.a.addTask("cp", "x", hs.root, []string{"a"})
		hs.keys("q")
		hs.has("A task is still running", "Quit filu?")
		if hs.quit {
			t.Fatalf("one tab: quit without asking while a task runs (shell %v)", shell)
		}
	}

	hs := newHarness(t, 120, 40, nil)
	t.Setenv(envLastDirFile, filepath.Join(hs.home, "lastdir"))
	hs.keys("t", "s")
	hs.a.addTask("cp", "x", hs.root, []string{"a"})
	hs.keys("q", "1")
	hs.has("A task is still running", "Quit filu, your shell in ~/work?")
	if hs.quit {
		t.Fatal("quit without asking while a task runs")
	}
	hs.keys("Esc")
	hs.has("YOUR SHELL GOES TO") // back to the menu (tdp T1)
	hs.keys("1", "Enter")
	if !hs.quit {
		t.Fatal("quitting once asked did not quit")
	}
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
	hs.hasNot("SEARCH") // the menu it was opened from is gone too (tdp T1)
}

// What is typed starting with / or ~/ is a path: the finder lists what is
// under the deepest directory it names, outside the tab's directory too.
func TestPathSearch(t *testing.T) {
	for _, typed := range []string{"~/work/h", "<root>/h"} {
		hs := newHarness(t, 120, 40, map[string]string{"deep/er/needle.txt": "", "hay.txt": ""})
		hs.a.cd(hs.path("deep"), "")
		hs.keys("/", "f")
		hs.drain()
		hs.hasNot("hay.txt")
		hs.keys(strings.Replace(typed, "<root>", hs.root, 1))
		hs.drain()
		hs.has("~/work/hay.txt")
		hs.hasNot("needle.txt")
		hs.keys("Enter", "Enter")
		if hs.a.cur().dir != hs.root || hs.cursor() != "hay.txt" {
			t.Fatalf("%s: [1] is on %s, the cursor on %q", typed, hs.a.cur().dir, hs.cursor())
		}
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

// Content search lists the files that hold what is typed, as it is typed,
// each previewed from its first match.
func TestContentSearch(t *testing.T) {
	if _, err := lookPath("rg"); err != nil {
		t.Skip("ripgrep is not installed")
	}
	var long []string
	for i := 1; i <= 100; i++ {
		long = append(long, fmt.Sprintf("alpha%02d", i))
	}
	long[14] = "a needle here"
	hs := newHarness(t, 120, 40, map[string]string{"a.txt": "nothing\n", "c.txt": strings.Join(long, "\n") + "\n"})
	hs.keys("/", "c")
	hs.drain()
	hs.has("a.txt") // nothing typed: everything
	hs.keys("needle")
	hs.drain()
	hs.has("c.txt", "a needle here", "alpha13")
	hs.hasNot("a.txt   ", "alpha01")
	if it := hs.a.finder.Items; len(it) != 1 || it[0].Line != 15 {
		t.Fatalf("found %+v", it)
	}
	hs.keys("Enter", "Enter")
	if hs.cursor() != "c.txt" {
		t.Fatalf("the cursor is on %q", hs.cursor())
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
	hs.keys("q") // only closes it (tdp S3)
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
