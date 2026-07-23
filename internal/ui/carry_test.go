package ui

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCarryPick(t *testing.T) {
	m := carryModel{items: []string{"/a", "/b", "/c"}}
	if len(m.landSet()) != 3 {
		t.Errorf("no pick → land everything (3), got %d", len(m.landSet()))
	}
	m.cursor = 1
	m.togglePick()
	if !m.picked["/b"] {
		t.Fatal("togglePick should pick /b")
	}
	if set := m.landSet(); len(set) != 1 || !set["/b"] {
		t.Errorf("landSet should be just /b, got %v", set)
	}
	m.togglePick()
	if m.picked["/b"] {
		t.Error("second togglePick should unpick /b")
	}
}

func TestCarryLandPickedOnly(t *testing.T) {
	src, dst := t.TempDir(), t.TempDir()
	a := filepath.Join(src, "a.txt")
	b := filepath.Join(src, "b.txt")
	for _, p := range []string{a, b} {
		if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	m := carryModel{items: []string{a, b}}
	m.cursor = 0
	m.togglePick() // pick only a
	m.land(dst, false)

	if _, err := os.Stat(filepath.Join(dst, "a.txt")); err != nil {
		t.Error("picked a.txt should be copied")
	}
	if _, err := os.Stat(filepath.Join(dst, "b.txt")); err == nil {
		t.Error("unpicked b.txt should not be copied")
	}
	if len(m.items) != 2 {
		t.Errorf("copy keeps the bucket, got %d items", len(m.items))
	}
}
