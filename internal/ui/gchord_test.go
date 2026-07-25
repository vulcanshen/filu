package ui

import (
	"os"
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// TestGgChordJumpsToTop covers the vim g-prefix chord on the main panels: a lone
// g arms and waits (no move), a second g jumps to the top, and a non-g key after
// g cancels the chord and runs on its own.
func TestGgChordJumpsToTop(t *testing.T) {
	dir := t.TempDir()
	for _, n := range []string{"a", "b", "c", "d"} {
		mustWrite(t, filepath.Join(dir, n))
	}
	m := minModel()
	m.width, m.height = 80, 24
	m.tabs = []listModel{newList(dir)}
	m.tab = 0
	m.cur().cursor = 3 // off the top

	g := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("g")}

	// first g: arm, but do not move
	model, _ := m.Update(g)
	m = model.(AppModel)
	if !m.pendingG {
		t.Fatal("first g should arm the g-chord")
	}
	if m.cur().cursor != 3 {
		t.Errorf("first g must not move the cursor, got %d", m.cur().cursor)
	}

	// second g: gg → jump to the top and disarm
	model, _ = m.Update(g)
	m = model.(AppModel)
	if m.pendingG {
		t.Error("gg should clear the pending state")
	}
	if m.cur().cursor != 0 {
		t.Errorf("gg should jump to the top, cursor=%d", m.cur().cursor)
	}

	// g then a non-g key cancels: the key runs on its own, no jump-to-top
	m.cur().cursor = 3
	model, _ = m.Update(g)
	m = model.(AppModel)
	model, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	m = model.(AppModel)
	if m.pendingG {
		t.Error("a non-g second key should clear the pending state")
	}
	if m.cur().cursor != 2 {
		t.Errorf("g then k should move up one (not jump to top), cursor=%d", m.cur().cursor)
	}
}

// TestGoChordOpensGoto covers the `go` chord: a lone g arms, then o opens the
// goto finder rooted at $HOME in dirs-only, name (not content) mode.
func TestGoChordOpensGoto(t *testing.T) {
	m := minModel()
	m.width, m.height = 80, 24

	g := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("g")}
	o := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("o")}

	model, _ := m.Update(g)
	m = model.(AppModel)
	if !m.pendingG {
		t.Fatal("first g should arm the chord")
	}
	if m.search.isActive() {
		t.Fatal("g alone must not open goto")
	}

	model, _ = m.Update(o)
	m = model.(AppModel)
	if m.pendingG {
		t.Error("`go` should clear the pending state")
	}
	if !m.search.isActive() {
		t.Fatal("`go` should open the goto finder")
	}
	if !m.search.dirsOnly || m.search.byContent {
		t.Errorf("goto should be dirs-only name mode, dirsOnly=%v byContent=%v", m.search.dirsOnly, m.search.byContent)
	}
	if home, _ := os.UserHomeDir(); home != "" && m.search.root != home {
		t.Errorf("goto root = %q, want $HOME %q", m.search.root, home)
	}
}

// TestBracketHotkeyChord checks the Space-menu label renders a chord key inside
// the label ("[go]to"), still handles single letters ("[S]ort"), and leaves a
// multi-char key that isn't in the label plain ("Jump").
func TestBracketHotkeyChord(t *testing.T) {
	cases := []struct{ label, key, want string }{
		{"Goto", "go", "[go]to"},
		{"Sort", "S", "[S]ort"},
		{"Search", "/", "[/] Search"},
		{"Jump", "enter", "Jump"},
	}
	for _, c := range cases {
		if got := bracketHotkey(c.label, c.key); got != c.want {
			t.Errorf("bracketHotkey(%q,%q) = %q, want %q", c.label, c.key, got, c.want)
		}
	}
}
