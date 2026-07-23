package ui

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// sessionState is what filu restores on the next launch (IDEA.md: "where you
// were is where you restart") — the 3 tab dirs + cursors, active tab, focus,
// detail tab, carry bucket, and pinned places.
type sessionState struct {
	Tabs   []tabState `yaml:"tabs"`
	Tab    int        `yaml:"tab"`
	Focus  int        `yaml:"focus"`
	Detail int        `yaml:"detail"`
	Carry  []string   `yaml:"carry,omitempty"`
	Pinned []string   `yaml:"pinned,omitempty"`
}

type tabState struct {
	Dir    string `yaml:"dir"`
	Cursor int    `yaml:"cursor"`
}

func stateFilePath() (string, bool) {
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
		Tab:    m.tab,
		Focus:  int(m.focus),
		Detail: int(m.detail),
		Carry:  m.carry.items,
	}
	for _, t := range m.tabs {
		st.Tabs = append(st.Tabs, tabState{Dir: t.dir, Cursor: t.cursor})
	}
	for _, p := range m.places.pinned {
		st.Pinned = append(st.Pinned, p.path)
	}
	return st
}

// applyState restores a saved session onto a freshly-built model.
func (m *AppModel) applyState(st sessionState) {
	for i := 0; i < len(m.tabs) && i < len(st.Tabs); i++ {
		if st.Tabs[i].Dir == "" {
			continue
		}
		m.tabs[i] = newList(st.Tabs[i].Dir)
		m.tabs[i].cursor = st.Tabs[i].Cursor
		m.tabs[i].clampCursor()
	}
	if st.Tab >= 0 && st.Tab < len(m.tabs) {
		m.tab = st.Tab
	}
	if st.Focus >= int(panelPin) && st.Focus <= int(panelCarry) {
		m.focus = panelID(st.Focus)
	}
	if st.Detail == int(tabPreview) || st.Detail == int(tabMeta) {
		m.detail = detailTab(st.Detail)
	}
	m.carry.items = st.Carry
	for _, p := range st.Pinned {
		m.places.pinned = append(m.places.pinned, place{label: filepath.Base(p), path: p, icon: iconPin})
	}
}
