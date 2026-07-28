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
// Goto picker (a {Favorites, Search} menu, not the finder directly).
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
	if m.gotoMenu.isActive() {
		t.Fatal("g alone must not open the goto picker")
	}

	model, _ = m.Update(o)
	m = model.(AppModel)
	if m.pendingG {
		t.Error("`go` should clear the pending state")
	}
	if !m.gotoMenu.isActive() {
		t.Fatal("`go` should open the goto picker")
	}
	if m.search.isActive() {
		t.Error("`go` opens the picker menu, not the finder directly")
	}
	if m.gotoStep != gotoStepRoot {
		t.Errorf("goto picker should start at the root step, got %v", m.gotoStep)
	}
}

// TestGotoMenuFlow drives the Goto picker's branches: Search opens the $HOME
// dirs-only finder; Favorites drills into the list; a number jumps the active tab
// to that dir; f unfavorites in place (the Places-sidebar replacement).
func TestGotoMenuFlow(t *testing.T) {
	// Search branch → the dirs-only finder rooted at $HOME, no new-tab intent.
	m := minModel()
	m.width, m.height = 80, 24
	m.openGotoMenu()
	m.advanceGotoFlow("/")
	if !m.search.isActive() {
		t.Fatal("Goto → Search should open the finder")
	}
	if !m.search.dirsOnly || m.search.byContent || m.search.newTab {
		t.Errorf("Goto Search: want dirs-only name-mode reveal, dirsOnly=%v byContent=%v newTab=%v",
			m.search.dirsOnly, m.search.byContent, m.search.newTab)
	}
	if home, _ := os.UserHomeDir(); home != "" && m.search.root != home {
		t.Errorf("Goto Search root = %q, want $HOME %q", m.search.root, home)
	}

	// Favorites branch → drill, then a number jumps the active tab to that dir.
	dir := t.TempDir()
	m2 := minModel()
	m2.width, m2.height = 80, 24
	m2.places.pinned = []place{{label: "x", path: dir, icon: iconPin}}
	m2.openGotoMenu()
	m2.advanceGotoFlow("f")
	if m2.gotoStep != gotoStepPinned {
		t.Fatal("Goto → Favorites should drill into the favorites step")
	}
	m2.advanceGotoFlow("1")
	if m2.cur().dir != dir {
		t.Errorf("Goto Favorites jump: active tab dir = %q, want %q", m2.cur().dir, dir)
	}

	// f unfavorites the highlighted dir in the picker (was panel [1]'s P).
	m3 := minModel()
	m3.width, m3.height = 80, 24
	m3.places.pinned = []place{{label: "x", path: dir, icon: iconPin}}
	m3.openGotoMenu()
	m3.advanceGotoFlow("f")
	m3.gotoMenu.cursor = 0
	m3.unpinAtGotoCursor()
	if len(m3.places.pinned) != 0 {
		t.Errorf("unpinAtGotoCursor should remove the pin, %d left", len(m3.places.pinned))
	}
}

// TestSearchChooserFlow: `/` opens the {filename, content} chooser (not the
// finder directly); filename opens the by-name finder, content the by-content
// finder. The old top-level `f`=Find binding is gone (freed for Favorite).
func TestSearchChooserFlow(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "a.go"))

	newM := func() AppModel {
		m := minModel()
		m.width, m.height = 100, 30
		m.tabs = []listModel{newList(dir)}
		m.tab = 0
		m.searchMenu = newSearchMenu()
		return m
	}
	slash := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")}
	press := func(m AppModel, r string) AppModel {
		model, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(r)})
		return model.(AppModel)
	}

	// `/` opens the chooser (not the finder); two items keyed f / c.
	m := newM()
	model, _ := m.Update(slash)
	m = model.(AppModel)
	if !m.searchMenu.isActive() {
		t.Fatal("/ should open the Search chooser")
	}
	if m.search.isActive() {
		t.Error("/ opens the chooser, not the finder directly")
	}
	if len(m.searchMenu.items) != 2 || m.searchMenu.items[0].key != "f" || m.searchMenu.items[1].key != "c" {
		t.Fatalf("chooser should offer filename(f)/content(c), got %+v", m.searchMenu.items)
	}

	// filename → by-name finder.
	m.searchMenu.anim.state = popupOpen // interactive so the commit lands
	m = press(m, "f")
	if !m.search.isActive() || m.search.byContent {
		t.Errorf("filename should open the by-name finder, active=%v byContent=%v", m.search.isActive(), m.search.byContent)
	}

	// content → by-content finder.
	m2 := newM()
	model, _ = m2.Update(slash)
	m2 = model.(AppModel)
	m2.searchMenu.anim.state = popupOpen
	m2 = press(m2, "c")
	if !m2.search.isActive() || !m2.search.byContent {
		t.Errorf("content should open the by-content finder, active=%v byContent=%v", m2.search.isActive(), m2.search.byContent)
	}

	// the old top-level f=Find binding is gone.
	m3 := press(newM(), "f")
	if m3.search.isActive() || m3.searchMenu.isActive() {
		t.Error("top-level f should no longer open Find (freed for Favorite)")
	}
}

// TestSearchChooserBecomesInteractive is the regression for the `/` hang: the
// chooser's open animation must be driven by AnimTickMsg (its tick has to be in
// the dispatch batch), else it never becomes interactive and swallows every key.
// Unlike TestSearchChooserFlow this does NOT force anim.state — it drives real
// ticks, the only way to catch missing tick wiring.
func TestSearchChooserBecomesInteractive(t *testing.T) {
	m := minModel()
	m.width, m.height = 100, 30
	m.searchMenu = newSearchMenu()

	model, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")})
	m = model.(AppModel)
	if !m.searchMenu.isActive() {
		t.Fatal("/ should open the Search chooser")
	}
	tick := AnimTickMsg{Target: "searchmenu"}
	for i := 0; i < 20 && !m.searchMenu.isInteractive(); i++ {
		model, _ = m.Update(tick)
		m = model.(AppModel)
	}
	if !m.searchMenu.isInteractive() {
		t.Fatal("searchMenu never became interactive — AnimTickMsg not dispatched to it (the `/` hang)")
	}
}

// TestShellConfirmsFirst: `s` opens a confirm (naming the target dir) rather than
// launching the shell directly; the PTY only starts once the confirm is accepted.
func TestShellConfirmsFirst(t *testing.T) {
	dir := t.TempDir()
	m := minModel()
	m.width, m.height = 80, 24
	m.tabs = []listModel{newList(dir)}
	m.tab = 0

	m.handleListKey("s")
	if !m.confirm.isActive() {
		t.Fatal("s should open a confirm, not the shell directly")
	}
	if m.confirmAction != confirmShell {
		t.Errorf("confirmAction = %v, want confirmShell", m.confirmAction)
	}
	if m.pty.isRendered() {
		t.Error("the shell must not start until the confirm is accepted")
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
