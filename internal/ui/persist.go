package ui

import (
	"os"
	"path/filepath"
	"sort"
	"time"

	"gopkg.in/yaml.v3"
)

// sessionState is what filu restores on the next launch (IDEA.md: "where you
// were is where you restart") — the tabs the user created beyond the CWD tab
// (dirs + cursors), marks bucket, pinned dirs, and the per-directory sort chains.
// Tab [0] always reopens at the CWD and is the active tab on launch; the first
// tab's state, the active-tab index, and the focused panel are deliberately NOT
// persisted (launch always focuses the list).
type sessionState struct {
	Tabs   []tabState      `yaml:"tabs"`            // the tabs created beyond the CWD tab [0]
	Marks  []string        `yaml:"carry,omitempty"` // yaml tag kept as "carry" so old state.yaml still loads
	Pinned []string        `yaml:"pinned,omitempty"`
	Tasks  []persistedTask `yaml:"tasks,omitempty"`
	Sorts  []dirSortYAML   `yaml:"sorts,omitempty"` // per-directory sort chains
}

// dirSortYAML is one directory's persisted sort chain, keyed by its exact path.
type dirSortYAML struct {
	Path  string         `yaml:"path"`
	Rules []sortRuleYAML `yaml:"rules"`
}

// sortRuleYAML is one persisted sort tier.
type sortRuleYAML struct {
	Col int  `yaml:"col"`
	Asc bool `yaml:"asc"`
}

type tabState struct {
	Dir    string `yaml:"dir"`
	Cursor int    `yaml:"cursor"`
}

// persistedTask is a land task on disk. Status "undone" means it was running
// when the app exited — on the next launch it restores as taskPending.
type persistedTask struct {
	ID     int       `yaml:"id"`
	Action string    `yaml:"action"`
	Dest   string    `yaml:"dest"`
	Path   string    `yaml:"path"`
	Srcs   []string  `yaml:"srcs,omitempty"`
	Total  int       `yaml:"total"`
	Failed int       `yaml:"failed,omitempty"`
	At     time.Time `yaml:"at,omitempty"`
	Status string    `yaml:"status"` // "done" / "undone" / "error"
}

func taskStatusString(s taskStatus) string {
	switch s {
	case taskDone:
		return "done"
	case taskError:
		return "error"
	default: // taskRunning / taskPending are both unfinished on disk
		return "undone"
	}
}

var statePathOverride string // tests redirect state I/O here

func stateFilePath() (string, bool) {
	if statePathOverride != "" {
		return statePathOverride, true
	}
	if p := os.Getenv("FILU_STATE"); p != "" { // redirect state I/O (demo recordings / isolated runs)
		return p, true
	}
	dir, ok := filuConfigDir()
	if !ok {
		return "", false
	}
	return filepath.Join(dir, "state.yaml"), true
}

func loadState() (sessionState, bool) {
	path, ok := stateFilePath()
	if !ok {
		return sessionState{}, false
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return sessionState{}, false
	}
	var st sessionState
	if yaml.Unmarshal(data, &st) != nil {
		return sessionState{}, false
	}
	return st, true
}

// saveState writes best-effort; failures are silent (persistence is a nicety,
// not a guarantee).
func saveState(st sessionState) {
	path, ok := stateFilePath()
	if !ok {
		return
	}
	if os.MkdirAll(filepath.Dir(path), 0o755) != nil {
		return
	}
	data, err := yaml.Marshal(st)
	if err != nil {
		return
	}
	_ = os.WriteFile(path, data, 0o644)
}

// snapshotState captures the current model for persistence.
func (m AppModel) snapshotState() sessionState {
	st := sessionState{
		Marks: m.marks.items,
	}
	for i := 1; i < len(m.tabs); i++ { // tab [0] always reopens at CWD — skip it
		st.Tabs = append(st.Tabs, tabState{Dir: m.tabs[i].dir, Cursor: m.tabs[i].cursor})
	}
	for _, p := range m.places.pinned {
		st.Pinned = append(st.Pinned, p.path)
	}
	for _, t := range m.tasks {
		st.Tasks = append(st.Tasks, persistedTask{
			ID: t.id, Action: t.action, Dest: t.dest, Path: t.destPath, Srcs: t.srcs, Total: t.total,
			Failed: t.failed, At: t.at, Status: taskStatusString(t.status),
		})
	}
	paths := make([]string, 0, len(sortByDir))
	for p := range sortByDir {
		paths = append(paths, p)
	}
	sort.Strings(paths) // deterministic output → stable state.yaml diffs
	for _, p := range paths {
		ds := dirSortYAML{Path: p}
		for _, r := range sortByDir[p] {
			ds.Rules = append(ds.Rules, sortRuleYAML{Col: int(r.col), Asc: r.asc})
		}
		st.Sorts = append(st.Sorts, ds)
	}
	return st
}

// applyState restores a saved session onto a freshly-built model.
func (m *AppModel) applyState(st sessionState) {
	// restore the per-directory sorts before any tab reloads, so they sort right.
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
	// tab [0] (the CWD) was built in New() before the sorts loaded — re-sort it now.
	if len(m.tabs) > 0 {
		m.tabs[0].reloadPreserving()
	}
	for _, ts := range st.Tabs { // restore the tabs the user created beyond the CWD tab
		if ts.Dir == "" || len(m.tabs) >= maxTabs {
			continue
		}
		nl := newList(ts.Dir)
		nl.cursor = ts.Cursor
		nl.clampCursor()
		m.tabs = append(m.tabs, nl)
	}
	// focus is not restored — launch always focuses the list (set in New()).
	m.marks.items = st.Marks
	for _, p := range st.Pinned {
		m.places.pinned = append(m.places.pinned, place{label: filepath.Base(p), path: p, icon: iconPin})
	}
	for _, pt := range st.Tasks { // "undone" tasks were interrupted → pending
		status := taskPending
		switch pt.Status {
		case "done":
			status = taskDone
		case "error":
			status = taskError
		}
		m.tasks = append(m.tasks, landTask{
			id: pt.ID, action: pt.Action, dest: pt.Dest, destPath: pt.Path, srcs: pt.Srcs,
			total: pt.Total, done: pt.Total, failed: pt.Failed, at: pt.At, status: status,
		})
		if pt.ID > m.nextTaskID {
			m.nextTaskID = pt.ID
		}
	}
}
