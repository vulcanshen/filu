package ui

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
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
	action   string   // "cp" / "mv"
	dest     string   // destination basename (display)
	destPath string   // destination dir (reload matching tabs on finish)
	srcs     []string // source paths
	total    int
	done     int
	failed   int       // failures on finish (for the error line)
	at       time.Time // when the task was created (shown as the log timestamp)
	status   taskStatus
}

// landMsg is emitted by a land goroutine: per-item progress, then a finished
// message with the failure count, the srcs that left the bucket (moves), and the
// file the task produced (zips).
type landMsg struct {
	taskID   int
	done     int
	total    int
	finished bool
	failed   int
	moved    []string
	produced string
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

// startLand kicks off a land of the marks bucket's subset.
func (m *AppModel) startLand(destDir string, move bool) tea.Cmd {
	return m.startLandItems(m.marks.landItems(), destDir, move)
}

// startLandItems adds a running task (persisted immediately as "undone" so an
// interrupt is recorded) and launches the async copy/move.
func (m *AppModel) startLandItems(items []string, destDir string, move bool) tea.Cmd {
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
		dest: filepath.Base(destDir), destPath: destDir, srcs: items,
		total: len(items), at: time.Now(), status: taskRunning,
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

func (m *AppModel) clampTaskCursor() {
	if m.taskCursor > len(m.tasks)-1 {
		m.taskCursor = len(m.tasks) - 1
	}
	if m.taskCursor < 0 {
		m.taskCursor = 0
	}
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
		m.tasks[i].failed = msg.failed
		if msg.failed > 0 {
			m.tasks[i].status = taskError
		} else {
			m.tasks[i].status = taskDone
		}
		destPath := m.tasks[i].destPath
		for _, src := range msg.moved {
			m.marks.removeItem(src)
		}
		if msg.produced != "" { // a zip: carry it as the sole pick, ready to land
			m.marks.items = append(m.marks.items, msg.produced)
			m.marks.picked = map[string]bool{msg.produced: true}
			m.marks.cursor = len(m.marks.items) - 1
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

// tasksView renders the merged Tasks tab (running + done + pending + error),
// with a cursor when focused.
func (m AppModel) tasksView(w, rows int, focused bool) string {
	if len(m.tasks) == 0 {
		return centeredNote(w, rows, "(no tasks)")
	}
	cursorBg := handColor
	if !focused {
		cursorBg = borderDim
	}
	cur := lipgloss.NewStyle().Foreground(lipgloss.Color(baseHex)).Background(cursorBg)

	start := 0
	if m.taskCursor >= rows { // keep the cursor in view
		start = m.taskCursor - rows + 1
	}
	end := min(start+rows, len(m.tasks))
	var b strings.Builder
	for i := start; i < end; i++ {
		if focused && i == m.taskCursor {
			b.WriteString(cur.Render(padDisp(truncate(ansi.Strip(m.taskLine(m.tasks[i])), w), w)))
		} else {
			b.WriteString(truncate(m.taskLine(m.tasks[i]), w))
		}
		if i < end-1 {
			b.WriteByte('\n')
		}
	}
	return b.String()
}

// taskTime formats a task's timestamp for the log as YYYY-MM-DD HH:MM:SS (19
// wide, so the icon column stays aligned). Zero = blank.
func taskTime(at time.Time) string {
	if at.IsZero() {
		return strings.Repeat(" ", 19)
	}
	return at.Format("2006-01-02 15:04:05")
}

// taskLine renders one Tasks-tab row in plain language: a timestamp, an icon, the
// action, what it acted on (a filename, or "N items"), and the destination — no
// internal id.
func (m AppModel) taskLine(t landTask) string {
	green := lipgloss.NewStyle().Foreground(lipgloss.Color("#a6e3a1"))
	peach := lipgloss.NewStyle().Foreground(lipgloss.Color("#fab387"))
	red := lipgloss.NewStyle().Foreground(lipgloss.Color("#f38ba8"))
	blue := lipgloss.NewStyle().Foreground(focusColor)
	dim := lipgloss.NewStyle().Foreground(dimColor)

	subject := fmt.Sprintf("%d items", t.total)
	if len(t.srcs) == 1 {
		subject = safeName(filepath.Base(t.srcs[0]))
	}
	verb, verbing, verbed := "Copy", "Copying", "Copied"
	switch t.action {
	case "mv":
		verb, verbing, verbed = "Move", "Moving", "Moved"
	case "zip":
		verb, verbing, verbed = "Zip", "Zipping", "Zipped"
	}
	to := " → " + t.dest
	stamp := " " + dim.Render(taskTime(t.at)) + " " // leading timestamp column

	switch t.status {
	case taskRunning:
		spin := spinnerFrames[m.spinnerFrame%len(spinnerFrames)]
		return stamp + blue.Render(spin) + " " + verbing + " " + subject + to + "  " + dim.Render(fmt.Sprintf("%d/%d", t.done, t.total))
	case taskDone:
		return stamp + green.Render(string(rune(0xf00c))) + " " + verbed + " " + subject + to // tick
	case taskPending:
		return stamp + peach.Render(string(rune(0xf017))) + " " + verb + " " + subject + to + "  " + dim.Render("· interrupted") // clock
	case taskError:
		return stamp + red.Render(string(rune(0xf00d))) + " " + verb + " " + subject + to + "  " + red.Render(fmt.Sprintf("· %d/%d failed", t.failed, t.total)) // cross
	}
	return stamp + verb + " " + subject + to
}
