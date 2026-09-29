package ui

import (
	"os"
	"path/filepath"
	"testing"
)

// FILU__CONFIG names a config directory, config.yaml inside it (tdp D6: an
// app's variables are APP__NAME, CONFIG a directory; used by the demo tapes to
// isolate state); the old FILU_CONFIG is not read any more, and the test-only
// override still wins.
func TestConfigPathEnvOverride(t *testing.T) {
	old := configPathOverride
	configPathOverride = ""
	defer func() { configPathOverride = old }()

	t.Setenv("XDG_CONFIG_HOME", "/tmp/xdg")
	t.Setenv("FILU_CONFIG", "/tmp/old/config.yaml") // renamed without a fallback
	if got, _ := configFilePath(); got != "/tmp/xdg/filu/config.yaml" {
		t.Errorf("the old FILU_CONFIG should be ignored, got %q", got)
	}
	t.Setenv("FILU__CONFIG", "/tmp/filu-demo")
	if got, ok := configFilePath(); !ok || got != "/tmp/filu-demo/config.yaml" {
		t.Errorf("FILU__CONFIG not honoured as a directory: got %q ok=%v", got, ok)
	}

	configPathOverride = "/tmp/override/config.yaml" // test override beats the env
	if got, _ := configFilePath(); got != "/tmp/override/config.yaml" {
		t.Errorf("configPathOverride should win over FILU__CONFIG, got %q", got)
	}
}

// TestConfigDirHonoursXDG: XDG_CONFIG_HOME redirects both config.yaml and
// state.yaml on every platform (so a macOS user can opt into ~/.config), but the
// FILU__CONFIG / FILU__STATE directories still win over it.
func TestConfigDirHonoursXDG(t *testing.T) {
	oldC, oldS := configPathOverride, statePathOverride
	configPathOverride, statePathOverride = "", ""
	defer func() { configPathOverride, statePathOverride = oldC, oldS }()
	t.Setenv("FILU__CONFIG", "") // clear the directory overrides so XDG is exercised
	t.Setenv("FILU__STATE", "")

	t.Setenv("XDG_CONFIG_HOME", "/tmp/xdg")
	if got, ok := configFilePath(); !ok || got != "/tmp/xdg/filu/config.yaml" {
		t.Errorf("XDG_CONFIG_HOME should put config at /tmp/xdg/filu/config.yaml, got %q ok=%v", got, ok)
	}
	if got, ok := stateFilePath(); !ok || got != "/tmp/xdg/filu/state.yaml" {
		t.Errorf("XDG_CONFIG_HOME should put state at /tmp/xdg/filu/state.yaml, got %q ok=%v", got, ok)
	}

	// the directory overrides still beat XDG
	t.Setenv("FILU__CONFIG", "/tmp/explicit-config")
	t.Setenv("FILU__STATE", "/tmp/explicit-state")
	if got, _ := configFilePath(); got != "/tmp/explicit-config/config.yaml" {
		t.Errorf("FILU__CONFIG should win over XDG, got %q", got)
	}
	if got, _ := stateFilePath(); got != "/tmp/explicit-state/state.yaml" {
		t.Errorf("FILU__STATE should win over XDG, got %q", got)
	}
}

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
