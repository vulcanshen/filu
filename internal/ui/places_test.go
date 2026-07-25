package ui

import (
	"os"
	"path/filepath"
	"testing"
)

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
