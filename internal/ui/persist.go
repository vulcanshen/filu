package ui

import (
	"errors"
	"io/fs"
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
	if d := os.Getenv("FILU__STATE"); d != "" { // a state directory (demo recordings / isolated runs, tdp E2, E3)
		return filepath.Join(d, "state.yaml"), true
	}
	dir, ok := filuConfigDir()
	if !ok {
		return "", false
	}
	return filepath.Join(dir, "state.yaml"), true
}

// dataState is filu's data (tdp E3): what is gone for good when it is deleted
// and worth a backup — the favorites and the task log. It lives in
// ~/.filu/data.yaml (FILU__DATA names another directory), apart from the state,
// which only brings back where you were.
type dataState struct {
	Favorites []string        `yaml:"favorites,omitempty"`
	Tasks     []persistedTask `yaml:"tasks,omitempty"`
}

var dataPathOverride string // tests redirect data I/O here

func dataFilePath() (string, bool) {
	if dataPathOverride != "" {
		return dataPathOverride, true
	}
	if d := os.Getenv("FILU__DATA"); d != "" { // a data directory (tdp E2, E3)
		return filepath.Join(d, "data.yaml"), true
	}
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return "", false
	}
	return filepath.Join(home, ".filu", "data.yaml"), true
}

// readYAML reads path into v: found false when there is no such file, err
// when it cannot be read or makes no sense.
func readYAML(path string, v any) (found bool, err error) {
	b, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, yaml.Unmarshal(b, v)
}

// writeYAML writes v to path, making its directory.
func writeYAML(path string, v any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	b, err := yaml.Marshal(v)
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}

// loadState reads the saved session and the data: ok false when there is
// neither (the first run), err when one cannot be read or makes no sense.
// moved reports favorites or tasks that were still in state.yaml, where
// versions before the data directory kept them: the next save moves them.
// When data.yaml has its own, those in state.yaml stay there (leftInState).
func loadState() (st sessionState, ok bool, moved bool, err error) {
	leftInState = dataState{}
	if path, found := stateFilePath(); found {
		if ok, err = readYAML(path, &st); err != nil {
			return sessionState{}, false, false, err
		}
	}
	path, found := dataFilePath()
	if !found {
		return st, ok, false, nil
	}
	var d dataState
	has, err := readYAML(path, &d)
	switch {
	case err != nil:
		return sessionState{}, false, false, err
	case has:
		if len(st.Pinned) > 0 || len(st.Tasks) > 0 {
			leftInState = dataState{Favorites: st.Pinned, Tasks: st.Tasks}
		}
		st.Pinned, st.Tasks = d.Favorites, d.Tasks
		return st, true, false, nil
	}
	return st, ok, len(st.Pinned) > 0 || len(st.Tasks) > 0, nil
}

// leftInState are the favorites and tasks state.yaml still holds while
// data.yaml has its own: left there for the user to put together, not
// deleted (tdp E3), and so written back with every save.
var leftInState dataState

// saveState writes the session and the data and reports a failure, so the
// caller can show it (tdp F5): a save that silently fails loses tabs, marks,
// favorites and sorts. No path (no home dir) is not a failure — there is
// nowhere to save to.
func saveState(st sessionState) error {
	if path, ok := dataFilePath(); ok {
		if err := writeYAML(path, dataState{Favorites: st.Pinned, Tasks: st.Tasks}); err != nil {
			return err
		}
		st.Pinned, st.Tasks = leftInState.Favorites, leftInState.Tasks // in data.yaml, not in the state
	}
	if path, ok := stateFilePath(); ok {
		return writeYAML(path, st)
	}
	return nil
}
