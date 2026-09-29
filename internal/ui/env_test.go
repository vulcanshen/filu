//go:build darwin || linux

package ui

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// tdp D6 (v0.1.21): filu reads its variables as FILU__NAME — the app name, two
// underscores, the name — and the old single-underscore names are not read any
// more (renamed without a fallback). The names are written out here, not taken
// from the constants under test.
func TestEnvNamesFollowTheFamily(t *testing.T) {
	// cd-on-quit: the file filu writes the chosen directory to.
	dirFile := filepath.Join(t.TempDir(), "cwd")
	t.Setenv("FILU_LAST_DIR_FILE", dirFile)
	writeLastDir("/old")
	if _, err := os.Stat(dirFile); err == nil {
		t.Error("the old FILU_LAST_DIR_FILE should not be read")
	}
	t.Setenv("FILU__LAST_DIR_FILE", dirFile)
	writeLastDir("/new")
	if got, _ := os.ReadFile(dirFile); string(got) != "/new" {
		t.Errorf("FILU__LAST_DIR_FILE got %q, want /new", got)
	}

	// The icon width override (tests run without a terminal, so no probe).
	defer restoreIconCells(iconCells)
	iconCells = 1
	t.Setenv("FILU_ICON_WIDTH", "2")
	DetectIconWidth()
	if iconCells != 1 {
		t.Error("the old FILU_ICON_WIDTH should not be read")
	}
	t.Setenv("FILU__ICON_WIDTH", "2")
	DetectIconWidth()
	if iconCells != 2 {
		t.Errorf("FILU__ICON_WIDTH=2 should set two cells, got %d", iconCells)
	}

	// The forced repaint on navigation.
	repaints := func() bool {
		model, cmd := f4Model(t).Update(runes("j"))
		_ = model
		if cmd == nil {
			return false
		}
		for _, msg := range batchMsgs(cmd) {
			if reflect.DeepEqual(msg, tea.ClearScreen()) {
				return true
			}
		}
		return false
	}
	t.Setenv("FILU_REPAINT", "1")
	if repaints() {
		t.Error("the old FILU_REPAINT should not be read")
	}
	t.Setenv("FILU__REPAINT", "1")
	if !repaints() {
		t.Error("FILU__REPAINT=1 should repaint on navigation")
	}
}
