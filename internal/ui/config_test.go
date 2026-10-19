package ui

import (
	"os"
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// FILU__CONFIG names a config directory, config.yaml inside it (tdp E2: an
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

// The settings directory is ~/.config/filu on every platform (tdp E3); the
// first start moves what is in the Application Support one older versions used
// on macOS there, one item at a time: what the new place has already stays in
// the old one, the old one goes once empty, and only a move that fails leaves
// filu using the old one.
func TestE3ConfigDir(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("FILU__CONFIG", "")
	oldC := configPathOverride
	configPathOverride = ""
	defer func() { configPathOverride, stuckInOld = oldC, false }()
	newDir := filepath.Join(home, ".config", "filu")
	if got, _ := filuConfigDir(); got != newDir {
		t.Fatalf("the settings directory is %s", got)
	}
	old := legacyConfigDir(home)
	if old == "" {
		return // only macOS had another place
	}
	write := func(dir, name, body string) {
		t.Helper()
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	read := func(dir, name string) string {
		b, _ := os.ReadFile(filepath.Join(dir, name))
		return string(b)
	}
	write(old, "config.yaml", "finder_cap: 7\n")
	write(old, "state.yaml", "tabs: []\n")
	m := moveOldSettings()
	if m.err != nil || !m.moved || m.from != old || m.to != newDir || len(m.left) != 0 {
		t.Fatalf("moved %+v", m)
	}
	if read(newDir, "config.yaml") != "finder_cap: 7\n" || read(newDir, "state.yaml") != "tabs: []\n" || isDir(old) {
		t.Fatal("not everything is in the new place, or the old one is still there")
	}
	if m := moveOldSettings(); m.from != "" {
		t.Fatal("a second start moved something again")
	}

	// The new place has config.yaml already: it stays as it is, and so does
	// the old one; state.yaml moves.
	write(old, "config.yaml", "finder_cap: 8\n")
	write(old, "state.yaml", "tabs: []\n")
	if err := os.Remove(filepath.Join(newDir, "state.yaml")); err != nil {
		t.Fatal(err)
	}
	m = moveOldSettings()
	if m.err != nil || !m.moved || len(m.left) != 1 || m.left[0] != "config.yaml" {
		t.Fatalf("moved %+v", m)
	}
	if read(newDir, "config.yaml") != "finder_cap: 7\n" || read(old, "config.yaml") != "finder_cap: 8\n" ||
		read(newDir, "state.yaml") != "tabs: []\n" || read(old, "state.yaml") != "" {
		t.Fatal("an item in both places was moved or deleted, or one in the old place only was not moved")
	}

	// A move that fails: filu keeps the old one.
	if err := os.RemoveAll(filepath.Join(home, ".config")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, ".config"), nil, 0o644); err != nil { // a file: nothing goes under it
		t.Fatal(err)
	}
	if m := moveOldSettings(); m.err == nil || read(old, "config.yaml") != "finder_cap: 8\n" {
		t.Fatalf("moving under a file: %+v", m)
	}
	if got, _ := filuConfigDir(); got != old {
		t.Fatalf("after a failed move the settings directory is %s", got)
	}
}

// Started on settings in both places, filu says so with both paths
// (tdp E3).
func TestE3FoundInTwoPlaces(t *testing.T) {
	if legacyConfigDir("/") == "" {
		t.Skip("only macOS had another place")
	}
	hs := newHarness(t, 120, 40, nil)
	t.Setenv("FILU__CONFIG", "")
	oldC := configPathOverride
	configPathOverride = ""
	defer func() { configPathOverride = oldC }()
	old, cur := legacyConfigDir(hs.home), filepath.Join(hs.home, ".config", "filu")
	for _, dir := range []string{old, cur} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "config.yaml"), []byte("finder_cap: 7\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	hs.a = NewApp(hs.root, "")
	hs.send(tea.WindowSizeMsg{Width: 120, Height: 40})
	hs.run(hs.a.start())
	hs.has("FOUND IN TWO PLACES", "Support/filu/config.yaml", ".config/filu/config.yaml")
}

// config.yaml takes the family's shared settings (tdp E3).
func TestSharedSettings(t *testing.T) {
	dir := t.TempDir()
	old := configPathOverride
	configPathOverride = filepath.Join(dir, "config.yaml")
	defer func() { configPathOverride = old }()
	defer resetGlobals()
	if err := os.WriteFile(configPathOverride, []byte("input_tab_width: 2\nhide_secrets_in_preview: true\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := loadConfig(); err != nil {
		t.Fatal(err)
	}
	if inputTabWidth != 2 || !hideSecretsInPreview {
		t.Fatalf("input_tab_width %d, hide_secrets_in_preview %v", inputTabWidth, hideSecretsInPreview)
	}
	if got := sanitizeLine("\tx"); got != "  x" {
		t.Fatalf("a tab in the preview is %q", got)
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
