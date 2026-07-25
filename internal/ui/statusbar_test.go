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
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "a.go"))
	mustWrite(t, filepath.Join(dir, ".hidden"))

	m := minModel()
	m.width, m.height = 100, 30
	m.tabs = []listModel{newList(dir)}
	m.tab = 0

	bar := m.statusBar(m.width)
	if got := dispWidth(bar); got != m.width {
		t.Errorf("statusBar width = %d, want %d", got, m.width)
	}
	plain := ansi.Strip(bar)
	// The dir stat (perm) plus the live item / hidden counts must all show.
	for _, want := range []string{"drwx", "1 items", "1 hidden", "free"} {
		if !strings.Contains(plain, want) {
			t.Errorf("statusBar missing %q:\n%s", want, plain)
		}
	}
}
