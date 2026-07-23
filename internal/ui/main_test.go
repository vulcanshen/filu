package ui

import (
	"os"
	"path/filepath"
	"testing"
)

// TestMain redirects all state persistence to a throwaway temp file so tests
// that trigger saveState never touch the user's real ~/.config/filu/state.yaml.
func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "filu-test")
	if err == nil {
		statePathOverride = filepath.Join(dir, "state.yaml")
	}
	code := m.Run()
	if dir != "" {
		os.RemoveAll(dir)
	}
	os.Exit(code)
}
