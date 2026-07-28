package ui

import (
	"path/filepath"
	"testing"
	"time"

	"gopkg.in/yaml.v3"
)

func names(items []fileItem) []string {
	out := make([]string, len(items))
	for i, it := range items {
		out[i] = it.name
	}
	return out
}

func TestSortItemsDirsFirstAndChain(t *testing.T) {
	mk := func(name string, dir bool, size int64, minute int) fileItem {
		return fileItem{name: name, isDir: dir, size: size, mtime: time.Unix(int64(minute)*60, 0)}
	}
	base := []fileItem{
		mk("zeta.txt", false, 10, 3),
		mk("Alpha.go", false, 30, 1),
		mk("dirB", true, 0, 5),
		mk("mid.md", false, 20, 2),
		mk("dirA", true, 0, 4),
	}

	// default (nil chain): dirs first (name asc), then files name asc (ci)
	items := append([]fileItem(nil), base...)
	sortItems(items, nil)
	if got := names(items); got[0] != "dirA" || got[1] != "dirB" || got[2] != "Alpha.go" || got[len(got)-1] != "zeta.txt" {
		t.Errorf("default order wrong: %v", got)
	}

	// size descending (files): alpha(30) > mid(20) > zeta(10); dirs still first
	items = append([]fileItem(nil), base...)
	sortItems(items, []sortRule{{sortSize, false}})
	if got := names(items)[2:]; got[0] != "Alpha.go" || got[1] != "mid.md" || got[2] != "zeta.txt" {
		t.Errorf("size desc wrong: %v", names(items))
	}

	// mtime ascending (files): Alpha(1) < mid(2) < zeta(3)
	items = append([]fileItem(nil), base...)
	sortItems(items, []sortRule{{sortMtime, true}})
	if got := names(items)[2:]; got[0] != "Alpha.go" || got[1] != "mid.md" || got[2] != "zeta.txt" {
		t.Errorf("mtime asc wrong: %v", names(items))
	}
}

func TestSortPerDirSetUnset(t *testing.T) {
	defer func() { sortByDir = map[string][]sortRule{} }()
	sortByDir = map[string][]sortRule{}
	d := "/tmp/proj"

	setSortFor(d, sortSize, true)
	setSortFor(d, sortName, false)
	if len(sortByDir[d]) != 2 {
		t.Fatalf("want 2 tiers, got %+v", sortByDir[d])
	}
	setSortFor(d, sortSize, false) // upsert direction, not a new tier
	if len(sortByDir[d]) != 2 || sortByDir[d][0].asc {
		t.Errorf("upsert should flip size, not append: %+v", sortByDir[d])
	}
	unsetSortFor(d, sortSize)
	if len(sortByDir[d]) != 1 || sortByDir[d][0].col != sortName {
		t.Errorf("unset size should leave name: %+v", sortByDir[d])
	}
	// unsetting the last tier drops the whole entry (nothing set == default).
	unsetSortFor(d, sortName)
	if _, ok := sortByDir[d]; ok {
		t.Errorf("an empty chain should drop the entry, got %+v", sortByDir[d])
	}
	// a different dir is independent.
	setSortFor("/tmp/other", sortMtime, true)
	if sortByDir[d] != nil {
		t.Errorf("%q must stay unset, got %+v", d, sortByDir[d])
	}
}

func TestSortPickerFlow(t *testing.T) {
	defer func() { sortByDir = map[string][]sortRule{} }()
	sortByDir = map[string][]sortRule{}
	dir := t.TempDir()
	statePathOverride = filepath.Join(dir, "state.yaml") // don't pollute the real state
	defer func() { statePathOverride = "" }()
	writeFile(t, filepath.Join(dir, "a.txt"))
	m := AppModel{sortMenu: newSortMenu(), taskCh: make(chan landMsg, 1), watched: map[string]bool{}}
	m.tabs = []listModel{newList(dir)}
	key := cleanDir(dir)

	m.openSortColumnPicker()
	if m.sortStep != sortStepColumn {
		t.Fatal("picker should open on the column step")
	}

	m.advanceSortFlow("m") // pick Modified → direction step
	if m.sortStep != sortStepDirection || m.sortFlowCol != sortMtime {
		t.Fatalf("after column pick: step=%v col=%v", m.sortStep, m.sortFlowCol)
	}
	m.advanceSortFlow("d") // Descending → this dir's chain=[mtime desc], loop to column
	if got := sortByDir[key]; len(got) != 1 || got[0].col != sortMtime || got[0].asc {
		t.Fatalf("chain after mtime desc: %+v", got)
	}
	if m.sortStep != sortStepColumn {
		t.Error("should loop back to the column step")
	}

	m.advanceSortFlow("n") // add Name asc as a second tier
	m.advanceSortFlow("a")
	if got := sortByDir[key]; len(got) != 2 || got[1].col != sortName || !got[1].asc {
		t.Fatalf("chain after adding name asc: %+v", got)
	}

	m.advanceSortFlow("m") // unset Modified
	m.advanceSortFlow("u")
	if got := sortByDir[key]; len(got) != 1 || got[0].col != sortName {
		t.Fatalf("chain after unset modified: %+v", got)
	}

	m.advanceSortFlow("r") // Reset
	if _, ok := sortByDir[key]; ok {
		t.Errorf("reset should clear the dir's chain: %+v", sortByDir[key])
	}
}

func TestSortPersistRoundtrip(t *testing.T) {
	defer func() { sortByDir = map[string][]sortRule{} }()
	sortByDir = map[string][]sortRule{
		"/tmp/a": {{sortSize, false}, {sortName, true}},
		"/tmp/b": {{sortMtime, true}},
	}

	var m AppModel
	m.tabs = []listModel{{dir: "/tmp"}}
	data, err := yaml.Marshal(m.snapshotState())
	if err != nil {
		t.Fatal(err)
	}

	sortByDir = map[string][]sortRule{} // ensure applyState is what restores it
	var st sessionState
	if err := yaml.Unmarshal(data, &st); err != nil {
		t.Fatal(err)
	}
	var got AppModel // no tabs → applyState skips the tab-0 reload, still restores sorts
	got.applyState(st)
	if a := sortByDir["/tmp/a"]; len(a) != 2 || a[0].col != sortSize || a[0].asc || a[1].col != sortName || !a[1].asc {
		t.Errorf("/tmp/a chain not restored: %+v", a)
	}
	if b := sortByDir["/tmp/b"]; len(b) != 1 || b[0].col != sortMtime || !b[0].asc {
		t.Errorf("/tmp/b chain not restored: %+v", b)
	}
}
