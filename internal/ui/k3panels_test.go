package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

var enterKey = tea.KeyMsg{Type: tea.KeyEnter}

// panel3Model is f4Model with a second directory holding a file, for marks and
// favorites to point at, and focus on panel [3]'s given tab.
func panel3Model(t *testing.T, tab int) (AppModel, string) {
	t.Helper()
	m := f4Model(t)
	m.tabs = m.tabs[:1]
	other := t.TempDir()
	if err := os.WriteFile(filepath.Join(other, "b.txt"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(other, "a0.txt"), nil, 0o644); err != nil { // sorts first, so landing on b.txt is a real move
		t.Fatal(err)
	}
	m.focus, m.marksTab = panelMarks, tab
	return m, other
}

// tdp K3: Enter on panel [2] opens the scrollable view, as y does.
func TestK3EnterOnPreviewOpensViewport(t *testing.T) {
	m := f4Model(t)
	m.focus = panelDetail
	m.refreshPreview()
	m = press(t, m, enterKey)
	if !m.detailYank.owns() {
		t.Error("Enter on the preview should open the scrollable view")
	}
}

// tdp K3 (2026-09-28): Enter on a mark shows the file in [1] — in the tab
// already at its directory, else a new tab — with the cursor on it.
func TestK3EnterOnMarkShowsFile(t *testing.T) {
	m, other := panel3Model(t, 0)
	m.marks.items = []string{filepath.Join(other, "b.txt")}

	m = press(t, m, enterKey) // no tab there yet → a new one
	if len(m.tabs) != 2 || m.cur().dir != other || m.focus != panelList {
		t.Fatalf("Enter on a mark should open its dir in a new tab and focus [1]: %d tabs, at %q, focus %d",
			len(m.tabs), m.cur().dir, m.focus)
	}
	if it := m.cur().cursorItem(); it.name != "b.txt" {
		t.Errorf("the cursor should land on the marked file, on %q", it.name)
	}

	m.tab, m.focus = 0, panelMarks
	m = press(t, m, enterKey) // the tab exists now → switch to it, no third tab
	if len(m.tabs) != 2 || m.tab != 1 {
		t.Errorf("Enter should reuse the tab already there: %d tabs, active %d", len(m.tabs), m.tab)
	}
}

// tdp K3 (2026-09-28): Favorites work the same way as marks.
func TestK3EnterOnFavoriteShowsDir(t *testing.T) {
	m, other := panel3Model(t, 2)
	m.places.pinned = []place{{path: other, label: "other"}}
	m = press(t, m, enterKey)
	if len(m.tabs) != 2 || m.cur().dir != other || m.focus != panelList {
		t.Errorf("Enter on a favorite should open it in a new tab: %d tabs, at %q", len(m.tabs), m.cur().dir)
	}
	m.focus = panelMarks
	m = press(t, m, enterKey)
	if len(m.tabs) != 2 {
		t.Errorf("a second Enter should reuse the tab, got %d tabs", len(m.tabs))
	}
}

// With every tab in use and none at the directory, Enter says so and opens
// nothing.
func TestK3EnterAtTabLimitToasts(t *testing.T) {
	m, other := panel3Model(t, 2)
	for len(m.tabs) < maxTabs {
		m.addTab(m.tabs[0].dir)
	}
	m.focus = panelMarks
	m.places.pinned = []place{{path: other, label: "other"}}
	m = press(t, m, enterKey)
	if len(m.tabs) != maxTabs || !m.toast.owns() || !strings.Contains(m.toast.message, "tabs are in use") {
		t.Errorf("at the limit Enter should toast, not open: %d tabs, toast %q", len(m.tabs), m.toast.message)
	}
}

// tdp K3 (2026-09-28): Enter on a task takes the active tab to where it landed.
func TestK3EnterOnTaskGoesToDestination(t *testing.T) {
	m, other := panel3Model(t, 1)
	m.tasks = []landTask{{id: 1, destPath: other, status: taskDone}}
	m = press(t, m, enterKey)
	if m.cur().dir != other || m.focus != panelList || len(m.tabs) != 1 {
		t.Errorf("Enter on a task should move the active tab there: at %q, focus %d, %d tabs", m.cur().dir, m.focus, len(m.tabs))
	}
}

// tdp K6: each panel's key reference says what Enter does there.
func TestK3KeyRefNamesEnter(t *testing.T) {
	for _, tc := range []struct {
		focus panelID
		tab   int
		want  string
	}{
		{panelList, 0, "on a file, its details"},
		{panelDetail, 0, "scrollable view"},
		{panelMarks, 0, "show the file in [1]"},
		{panelMarks, 1, "where the task landed"},
		{panelMarks, 2, "show the directory in [1]"},
	} {
		m := AppModel{focus: tc.focus, marksTab: tc.tab, tabs: []listModel{{dir: "/tmp"}}}
		_, rows := m.panelKeyRef()
		found := false
		for _, r := range rows {
			if r.key == "Enter" && strings.Contains(r.desc, tc.want) {
				found = true
			}
		}
		if !found {
			t.Errorf("panel %d tab %d: the key reference should say Enter does %q", tc.focus, tc.tab, tc.want)
		}
	}
}
