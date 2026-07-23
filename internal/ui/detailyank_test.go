package ui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func dyRune(s string) tea.KeyMsg { return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)} }

func openYank(lines ...string) detailYank {
	m := newDetailYank()
	m.setSize(80, 24)
	m.open("t", lines)
	m.anim.state = popupOpen // skip the open animation so update() is interactive
	return m
}

func TestDetailYankVisualSelectionText(t *testing.T) {
	m := openYank("hello", "world")

	m, _ = m.update(dyRune("v")) // anchor at (0,0)
	m, _ = m.update(dyRune("l"))
	m, _ = m.update(dyRune("l")) // cursor at (0,2) → select "hel"
	if got := m.selectionText(); got != "hel" {
		t.Errorf("single-line selection = %q, want hel", got)
	}

	m, _ = m.update(dyRune("j")) // extend to (1,?) — multi-line
	if got := m.selectionText(); got == "hel" || got == "" {
		t.Errorf("multi-line selection should span both lines, got %q", got)
	}
}

func TestDetailYankVisualYankCopies(t *testing.T) {
	m := openYank("abcdef")
	m, _ = m.update(dyRune("v"))
	m, _ = m.update(dyRune("l"))
	_, cmd := m.update(dyRune("y"))
	if _, ok := runQuiet(cmd).(clipboardCopiedMsg); !ok {
		t.Error("y in visual mode should copy the selection")
	}
}

func TestDetailYankCopiesAllWithoutSelection(t *testing.T) {
	m := openYank("a", "b", "c")
	if m.full != "a\nb\nc" {
		t.Fatalf("full = %q, want a\\nb\\nc", m.full)
	}
	_, cmd := m.update(dyRune("y"))
	if _, ok := runQuiet(cmd).(clipboardCopiedMsg); !ok {
		t.Error("y without a selection should copy everything")
	}
}

func TestDetailYankEscPeelsVisualThenCloses(t *testing.T) {
	m := openYank("x")
	m, _ = m.update(dyRune("v"))
	if !m.visual {
		t.Fatal("v should enter visual mode")
	}
	m, _ = m.update(tea.KeyMsg{Type: tea.KeyEsc})
	if m.visual {
		t.Error("first Esc should leave visual mode, not close")
	}
	if _, cmd := m.update(tea.KeyMsg{Type: tea.KeyEsc}); cmd == nil {
		t.Error("second Esc should close the viewport")
	}
}

func TestDetailYankCursorClampsToLine(t *testing.T) {
	m := openYank("ab", "wxyz")
	m, _ = m.update(dyRune("$")) // end of line 0 → col 1
	if m.cursorCol != 1 {
		t.Errorf("$ on 'ab' → col %d, want 1", m.cursorCol)
	}
	m, _ = m.update(dyRune("j")) // down to 'wxyz'; col clamps within the new line
	if m.cursorCol > m.lastCol(1) {
		t.Errorf("cursor col %d overruns line 1 (last %d)", m.cursorCol, m.lastCol(1))
	}
}
