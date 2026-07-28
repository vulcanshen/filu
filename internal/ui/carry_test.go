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
	if !m.carry.inBucket()[target] {
		t.Fatal("m should mark the cursor item into the bucket")
	}
	m.handleListKey("m") // toggle off
	if len(m.carry.items) != 0 {
		t.Errorf("second m should unmark; %d in bucket", len(m.carry.items))
	}
	m.handleListKey("p") // old pick key: no longer marks
	if len(m.carry.items) != 0 {
		t.Errorf("old p should not mark; %d in bucket", len(m.carry.items))
	}

	// Space-menu bindings: Mark=m, and (with a marked item) Copy=c, Move here=v.
	m.carry.items = []string{target}
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
	m := carryModel{items: []string{"/a", "/b", "/c"}}
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
	m := carryModel{items: []string{"/a", "/b", "/c"}}
	m.cursor = 0
	m.togglePick() // pick /a only
	if got := m.landItems(); len(got) != 1 || got[0] != "/a" {
		t.Errorf("landItems should be just /a, got %v", got)
	}
}

func TestCarryViewShowsFullPath(t *testing.T) {
	m := carryModel{items: []string{"/tmp/projects/demo/notes.txt"}}
	out := ansi.Strip(m.view(60, 4, false))
	if !strings.Contains(out, "projects/demo/notes.txt") {
		t.Errorf("carry view should show the full path, got %q", out)
	}
}

// TestCarryPickGlyphIsDistinct: a picked Carries item uses carryPickGlyph, never
// panel [2]'s bucket pickGlyph — the two "picked" states must not read the same.
func TestCarryPickGlyphIsDistinct(t *testing.T) {
	m := carryModel{items: []string{"/a"}, picked: map[string]bool{"/a": true}}
	out := ansi.Strip(m.view(60, 4, false))
	if !strings.Contains(out, carryPickGlyph) {
		t.Errorf("picked carry item should show carryPickGlyph, got %q", out)
	}
	if strings.Contains(out, pickGlyph) {
		t.Errorf("panel [4] pick must not reuse panel [2]'s bucket glyph, got %q", out)
	}
}

func TestHandleLandMsgFinish(t *testing.T) {
	var m AppModel
	m.tabs = []listModel{{}} // one tab so refreshPreview's cur() is valid
	m.tasks = []landTask{{id: 1, action: "cp", dest: "dst", total: 2}}
	m.carry.items = []string{"/a", "/b"}

	m.handleLandMsg(landMsg{taskID: 1, done: 1, total: 2}) // progress
	if m.tasks[0].done != 1 {
		t.Errorf("progress not applied: %+v", m.tasks[0])
	}
	m.handleLandMsg(landMsg{taskID: 1, done: 2, total: 2, finished: true, moved: []string{"/a"}})
	if len(m.tasks) != 1 || m.tasks[0].status != taskDone {
		t.Errorf("finished task should stay as done, got %+v", m.tasks)
	}
	if len(m.carry.items) != 1 || m.carry.items[0] != "/b" {
		t.Errorf("moved /a should leave the bucket, got %v", m.carry.items)
	}
}

func TestRedoTask(t *testing.T) {
	src, dst := t.TempDir(), t.TempDir()
	a := filepath.Join(src, "a.txt")
	if err := os.WriteFile(a, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	m := AppModel{taskCh: make(chan landMsg, 16)}
	done := landTask{id: 1, action: "cp", dest: filepath.Base(dst), destPath: dst, srcs: []string{a}, total: 1, status: taskDone}
	m.tasks = []landTask{done}

	m.redoTask(done)
	if len(m.tasks) != 2 || m.tasks[1].status != taskRunning {
		t.Fatalf("redo should add a running task, got %+v", m.tasks)
	}
	for { // let the goroutine finish
		if (<-m.taskCh).finished {
			break
		}
	}
	if _, err := os.Stat(filepath.Join(dst, "a.txt")); err != nil {
		t.Error("redo should re-copy a.txt")
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
