package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

// TestMarkMoveRekey: `m` marks the cursor item into the bucket (was `p`); the old
// `p` no longer marks. The list Space menu binds Mark=m, Copy=c, Move here=v.
func TestMarkMoveRekey(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "f.txt"))
	target := filepath.Join(dir, "f.txt")

	m := minModel()
	m.width, m.height = 100, 30
	m.tabs = []listModel{newList(dir)}
	m.tab = 0
	m.cur().cursor = 0

	m.handleListKey("m") // mark
	if !m.marks.inBucket()[target] {
		t.Fatal("m should mark the cursor item into the bucket")
	}
	m.handleListKey("m") // toggle off
	if len(m.marks.items) != 0 {
		t.Errorf("second m should unmark; %d in bucket", len(m.marks.items))
	}
	m.handleListKey("p") // old pick key: no longer marks
	if len(m.marks.items) != 0 {
		t.Errorf("old p should not mark; %d in bucket", len(m.marks.items))
	}

	// Space-menu bindings: Mark=m, and (with a marked item) Copy=c, Move here=v.
	m.marks.items = []string{target}
	items, _ := m.buildSpaceMenu()
	byLabel := map[string]string{}
	for _, it := range items {
		byLabel[it.label] = it.key
	}
	for label, want := range map[string]string{"Mark": "m", "Copy": "c", "Move here": "v"} {
		if byLabel[label] != want {
			t.Errorf("menu %q key = %q, want %q", label, byLabel[label], want)
		}
	}
}

func TestCarryPick(t *testing.T) {
	m := marksModel{items: []string{"/a", "/b", "/c"}}
	if len(m.landSet()) != 3 {
		t.Errorf("no pick → land everything (3), got %d", len(m.landSet()))
	}
	m.cursor = 1
	m.togglePick()
	if !m.picked["/b"] {
		t.Fatal("togglePick should pick /b")
	}
	if set := m.landSet(); len(set) != 1 || !set["/b"] {
		t.Errorf("landSet should be just /b, got %v", set)
	}
	m.togglePick()
	if m.picked["/b"] {
		t.Error("second togglePick should unpick /b")
	}
}

func TestLandItemsPickedOnly(t *testing.T) {
	m := marksModel{items: []string{"/a", "/b", "/c"}}
	m.cursor = 0
	m.togglePick() // pick /a only
	if got := m.landItems(); len(got) != 1 || got[0] != "/a" {
		t.Errorf("landItems should be just /a, got %v", got)
	}
}

func TestMarksViewShowsFullPath(t *testing.T) {
	m := marksModel{items: []string{"/tmp/projects/demo/notes.txt"}}
	out := ansi.Strip(m.view(60, 4, false))
	if !strings.Contains(out, "projects/demo/notes.txt") {
		t.Errorf("marks view should show the full path, got %q", out)
	}
}

// TestMarkPickGlyphIsDistinct: a picked Marks item uses markPickGlyph, never the
// list's mark markGlyph — the two "picked" states must not read the same.
func TestMarkPickGlyphIsDistinct(t *testing.T) {
	m := marksModel{items: []string{"/a"}, picked: map[string]bool{"/a": true}}
	out := ansi.Strip(m.view(60, 4, false))
	if !strings.Contains(out, markPickGlyph) {
		t.Errorf("picked marks item should show markPickGlyph, got %q", out)
	}
	if strings.Contains(out, markGlyph) {
		t.Errorf("the Marks pick must not reuse the list's mark glyph, got %q", out)
	}
}

func TestHandleLandMsgFinish(t *testing.T) {
	var m AppModel
	m.tabs = []listModel{{}} // one tab so refreshPreview's cur() is valid
	m.tasks = []landTask{{id: 1, action: "cp", dest: "dst", total: 2}}
	m.marks.items = []string{"/a", "/b"}

	m.handleLandMsg(landMsg{taskID: 1, done: 1, total: 2}) // progress
	if m.tasks[0].done != 1 {
		t.Errorf("progress not applied: %+v", m.tasks[0])
	}
	m.handleLandMsg(landMsg{taskID: 1, done: 2, total: 2, finished: true, moved: []string{"/a"}})
	if len(m.tasks) != 1 || m.tasks[0].status != taskDone {
		t.Errorf("finished task should stay as done, got %+v", m.tasks)
	}
	if len(m.marks.items) != 1 || m.marks.items[0] != "/b" {
		t.Errorf("moved /a should leave the bucket, got %v", m.marks.items)
	}
}

func TestRunLandCopy(t *testing.T) {
	src, dst := t.TempDir(), t.TempDir()
	a := filepath.Join(src, "a.txt")
	if err := os.WriteFile(a, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	ch := make(chan landMsg, 8)
	go runLand(1, []string{a}, dst, false, ch)

	var last landMsg
	for {
		msg := <-ch
		last = msg
		if msg.finished {
			break
		}
	}
	if last.total != 1 || last.done != 1 {
		t.Errorf("finish msg = %+v, want done/total 1/1", last)
	}
	if len(last.moved) != 0 {
		t.Errorf("copy should report no moved items, got %v", last.moved)
	}
	if _, err := os.Stat(filepath.Join(dst, "a.txt")); err != nil {
		t.Error("a.txt should be copied to dst")
	}
}
