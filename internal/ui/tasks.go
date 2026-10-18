package ui

import (
	"path/filepath"
	"strings"
	"time"
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

// taskTime formats a task's timestamp for the log as YYYY-MM-DD HH:MM:SS (19
// wide, so the icon column stays aligned). Zero = blank.
func taskTime(at time.Time) string {
	if at.IsZero() {
		return strings.Repeat(" ", 19)
	}
	return at.Format("2006-01-02 15:04:05")
}
