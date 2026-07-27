package ui

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func TestReadEntriesHiddenCount(t *testing.T) {
	dir := t.TempDir()
	for _, n := range []string{"a", "b", ".dot1", ".dot2"} {
		mustWrite(t, filepath.Join(dir, n))
	}
	// showHidden=false: dotfiles are excluded from items but still counted.
	vis, hidden, err := readEntries(dir, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(vis) != 2 || hidden != 2 {
		t.Errorf("showHidden=false: got %d items, %d hidden; want 2, 2", len(vis), hidden)
	}
	// showHidden=true: dotfiles included, count unchanged.
	all, hidden2, _ := readEntries(dir, true)
	if len(all) != 4 || hidden2 != 2 {
		t.Errorf("showHidden=true: got %d items, %d hidden; want 4, 2", len(all), hidden2)
	}
}

func TestStatusBarRender(t *testing.T) {
	m := minModel()
	m.width, m.height = 100, 30
	m.tabs = []listModel{newList(t.TempDir())}
	m.tab = 0

	// The bar is parked: its height is reserved but its content is blank for now
	// (per-item perms/mtime moved into the list columns).
	bar := m.statusBar(m.width)
	if got := dispWidth(bar); got != m.width {
		t.Errorf("statusBar width = %d, want %d", got, m.width)
	}
	if plain := strings.TrimSpace(ansi.Strip(bar)); plain != "" {
		t.Errorf("parked statusBar should be blank, got %q", plain)
	}
}
