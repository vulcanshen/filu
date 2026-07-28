package ui

import (
	"testing"

	"gopkg.in/yaml.v3"
)

// FILU_STATE redirects state I/O (used by the demo tapes to isolate state); the
// test-only override still wins over it.
func TestStatePathEnvOverride(t *testing.T) {
	old := statePathOverride
	statePathOverride = ""
	defer func() { statePathOverride = old }()

	t.Setenv("FILU_STATE", "/tmp/filu-demo/state.yaml")
	if got, ok := stateFilePath(); !ok || got != "/tmp/filu-demo/state.yaml" {
		t.Errorf("FILU_STATE not honoured: got %q ok=%v", got, ok)
	}

	statePathOverride = "/tmp/override/state.yaml" // test override beats the env
	if got, _ := stateFilePath(); got != "/tmp/override/state.yaml" {
		t.Errorf("statePathOverride should win over FILU_STATE, got %q", got)
	}
}

func TestSnapshotApplyRoundtrip(t *testing.T) {
	var m AppModel
	m.tabs = []listModel{{dir: "/tmp"}, {dir: "/usr", cursor: 2}, {dir: "/etc"}}
	m.tab = 1
	m.focus = panelTasks // the [4] panel — verifies the restore range extends to it
	m.marks.items = []string{"/a", "/b"}
	m.places = placesModel{pinned: []place{{path: "/home/me/proj", icon: iconPin, label: "proj"}}}

	data, err := yaml.Marshal(m.snapshotState())
	if err != nil {
		t.Fatal(err)
	}
	var st sessionState
	if err := yaml.Unmarshal(data, &st); err != nil {
		t.Fatal(err)
	}

	got := AppModel{}                     // like New(): pins restored onto an empty places
	got.tabs = []listModel{{dir: "/cwd"}} // like New(): one CWD tab, extras restored onto it
	got.applyState(st)

	if got.focus != panelTasks {
		t.Errorf("focus not restored: %d", got.focus)
	}
	if got.tab != 0 {
		t.Errorf("tab [0] should always be active on launch, got tab=%d", got.tab)
	}
	if len(got.marks.items) != 2 {
		t.Errorf("carry not restored: %v", got.marks.items)
	}
	if got.tabs[1].dir != "/usr" || got.tabs[1].cursor != 2 {
		t.Errorf("tab[1] not restored: dir=%q cursor=%d", got.tabs[1].dir, got.tabs[1].cursor)
	}
	if len(got.places.pinned) != 1 || got.places.pinned[0].path != "/home/me/proj" {
		t.Errorf("pinned dir not restored: %+v", got.places.pinned)
	}
}

func TestTaskStatusPersist(t *testing.T) {
	var m AppModel
	m.tasks = []landTask{
		{id: 1, action: "cp", dest: "d", total: 3, status: taskRunning},
		{id: 2, action: "mv", dest: "e", total: 1, status: taskDone},
		{id: 3, action: "cp", dest: "f", total: 2, status: taskError},
	}
	data, err := yaml.Marshal(m.snapshotState())
	if err != nil {
		t.Fatal(err)
	}
	var st sessionState
	if err := yaml.Unmarshal(data, &st); err != nil {
		t.Fatal(err)
	}

	var got AppModel
	got.applyState(st)
	if len(got.tasks) != 3 {
		t.Fatalf("tasks not restored: %d", len(got.tasks))
	}
	if got.tasks[0].status != taskPending {
		t.Errorf("a running task should restore as pending (interrupted), got %v", got.tasks[0].status)
	}
	if got.tasks[1].status != taskDone {
		t.Errorf("done should restore as done, got %v", got.tasks[1].status)
	}
	if got.tasks[2].status != taskError {
		t.Errorf("error should restore as error, got %v", got.tasks[2].status)
	}
	if got.nextTaskID != 3 {
		t.Errorf("nextTaskID should be the max loaded id 3, got %d", got.nextTaskID)
	}
}
