package ui

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// sessionState is what filu restores on the next launch (IDEA.md: "where you
// were is where you restart") — the tabs the user created beyond the CWD tab
// (dirs + cursors), focus, carry bucket, pinned dirs, and the sort chain. Tab [0]
// always reopens at the CWD and is the active tab on launch, so the first tab's
// state and the active-tab index are deliberately NOT persisted.
type sessionState struct {
	Tabs   []tabState      `yaml:"tabs"` // the tabs created beyond the CWD tab [0]
	Focus  int             `yaml:"focus"`
	Carry  []string        `yaml:"carry,omitempty"`
	Pinned []string        `yaml:"pinned,omitempty"`
	Tasks  []persistedTask `yaml:"tasks,omitempty"`
	Sort   []sortRuleYAML  `yaml:"sort,omitempty"`
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
	ID     int      `yaml:"id"`
	Action string   `yaml:"action"`
	Dest   string   `yaml:"dest"`
	Path   string   `yaml:"path"`
	Srcs   []string `yaml:"srcs,omitempty"`
	Total  int      `yaml:"total"`
	Status string   `yaml:"status"` // "done" / "undone" / "error"
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
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", false
	}
	return filepath.Join(dir, "filu", "state.yaml"), true
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
		Focus: int(m.focus),
		Carry: m.carry.items,
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
			Status: taskStatusString(t.status),
		})
	}
	for _, r := range sortChain {
		st.Sort = append(st.Sort, sortRuleYAML{Col: int(r.col), Asc: r.asc})
	}
	return st
}

// applyState restores a saved session onto a freshly-built model.
func (m *AppModel) applyState(st sessionState) {
	sortChain = nil // restore the sort before tabs reload so they sort correctly
	for _, r := range st.Sort {
		sortChain = append(sortChain, sortRule{col: sortCol(r.Col), asc: r.Asc})
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
	if st.Focus >= int(panelList) && st.Focus <= int(panelMeta) {
		m.focus = panelID(st.Focus)
	}
	m.carry.items = st.Carry
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
			total: pt.Total, done: pt.Total, status: status,
		})
		if pt.ID > m.nextTaskID {
			m.nextTaskID = pt.ID
		}
	}
}
