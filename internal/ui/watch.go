package ui

import (
	"path/filepath"
	"time"

	tea "github.com/charmbracelet/bubbletea"
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

// waitWatch blocks for the next coalesced watch message, re-issued after each
// one (like waitLand); a nil channel (no watcher) yields no command.
func (m AppModel) waitWatch() tea.Cmd {
	ch := m.watchCh
	if ch == nil {
		return nil
	}
	return func() tea.Msg { return <-ch }
}

// syncWatches makes the watcher follow exactly the three tab directories —
// adding ones newly navigated to, dropping ones no tab shows any more.
func (m *AppModel) syncWatches() {
	if m.watcher == nil {
		return
	}
	want := map[string]bool{}
	for _, t := range m.tabs {
		if t.dir != "" {
			want[t.dir] = true
		}
	}
	for d := range m.watched {
		if !want[d] {
			m.watcher.Remove(d)
			delete(m.watched, d)
		}
	}
	for d := range want {
		if !m.watched[d] && m.watcher.Add(d) == nil {
			m.watched[d] = true
		}
	}
}

// handleWatchMsg reloads every tab whose directory changed, keeping each cursor
// on its named entry, and refreshes the preview if the active tab moved.
func (m *AppModel) handleWatchMsg(msg watchMsg) {
	changed := map[string]bool{}
	for _, d := range msg.dirs {
		changed[d] = true
	}
	for i := range m.tabs {
		if changed[m.tabs[i].dir] {
			m.tabs[i].reloadPreserving()
		}
	}
	if changed[m.active().dir] {
		m.cur().ensureVisible(m.listRows())
		m.refreshPreview()
	}
}
