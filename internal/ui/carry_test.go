package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

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

func TestHandleLandMsgFinish(t *testing.T) {
	var m AppModel
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
