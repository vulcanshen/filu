package ui

import (
	"os"
	"path/filepath"
	"runtime"
)

// Nerd Font icons (rune values so no PUA glyph sits in source).
var (
	iconCWD = string(rune(0xf14de)) // launch dir (cd-on-quit picker)
	iconPin = string(rune(0xf450))  // pinned dir
)

type place struct {
	label string
	path  string
	icon  string
}

// placesModel holds the user's pinned directories. Pins are created from the
// list ([P]in) and reached through the Goto / new-tab pickers' Pinned drill-down
// — there is no longer a Places sidebar panel. Persisted in state.yaml.
type placesModel struct {
	pinned []place
}

// unpin removes path from the pinned list if present.
func (m *placesModel) unpin(path string) {
	for i, p := range m.pinned {
		if p.path == path {
			m.pinned = append(m.pinned[:i], m.pinned[i+1:]...)
			return
		}
	}
}

// togglePin adds or removes a pinned directory.
func (m *placesModel) togglePin(path string) {
	for i, p := range m.pinned {
		if p.path == path {
			m.pinned = append(m.pinned[:i], m.pinned[i+1:]...)
			return
		}
	}
	m.pinned = append(m.pinned, place{label: filepath.Base(path), path: path, icon: iconPin})
}

// pinnedSet is the set of pinned directory paths, for the list's pin mark glyph
// (symmetry with carry.inBucket). nil when nothing is pinned.
func (m placesModel) pinnedSet() map[string]bool {
	if len(m.pinned) == 0 {
		return nil
	}
	s := make(map[string]bool, len(m.pinned))
	for _, p := range m.pinned {
		s[p.path] = true
	}
	return s
}

// trashDir is the system trash location. TODO: move behind the platform
// interface (unix-first: macOS ~/.Trash, Linux XDG).
func trashDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	if runtime.GOOS == "darwin" {
		return filepath.Join(home, ".Trash")
	}
	return filepath.Join(home, ".local", "share", "Trash", "files")
}
