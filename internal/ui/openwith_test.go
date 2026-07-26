package ui

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfigOpenWith(t *testing.T) {
	orig := openWithApps
	defer func() { openWithApps, configPathOverride = orig, "" }()

	path := filepath.Join(t.TempDir(), "config.yaml")
	configPathOverride = path
	cfg := "open_with:\n  - name: VSCode\n    cmd: code\n  - name: IDEA\n    cmd: idea -p\n"
	if err := os.WriteFile(path, []byte(cfg), 0o644); err != nil {
		t.Fatal(err)
	}

	openWithApps = nil
	loadConfig()
	if len(openWithApps) != 2 {
		t.Fatalf("open_with should parse 2 apps, got %d", len(openWithApps))
	}
	if openWithApps[0].Name != "VSCode" || openWithApps[0].Cmd != "code" {
		t.Errorf("app[0] = %+v, want VSCode/code", openWithApps[0])
	}
	if openWithApps[1].Cmd != "idea -p" { // args survive
		t.Errorf("app[1].Cmd = %q, want 'idea -p'", openWithApps[1].Cmd)
	}
}

func TestOpenDefault(t *testing.T) {
	var opened string
	oldOpen := openFile
	openFile = func(p string) error { opened = p; return nil }
	defer func() { openFile = oldOpen }()

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "doc.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	m := AppModel{focus: panelList}
	m.tabs = []listModel{newList(dir)}
	cursorOn(&m, "doc.txt")

	// o = open with the OS default app, directly — no picker.
	cmd := m.openDefault()
	if cmd == nil {
		t.Fatal("openDefault should return a cmd")
	}
	cmd()
	if want := filepath.Join(dir, "doc.txt"); opened != want {
		t.Errorf("openDefault opened %q, want %q", opened, want)
	}
	if m.openWithMenu.isActive() {
		t.Error("o (default open) must not open the picker")
	}
}

func TestOpenOpenWithMenu(t *testing.T) {
	orig := openWithApps
	defer func() { openWithApps = orig }()
	openWithApps = []openWithApp{{Name: "VSCode", Cmd: "code"}, {Name: "IDEA", Cmd: "idea"}}

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "doc.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	m := AppModel{focus: panelList}
	m.tabs = []listModel{newList(dir)}
	cursorOn(&m, "doc.txt")

	if cmd := m.openOpenWith(); cmd == nil {
		t.Fatal("openOpenWith should return an open cmd")
	}
	if want := filepath.Join(dir, "doc.txt"); m.openWithPath != want {
		t.Errorf("openWithPath = %q, want %q", m.openWithPath, want)
	}
	// Default + 2 configured apps = 3 items, numbered 1..3.
	if n := len(m.openWithMenu.items); n != 3 {
		t.Fatalf("menu items = %d, want 3 (Default + 2 apps)", n)
	}
	if got := m.openWithMenu.items[0]; got.label != "Default" || got.key != "1" {
		t.Errorf("item[0] = %+v, want Default/1", got)
	}
	if got := m.openWithMenu.items[1]; got.label != "VSCode" || got.key != "2" {
		t.Errorf("item[1] = %+v, want VSCode/2", got)
	}
}

func TestRunOpenWith(t *testing.T) {
	orig := openWithApps
	defer func() { openWithApps = orig }()
	openWithApps = []openWithApp{{Name: "VSCode", Cmd: "code"}}

	var opened string
	oldOpen := openFile
	openFile = func(p string) error { opened = p; return nil }
	defer func() { openFile = oldOpen }()

	m := AppModel{openWithPath: "/tmp/x.txt"}

	// idx 1 = Default → openFileCmd → the openFile stub.
	cmd := m.runOpenWith(1)
	if cmd == nil {
		t.Fatal("runOpenWith(1) should return a cmd")
	}
	cmd()
	if opened != "/tmp/x.txt" {
		t.Errorf("Default opened %q, want /tmp/x.txt", opened)
	}

	// idx 2 = the first configured app → a non-nil cmd (not executed here, as it
	// would spawn a real process).
	if m.runOpenWith(2) == nil {
		t.Error("runOpenWith(2) should return a cmd for the configured app")
	}
	// out of range and no target both yield nil.
	if m.runOpenWith(9) != nil {
		t.Error("runOpenWith(9) out of range should be nil")
	}
	m.openWithPath = ""
	if m.runOpenWith(1) != nil {
		t.Error("runOpenWith with no path should be nil")
	}
}
