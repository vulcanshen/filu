package ui

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	origCap, origIgnore := finderCap, finderIgnoreDirs
	defer func() {
		finderCap, finderIgnoreDirs, configPathOverride = origCap, origIgnore, ""
	}()

	path := filepath.Join(t.TempDir(), "config.yaml")
	configPathOverride = path

	// 1. missing file → keeps the defaults and drops an editable template
	finderCap, finderIgnoreDirs = defaultFinderCap, defaultIgnoreDirs
	loadConfig()
	if finderCap != defaultFinderCap {
		t.Errorf("missing config should keep the default cap, got %d", finderCap)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("first run should write a template config: %v", err)
	}

	// 1b. that written template is valid and loads back to the defaults
	finderCap, finderIgnoreDirs = 999, nil
	loadConfig()
	if finderCap != defaultFinderCap {
		t.Errorf("the template should load back to the default cap, got %d", finderCap)
	}
	if len(finderIgnoreDirs) != len(defaultIgnoreDirs) {
		t.Errorf("the template should carry the default ignore list, got %v", finderIgnoreDirs)
	}

	// 2. explicit values override
	if err := os.WriteFile(path, []byte("finder_cap: 30000\nignore_dirs: [foo, bar]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	finderCap, finderIgnoreDirs = defaultFinderCap, defaultIgnoreDirs
	loadConfig()
	if finderCap != 30000 {
		t.Errorf("finder_cap should override, got %d", finderCap)
	}
	if len(finderIgnoreDirs) != 2 || finderIgnoreDirs[0] != "foo" {
		t.Errorf("ignore_dirs should override, got %v", finderIgnoreDirs)
	}

	// 3. an invalid cap is ignored, and an absent ignore_dirs keeps the default
	if err := os.WriteFile(path, []byte("finder_cap: 0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	finderCap, finderIgnoreDirs = defaultFinderCap, defaultIgnoreDirs
	loadConfig()
	if finderCap != defaultFinderCap {
		t.Errorf("finder_cap<=0 should be ignored, got %d", finderCap)
	}
	if len(finderIgnoreDirs) != len(defaultIgnoreDirs) {
		t.Errorf("absent ignore_dirs should keep the default, got %v", finderIgnoreDirs)
	}

	// 4. an explicit empty list means exclude nothing
	if err := os.WriteFile(path, []byte("ignore_dirs: []\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	finderIgnoreDirs = defaultIgnoreDirs
	loadConfig()
	if len(finderIgnoreDirs) != 0 {
		t.Errorf("an explicit empty ignore_dirs should exclude nothing, got %v", finderIgnoreDirs)
	}
}
