package ui

import (
	"os"
	"path/filepath"
	"testing"
)

// TestShortenPinPath checks the compact-path rendering for pinned dirs: home
// folds to ~, every segment but the last is cut to its first rune, and the last
// segment (the folder name) is kept whole.
func TestShortenPinPath(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("no home dir")
	}
	underHome := filepath.Join(home, "Documents", "sideproj", "filu")

	cases := map[string]string{
		underHome:        "~/D/s/filu",
		home:             "~",
		"/usr/local/bin": "/u/l/bin",
		"/":              "/",
		"/single":        "/single", // one segment past root stays whole
	}
	for in, want := range cases {
		if got := shortenPinPath(in); got != want {
			t.Errorf("shortenPinPath(%q) = %q, want %q", in, got, want)
		}
	}
}
