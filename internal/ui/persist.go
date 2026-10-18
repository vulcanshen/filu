package ui

import (
	"os"
	"path/filepath"
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
	if d := os.Getenv("FILU__STATE"); d != "" { // a state directory (demo recordings / isolated runs, tdp D6)
		return filepath.Join(d, "state.yaml"), true
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

// saveState writes the session and reports a failure, so the caller can show it
// (tdp F5): a save that silently fails loses tabs, marks, favorites and sorts.
// No state path (no home dir) is not a failure — there is nowhere to save to.
func saveState(st sessionState) error {
	path, ok := stateFilePath()
	if !ok {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := yaml.Marshal(st)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
