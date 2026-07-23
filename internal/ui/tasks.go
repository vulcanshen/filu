package ui

import (
	"fmt"
	"path/filepath"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// landTask is an in-flight land (copy/move) shown in the Progress tab.
type landTask struct {
	id       int
	action   string // "cp" / "mv"
	dest     string // destination basename (display)
	destPath string // destination dir (for reloading matching tabs on finish)
	done     int
	total    int
}

// landMsg is emitted by a land goroutine: per-item progress, then a finished
// message carrying the srcs that left the bucket (moves only).
type landMsg struct {
	taskID   int
	done     int
	total    int
	finished bool
	moved    []string
}

// runLand copies/moves items into destDir on a goroutine, streaming progress to
// ch. It never touches model state — the main loop applies the results.
func runLand(id int, items []string, destDir string, move bool, ch chan<- landMsg) {
	var moved []string
	for i, src := range items {
		dst := uniquePath(filepath.Join(destDir, filepath.Base(src)))
		var err error
		if move {
			err = movePath(src, dst)
		} else {
			err = copyPath(src, dst)
		}
		if err == nil && move {
			moved = append(moved, src)
		}
		ch <- landMsg{taskID: id, done: i + 1, total: len(items)}
	}
	ch <- landMsg{taskID: id, done: len(items), total: len(items), finished: true, moved: moved}
}

// waitLand blocks on the task channel for the next land message. Re-issued after
// each one so exactly one reader is outstanding (started in Init).
func (m AppModel) waitLand() tea.Cmd {
	ch := m.taskCh
	return func() tea.Msg { return <-ch }
}

// startLand snapshots the land subset and kicks off an async copy/move.
func (m *AppModel) startLand(destDir string, move bool) {
	items := m.carry.landItems()
	if len(items) == 0 {
		return
	}
	m.nextTaskID++
	action := "cp"
	if move {
		action = "mv"
	}
	m.tasks = append(m.tasks, landTask{
		id: m.nextTaskID, action: action,
		dest: filepath.Base(destDir), destPath: destDir, total: len(items),
	})
	go runLand(m.nextTaskID, items, destDir, move, m.taskCh)
}

// handleLandMsg applies a land goroutine's progress/finish to the model.
func (m *AppModel) handleLandMsg(msg landMsg) {
	for i := range m.tasks {
		if m.tasks[i].id != msg.taskID {
			continue
		}
		m.tasks[i].done = msg.done
		if !msg.finished {
			return
		}
		t := m.tasks[i]
		m.carry.history = append([]historyEntry{{t.action, t.total, t.dest, time.Now()}}, m.carry.history...)
		for _, src := range msg.moved {
			m.carry.removeItem(src)
		}
		m.tasks = append(m.tasks[:i], m.tasks[i+1:]...)
		for j := range m.tabs { // surface the landed files
			if m.tabs[j].dir == t.destPath {
				m.tabs[j].reload()
			}
		}
		m.refreshPreview()
		return
	}
}

// progressView renders the Progress tab.
func (m AppModel) progressView(w, rows int) string {
	if len(m.tasks) == 0 {
		return centeredNote(w, rows, "(no active tasks)")
	}
	lines := make([]string, len(m.tasks))
	for i, t := range m.tasks {
		pct := 0
		if t.total > 0 {
			pct = t.done * 100 / t.total
		}
		lines[i] = truncate(fmt.Sprintf(" %s #%d  %d/%d  %d%%", t.action, t.id, t.done, t.total, pct), w)
	}
	return renderLines(lines, w, rows)
}
