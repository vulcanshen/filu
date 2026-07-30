package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

// TestFavoriteKeyTogglesDir: `f` favorites the cursor dir (into places), toggles
// it back off, and is a no-op on a non-dir.
func TestFavoriteKeyTogglesDir(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	mustWrite(t, filepath.Join(dir, "file.txt"))

	m := minModel()
	m.width, m.height = 80, 24
	m.tabs = []listModel{newList(dir)}
	m.tab = 0
	l := m.cur()
	at := func(name string) {
		for i, it := range l.items {
			if it.name == name {
				l.cursor = i
				return
			}
		}
		t.Fatalf("%q not listed", name)
	}

	at("sub")
	m.handleListKey("f") // favorite the dir
	if s := m.places.pinnedSet(); !s[filepath.Join(dir, "sub")] {
		t.Fatalf("f should favorite the cursor dir; favorites=%v", m.places.pinned)
	}
	m.handleListKey("f") // toggle off
	if len(m.places.pinned) != 0 {
		t.Errorf("second f should unfavorite; %d left", len(m.places.pinned))
	}

	at("file.txt")
	m.handleListKey("f") // non-dir: no-op
	if len(m.places.pinned) != 0 {
		t.Errorf("f on a non-dir should not favorite; %d favorites", len(m.places.pinned))
	}
}

// TestFavoriteDirKeyFavoritesCurrentTab: [F] favorites the tab's *current*
// directory (l.dir) regardless of which item is highlighted — distinct from [f],
// which favorites the highlighted subdirectory.
func TestFavoriteDirKeyFavoritesCurrentTab(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "file.txt"))

	m := minModel()
	m.width, m.height = 80, 24
	m.tabs = []listModel{newList(dir)}
	m.tab = 0
	for i, it := range m.cur().items { // highlight a plain file
		if it.name == "file.txt" {
			m.cur().cursor = i
		}
	}

	m.handleListKey("F") // favorites the directory, not the highlighted file
	if s := m.places.pinnedSet(); !s[dir] {
		t.Fatalf("F should favorite the tab's current dir %q; favorites=%v", dir, m.places.pinned)
	}
	m.handleListKey("F") // toggle off
	if len(m.places.pinned) != 0 {
		t.Errorf("second F should unfavorite; %d left", len(m.places.pinned))
	}
}

// TestFitPath checks the compact-path rendering for pinned dirs, now shared with
// the header breadcrumb: it fits when it fits, then abbreviates front segments to
// their initial, then collapses the middle to … keeping root + current.
func TestFitPath(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("no home dir")
	}
	underHome := filepath.Join(home, "Documents", "sideproj", "filu")

	cases := []struct {
		path string
		w    int
		want string
	}{
		{underHome, 100, "~/Documents/sideproj/filu"}, // fits → full
		{underHome, 12, "~/D/s/filu"},                 // front initials
		{underHome, 8, "~/…/filu"},                    // middle collapse: root + current
		{"/usr/local/bin", 8, "/u/l/bin"},
		{"/", 5, "/"},
		{"/single", 20, "/single"}, // one segment past root stays whole
	}
	for _, c := range cases {
		if got := fitPath(c.path, c.w); got != c.want {
			t.Errorf("fitPath(%q, %d) = %q, want %q", c.path, c.w, got, c.want)
		}
	}
}

// TestFavoritesTabManage: the Favorites tab moves a cursor over the favorited
// dirs; D asks to confirm, and accepting unfavorites the highlighted one, keeping
// the cursor in range.
func TestFavoritesTabManage(t *testing.T) {
	m := minModel()
	m.width, m.height = 80, 24
	m.places.pinned = []place{
		{label: "a", path: "/home/me/a", icon: iconPin},
		{label: "b", path: "/home/me/b", icon: iconPin},
		{label: "c", path: "/home/me/c", icon: iconPin},
	}
	m.marksTab = 2 // Favorites tab

	m.handleMarksKey("j") // cursor 0 → 1 (b)
	if m.places.cursor != 1 {
		t.Fatalf("j should move to cursor 1, got %d", m.places.cursor)
	}
	// D arms a confirm — nothing is removed yet
	m.handleMarksKey("D")
	if m.confirmAction != confirmUnfavorite {
		t.Fatalf("D should arm the unfavorite confirm, got %v", m.confirmAction)
	}
	if len(m.places.pinned) != 3 {
		t.Errorf("D must not unfavorite before the confirm is accepted, %d left", len(m.places.pinned))
	}
	// accepting removes the highlighted "b"
	m.confirm.anim.state = popupOpen
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(AppModel)
	if len(m.places.pinned) != 2 || m.places.pinned[1].path != "/home/me/c" {
		t.Fatalf("accepting should unfavorite b; left=%v", m.places.pinned)
	}
	if m.places.cursor < 0 || m.places.cursor >= len(m.places.pinned) {
		t.Errorf("cursor should stay in range, got %d (len %d)", m.places.cursor, len(m.places.pinned))
	}
}

// TestPanelMarksTabCycle: l cycles panel [3] forward Marks → Tasks → Favorites →
// Marks, h cycles back.
func TestPanelMarksTabCycle(t *testing.T) {
	m := minModel()
	m.marksTab = 0
	for _, want := range []int{1, 2, 0} { // l advances
		m.handleMarksKey("l")
		if m.marksTab != want {
			t.Fatalf("l should advance to tab %d, got %d", want, m.marksTab)
		}
	}
	for _, want := range []int{2, 1, 0} { // h retreats
		m.handleMarksKey("h")
		if m.marksTab != want {
			t.Fatalf("h should retreat to tab %d, got %d", want, m.marksTab)
		}
	}
}

// TestFavoritesViewRendersPaths: the Favorites tab lists a favorite's full path,
// with an empty-state note when there are none.
func TestFavoritesViewRendersPaths(t *testing.T) {
	var p placesModel
	if !strings.Contains(p.view(60, 5, true), "no favorites") {
		t.Error("empty favorites should show the empty-state note")
	}
	p.pinned = []place{{label: "proj", path: "/home/me/work/proj", icon: iconPin}}
	if out := ansi.Strip(p.view(60, 5, true)); !strings.Contains(out, "proj") {
		t.Errorf("favorites view should show the path, got %q", out)
	}
}
