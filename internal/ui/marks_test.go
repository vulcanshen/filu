package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
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

// TestMarksViewTypeGlyphs: a bucket row carries the same type glyph panel [1]
// draws — a folder reads as a folder, a .go file as Go — not one generic file
// glyph for everything.
func TestMarksViewTypeGlyphs(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "pkg")
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	src := filepath.Join(dir, "main.go")
	if err := os.WriteFile(src, []byte("package main"), 0o644); err != nil {
		t.Fatal(err)
	}

	out := ansi.Strip(marksModel{items: []string{sub, src}}.view(70, 4, false))
	if !strings.Contains(out, iconDir) {
		t.Errorf("a marked directory should show the folder glyph, got %q", out)
	}
	if want := fileIcon(fileItem{name: "main.go"}); !strings.Contains(out, want) {
		t.Errorf("a marked .go file should show the Go glyph %q, got %q", want, out)
	}
}

// TestPathIcon: the glyph is resolved from the path, and an unreadable path still
// falls back to its extension rather than erroring out.
func TestPathIcon(t *testing.T) {
	dir := t.TempDir()
	if got := pathIcon(dir); got != iconDir {
		t.Errorf("pathIcon(dir) = %q, want the folder glyph %q", got, iconDir)
	}
	if got, want := pathIcon("/no/such/file.go"), fileIcon(fileItem{name: "file.go"}); got != want {
		t.Errorf("pathIcon on a missing path = %q, want the extension glyph %q", got, want)
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

// TestMarksClearConfirms: C on the Marks tab arms a confirm and leaves the
// bucket alone; accepting drops every mark and every pick.
func TestMarksClearConfirms(t *testing.T) {
	m := minModel()
	m.width, m.height = 80, 24
	m.focus = panelMarks
	m.marks.items = []string{"/tmp/a", "/tmp/b"}
	m.marks.picked = map[string]bool{"/tmp/b": true}
	m.marks.cursor = 1

	m.handleMarksKey("C")
	if m.confirmAction != confirmClearMarks {
		t.Fatalf("C should arm the clear confirm, got %v", m.confirmAction)
	}
	if len(m.marks.items) != 2 {
		t.Errorf("C must not clear before the confirm is accepted, %d left", len(m.marks.items))
	}

	m.confirm.anim.state = popupOpen
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(AppModel)
	if len(m.marks.items) != 0 || len(m.marks.picked) != 0 {
		t.Fatalf("accepting should empty the bucket, items=%v picked=%v", m.marks.items, m.marks.picked)
	}
	if m.marks.cursor != 0 {
		t.Errorf("cursor should reset to 0, got %d", m.marks.cursor)
	}
}

// TestMarksClearNoopWhenEmpty: C on an empty bucket arms nothing.
func TestMarksClearNoopWhenEmpty(t *testing.T) {
	m := minModel()
	m.focus = panelMarks
	m.handleMarksKey("C")
	if m.confirmAction != confirmNone {
		t.Errorf("C on an empty bucket should arm no confirm, got %v", m.confirmAction)
	}
}
