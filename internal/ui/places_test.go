package ui

import (
	"os"
	"path/filepath"
	"testing"
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
