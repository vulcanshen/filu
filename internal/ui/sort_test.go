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
	defer func() { sortChain = nil }()
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

	// default: dirs first (name asc), then files name asc (case-insensitive)
	sortChain = nil
	items := append([]fileItem(nil), base...)
	sortItems(items)
	if got := names(items); got[0] != "dirA" || got[1] != "dirB" || got[2] != "Alpha.go" || got[len(got)-1] != "zeta.txt" {
		t.Errorf("default order wrong: %v", got)
	}

	// size descending (files): alpha(30) > mid(20) > zeta(10); dirs still first
	sortChain = []sortRule{{sortSize, false}}
	items = append([]fileItem(nil), base...)
	sortItems(items)
	if got := names(items)[2:]; got[0] != "Alpha.go" || got[1] != "mid.md" || got[2] != "zeta.txt" {
		t.Errorf("size desc wrong: %v", names(items))
	}

	// mtime ascending (files): Alpha(1) < mid(2) < zeta(3)
	sortChain = []sortRule{{sortMtime, true}}
	items = append([]fileItem(nil), base...)
	sortItems(items)
	if got := names(items)[2:]; got[0] != "Alpha.go" || got[1] != "mid.md" || got[2] != "zeta.txt" {
		t.Errorf("mtime asc wrong: %v", names(items))
	}
}

func TestSortChainSetUnset(t *testing.T) {
	defer func() { sortChain = nil }()
	sortChain = nil
	sortChainSet(sortSize, true)
	sortChainSet(sortName, false)
	if len(sortChain) != 2 {
		t.Fatalf("want 2 tiers, got %+v", sortChain)
	}
	sortChainSet(sortSize, false) // upsert direction, not a new tier
	if len(sortChain) != 2 || sortChain[0].asc {
		t.Errorf("upsert should flip size, not append: %+v", sortChain)
	}
	sortChainUnset(sortSize)
	if len(sortChain) != 1 || sortChain[0].col != sortName {
		t.Errorf("unset size should leave name: %+v", sortChain)
	}
}

func TestSortPickerFlow(t *testing.T) {
	defer func() { sortChain = nil }()
	sortChain = nil
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "a.txt"))
	m := AppModel{sortMenu: newSortMenu(), taskCh: make(chan landMsg, 1), watched: map[string]bool{}}
	m.tabs = []listModel{newList(dir)}

	m.openSortColumnPicker()
	if m.sortStep != sortStepColumn {
		t.Fatal("picker should open on the column step")
	}

	m.advanceSortFlow("m") // pick Modified → direction step
	if m.sortStep != sortStepDirection || m.sortFlowCol != sortMtime {
		t.Fatalf("after column pick: step=%v col=%v", m.sortStep, m.sortFlowCol)
	}
	m.advanceSortFlow("d") // Descending → chain=[mtime desc], loop to column
	if len(sortChain) != 1 || sortChain[0].col != sortMtime || sortChain[0].asc {
		t.Fatalf("chain after mtime desc: %+v", sortChain)
	}
	if m.sortStep != sortStepColumn {
		t.Error("should loop back to the column step")
	}

	m.advanceSortFlow("n") // add Name asc as a second tier
	m.advanceSortFlow("a")
	if len(sortChain) != 2 || sortChain[1].col != sortName || !sortChain[1].asc {
		t.Fatalf("chain after adding name asc: %+v", sortChain)
	}

	m.advanceSortFlow("m") // unset Modified
	m.advanceSortFlow("u")
	if len(sortChain) != 1 || sortChain[0].col != sortName {
		t.Fatalf("chain after unset modified: %+v", sortChain)
	}

	m.advanceSortFlow("r") // Reset
	if len(sortChain) != 0 {
		t.Errorf("reset should clear the chain: %+v", sortChain)
	}
}

func TestSortPersistRoundtrip(t *testing.T) {
	defer func() { sortChain = nil }()
	sortChain = []sortRule{{sortSize, false}, {sortName, true}}

	var m AppModel
	m.tabs = []listModel{{dir: "/tmp"}}
	data, err := yaml.Marshal(m.snapshotState())
	if err != nil {
		t.Fatal(err)
	}

	sortChain = nil // ensure applyState is what restores it
	var st sessionState
	if err := yaml.Unmarshal(data, &st); err != nil {
		t.Fatal(err)
	}
	var got AppModel
	got.applyState(st)
	if len(sortChain) != 2 || sortChain[0].col != sortSize || sortChain[0].asc ||
		sortChain[1].col != sortName || !sortChain[1].asc {
		t.Errorf("sort chain not restored: %+v", sortChain)
	}
}
