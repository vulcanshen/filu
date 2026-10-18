package ui

import (
	"os"
	"path/filepath"
	"sort"

	tea "github.com/charmbracelet/bubbletea"
)

// snapshotState is the session to save: the tabs after the first (which always
// opens where filu starts), the marks, the favorites, the tasks and each
// directory's sort.
func (a *App) snapshotState() sessionState {
	st := sessionState{Marks: a.marks.items}
	for _, t := range a.tabs[1:] {
		st.Tabs = append(st.Tabs, tabState{Dir: t.dir, Cursor: t.cursorIndex()})
	}
	for _, p := range a.favs.pinned {
		st.Pinned = append(st.Pinned, p.path)
	}
	for _, t := range a.tasks {
		st.Tasks = append(st.Tasks, persistedTask{
			ID: t.id, Action: t.action, Dest: t.dest, Path: t.destPath, Srcs: t.srcs, Total: t.total,
			Failed: t.failed, At: t.at, Status: taskStatusString(t.status),
		})
	}
	paths := make([]string, 0, len(sortByDir))
	for p := range sortByDir {
		paths = append(paths, p)
	}
	sort.Strings(paths) // a stable state.yaml
	for _, p := range paths {
		ds := dirSortYAML{Path: p}
		for _, r := range sortByDir[p] {
			ds.Rules = append(ds.Rules, sortRuleYAML{Col: int(r.col), Asc: r.asc})
		}
		st.Sorts = append(st.Sorts, ds)
	}
	return st
}

// applyState brings back a saved session. A task that was running when filu
// ended comes back as interrupted.
func (a *App) applyState(st sessionState) {
	sortByDir = map[string][]sortRule{}
	for _, ds := range st.Sorts {
		var rules []sortRule
		for _, r := range ds.Rules {
			rules = append(rules, sortRule{col: sortCol(r.Col), asc: r.Asc})
		}
		if len(rules) > 0 {
			sortByDir[cleanDir(ds.Path)] = rules
		}
	}
	first := a.tabs[0]
	cur, hasCur := first.list.Current()
	first.reload() // sorted now that the sorts are back
	first.list.SetRows(fileRows(first, nil))
	if hasCur {
		first.list.Select(cur.ID)
	}
	for _, ts := range st.Tabs {
		if ts.Dir == "" || len(a.tabs) >= maxTabs {
			continue
		}
		t := newTab(ts.Dir)
		t.list.SetRows(fileRows(t, nil))
		t.selectIndex(ts.Cursor)
		a.tabs = append(a.tabs, t)
	}
	a.marks.items = st.Marks
	for _, p := range st.Pinned {
		a.favs.pinned = append(a.favs.pinned, place{label: filepath.Base(p), path: p, icon: iconPin})
	}
	for _, pt := range st.Tasks {
		status := taskPending
		switch pt.Status {
		case "done":
			status = taskDone
		case "error":
			status = taskError
		}
		a.tasks = append(a.tasks, landTask{
			id: pt.ID, action: pt.Action, dest: pt.Dest, destPath: pt.Path, srcs: pt.Srcs,
			total: pt.Total, done: pt.Total, failed: pt.Failed, at: pt.At, status: status,
		})
		a.nextTaskID = max(a.nextTaskID, pt.ID)
	}
}

// --- live refresh --------------------------------------------------------------

func (a *App) waitWatch() tea.Cmd {
	ch := a.watchCh
	if ch == nil {
		return nil
	}
	return func() tea.Msg { return <-ch }
}

// syncWatches makes the watcher follow the tabs' directories, no more.
func (a *App) syncWatches() {
	if a.watcher == nil {
		return
	}
	want := map[string]bool{}
	for _, t := range a.tabs {
		want[t.dir] = true
	}
	for d := range a.watched {
		if !want[d] {
			a.watcher.Remove(d)
			delete(a.watched, d)
		}
	}
	for d := range want {
		if !a.watched[d] && a.watcher.Add(d) == nil {
			a.watched[d] = true
		}
	}
}

// handleWatch reads again every tab whose directory changed on disk; each
// cursor stays on its entry.
func (a *App) handleWatch(msg watchMsg) {
	for _, d := range msg.dirs {
		a.reloadDir(d)
	}
	a.refreshPreview()
}

// envLastDirFile names the file filu writes the chosen directory to on quit, so
// a shell wrapper can cd there — superfile's cd_on_quit. Unset = feature off.
const envLastDirFile = "FILU__LAST_DIR_FILE"

// writeLastDir records dir for the shell wrapper's cd-on-quit, when enabled.
func writeLastDir(dir string) {
	path := os.Getenv(envLastDirFile)
	if path == "" || dir == "" {
		return
	}
	_ = os.WriteFile(path, []byte(dir), 0o644)
}
