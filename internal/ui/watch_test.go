package ui

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func writeFile(t *testing.T, path string) {
	t.Helper()
	if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestReloadPreservingKeepsCursorOnName(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "b.txt"))
	writeFile(t, filepath.Join(dir, "c.txt"))
	l := newList(dir) // sorted: b.txt, c.txt
	for i, it := range l.items {
		if it.name == "c.txt" {
			l.cursor = i
		}
	}
	writeFile(t, filepath.Join(dir, "a.txt")) // external add, sorts before the cursor

	l.reloadPreserving()
	if got := l.cursorItem().name; got != "c.txt" {
		t.Errorf("cursor should stay on c.txt after an external add, got %q", got)
	}
	if len(l.items) != 3 {
		t.Errorf("reload should pick up the new file: %d items", len(l.items))
	}
}

func TestHandleWatchMsgReloadsChangedTab(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "one.txt"))
	m := AppModel{taskCh: make(chan landMsg, 1), watched: map[string]bool{}}
	for i := range m.tabs {
		m.tabs[i] = newList(dir)
	}
	before := len(m.tabs[0].items)

	writeFile(t, filepath.Join(dir, "two.txt"))
	m.handleWatchMsg(watchMsg{dirs: []string{dir}})
	if got := len(m.tabs[0].items); got != before+1 {
		t.Errorf("watched tab should reload to %d items, got %d", before+1, got)
	}

	// a change to an unrelated dir must not touch the tab
	m.handleWatchMsg(watchMsg{dirs: []string{"/nonexistent/elsewhere"}})
	if got := len(m.tabs[0].items); got != before+1 {
		t.Errorf("unrelated change should not reload: %d items", got)
	}
}

func TestSyncWatchesFollowsTabDirs(t *testing.T) {
	d1, d2 := t.TempDir(), t.TempDir()
	w := newWatcher()
	if w == nil {
		t.Skip("no fsnotify watcher available")
	}
	defer w.Close()

	m := AppModel{watcher: w, watched: map[string]bool{}}
	m.tabs[0] = listModel{dir: d1}
	m.tabs[1] = listModel{dir: d1} // duplicate — should dedup to one watch
	m.tabs[2] = listModel{dir: d2}

	m.syncWatches()
	if !m.watched[d1] || !m.watched[d2] || len(m.watched) != 2 {
		t.Fatalf("watched should be {d1,d2}, got %v", m.watched)
	}

	m.tabs[2].dir = d1 // navigate away from d2
	m.syncWatches()
	if m.watched[d2] || !m.watched[d1] || len(m.watched) != 1 {
		t.Errorf("d2 should be dropped once no tab shows it, got %v", m.watched)
	}
}

func TestWatchLoopEmitsOnChange(t *testing.T) {
	dir := t.TempDir()
	w := newWatcher()
	if w == nil {
		t.Skip("no fsnotify watcher available")
	}
	defer w.Close()
	if err := w.Add(dir); err != nil {
		t.Fatal(err)
	}
	ch := make(chan watchMsg, 4)
	go watchLoop(w, ch)

	writeFile(t, filepath.Join(dir, "new.txt"))
	select {
	case msg := <-ch:
		found := false
		for _, d := range msg.dirs {
			if d == dir {
				found = true
			}
		}
		if !found {
			t.Errorf("watchMsg dirs %v should include %q", msg.dirs, dir)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("no watch event within 3s of a file create")
	}
}
