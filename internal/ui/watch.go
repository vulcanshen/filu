package ui

import (
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
)

// watchMsg carries the directories that changed on disk since the last flush.
type watchMsg struct{ dirs []string }

// newWatcher creates a directory watcher, or nil if fsnotify can't start — filu
// still runs, just without live refresh.
func newWatcher() *fsnotify.Watcher {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return nil
	}
	return w
}

// watchLoop coalesces fsnotify events: it gathers the dirs touched during a
// burst and emits one watchMsg after a short quiet gap, so a big copy fires a
// single reload rather than hundreds. Ends when the watcher is closed.
func watchLoop(w *fsnotify.Watcher, ch chan<- watchMsg) {
	changed := map[string]bool{}
	var flush <-chan time.Time
	for {
		select {
		case ev, ok := <-w.Events:
			if !ok {
				return
			}
			changed[filepath.Dir(ev.Name)] = true
			flush = time.After(120 * time.Millisecond)
		case _, ok := <-w.Errors:
			if !ok {
				return
			}
		case <-flush:
			dirs := make([]string, 0, len(changed))
			for d := range changed {
				dirs = append(dirs, d)
			}
			changed = map[string]bool{}
			flush = nil
			ch <- watchMsg{dirs: dirs}
		}
	}
}
