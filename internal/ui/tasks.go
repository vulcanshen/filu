package ui

import (
	"fmt"
	"path/filepath"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type taskStatus int

const (
	taskRunning taskStatus = iota // in flight (spinner)
	taskDone                      // finished ok (tick)
	taskPending                   // interrupted last session (restored, clock)
	taskError                     // finished with failures (cross)
)

const maxTasks = 30 // cap the merged task log

// landTask is a copy/move operation shown in the Tasks tab. Running tasks
// animate; finished ones stay as the log (Progress + History merged).
type landTask struct {
	id       int
	action   string // "cp" / "mv"
	dest     string // destination basename (display)
	destPath string // destination dir (reload matching tabs on finish)
	total    int
	done     int
	status   taskStatus
}

// landMsg is emitted by a land goroutine: per-item progress, then a finished
// message with the failure count and the srcs that left the bucket (moves).
type landMsg struct {
	taskID   int
	done     int
	total    int
	finished bool
	failed   int
	moved    []string
}

// spinnerFrames animates running tasks (braille dots).
var spinnerFrames = []string{
	string(rune(0x280b)), string(rune(0x2819)), string(rune(0x2839)), string(rune(0x2838)),
	string(rune(0x283c)), string(rune(0x2834)), string(rune(0x2826)), string(rune(0x2827)),
	string(rune(0x2807)), string(rune(0x280f)),
}

type spinnerTickMsg struct{}

func spinnerTick() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(time.Time) tea.Msg { return spinnerTickMsg{} })
}

// runLand copies/moves items into destDir on a goroutine, streaming progress to
// ch. It never touches model state — the main loop applies the results.
func runLand(id int, items []string, destDir string, move bool, ch chan<- landMsg) {
	var moved []string
	failed := 0
	for i, src := range items {
		dst := uniquePath(filepath.Join(destDir, filepath.Base(src)))
		var err error
		if move {
			err = movePath(src, dst)
		} else {
			err = copyPath(src, dst)
		}
		switch {
		case err != nil:
			failed++
		case move:
			moved = append(moved, src)
		}
		ch <- landMsg{taskID: id, done: i + 1, total: len(items)}
	}
	ch <- landMsg{taskID: id, done: len(items), total: len(items), finished: true, failed: failed, moved: moved}
}

// waitLand blocks on the task channel for the next land message (re-issued after
// each one so exactly one reader is outstanding; started in Init).
func (m AppModel) waitLand() tea.Cmd {
	ch := m.taskCh
	return func() tea.Msg { return <-ch }
}

func (m AppModel) anyRunning() bool {
	for _, t := range m.tasks {
		if t.status == taskRunning {
			return true
		}
	}
	return false
}

// startLand snapshots the land subset and kicks off an async copy/move, adding a
// running task (persisted immediately as "undone" so an interrupt is recorded).
func (m *AppModel) startLand(destDir string, move bool) tea.Cmd {
	items := m.carry.landItems()
	if len(items) == 0 {
		return nil
	}
	m.nextTaskID++
	action := "cp"
	if move {
		action = "mv"
	}
	m.tasks = append(m.tasks, landTask{
		id: m.nextTaskID, action: action,
		dest: filepath.Base(destDir), destPath: destDir,
		total: len(items), status: taskRunning,
	})
	m.capTasks()
	go runLand(m.nextTaskID, items, destDir, move, m.taskCh)
	saveState(m.snapshotState())
	if !m.spinning {
		m.spinning = true
		return spinnerTick()
	}
	return nil
}

func (m *AppModel) capTasks() {
	if len(m.tasks) > maxTasks {
		m.tasks = m.tasks[len(m.tasks)-maxTasks:]
	}
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
		if msg.failed > 0 {
			m.tasks[i].status = taskError
		} else {
			m.tasks[i].status = taskDone
		}
		destPath := m.tasks[i].destPath
		for _, src := range msg.moved {
			m.carry.removeItem(src)
		}
		for j := range m.tabs { // surface the landed files
			if m.tabs[j].dir == destPath {
				m.tabs[j].reload()
			}
		}
		m.refreshPreview()
		saveState(m.snapshotState()) // persist done / error
		return
	}
}

// tasksView renders the merged Tasks tab (running + done + pending + error).
func (m AppModel) tasksView(w, rows int) string {
	if len(m.tasks) == 0 {
		return centeredNote(w, rows, "(no tasks)")
	}
	lines := make([]string, len(m.tasks))
	for i, t := range m.tasks {
		lines[i] = truncate(m.taskLine(t), w)
	}
	return renderLines(lines, w, rows)
}

func (m AppModel) taskLine(t landTask) string {
	green := lipgloss.NewStyle().Foreground(lipgloss.Color("#a6e3a1"))
	peach := lipgloss.NewStyle().Foreground(lipgloss.Color("#fab387"))
	red := lipgloss.NewStyle().Foreground(lipgloss.Color("#f38ba8"))
	blue := lipgloss.NewStyle().Foreground(focusColor)
	head := fmt.Sprintf("%s #%d", t.action, t.id)

	switch t.status {
	case taskRunning:
		pct := 0
		if t.total > 0 {
			pct = t.done * 100 / t.total
		}
		spin := spinnerFrames[m.spinnerFrame%len(spinnerFrames)]
		return " " + blue.Render(spin) + " " + fmt.Sprintf("%s  %d/%d  %d%%", head, t.done, t.total, pct)
	case taskDone:
		return " " + green.Render(string(rune(0xf00c))) + " " + fmt.Sprintf("%s → %s (%d)", head, t.dest, t.total)
	case taskPending:
		return " " + peach.Render(string(rune(0xf017))) + " " + head + "  interrupted" // clock
	case taskError:
		return " " + red.Render(string(rune(0xf00d))) + " " + head + "  failed" // cross
	}
	return " " + head
}
