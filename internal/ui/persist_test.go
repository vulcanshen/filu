package ui

import (
	"testing"

	"gopkg.in/yaml.v3"
)

func TestSnapshotApplyRoundtrip(t *testing.T) {
	var m AppModel
	m.tabs[0] = listModel{dir: "/tmp"}
	m.tabs[1] = listModel{dir: "/usr", cursor: 2}
	m.tabs[2] = listModel{dir: "/etc"}
	m.tab = 1
	m.focus = panelDetail
	m.detail = tabMeta
	m.carry.items = []string{"/a", "/b"}
	m.places.pinned = []place{{path: "/home/me/proj", icon: iconPin, label: "proj"}}

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

	if got.tab != 1 || got.focus != panelDetail || got.detail != tabMeta {
		t.Errorf("scalars: tab=%d focus=%d detail=%d", got.tab, got.focus, got.detail)
	}
	if len(got.carry.items) != 2 {
		t.Errorf("carry not restored: %v", got.carry.items)
	}
	if len(got.places.pinned) != 1 || got.places.pinned[0].path != "/home/me/proj" {
		t.Errorf("pinned not restored: %v", got.places.pinned)
	}
	if got.tabs[1].dir != "/usr" {
		t.Errorf("tab[1] dir not restored: %q", got.tabs[1].dir)
	}
}
