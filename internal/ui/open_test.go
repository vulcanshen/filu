package ui

import (
	"os"
	"path/filepath"
	"testing"
)

// cursorOn moves the active tab's cursor onto the entry named `name`.
func cursorOn(m *AppModel, name string) {
	for i, it := range m.cur().items {
		if it.name == name {
			m.cur().cursor = i
			return
		}
	}
}

func TestEnterDescendsDirNotOpenFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "doc.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(dir, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}

	var opened string
	old := openFile
	openFile = func(p string) error { opened = p; return nil }
	defer func() { openFile = old }()

	m := AppModel{focus: panelList, taskCh: make(chan landMsg, 1)}
	m.tabs = []listModel{newList(dir)}

	// Enter on a file → no-op: it is not "open a file" in filu (that is [o]pen's
	// job). Nothing opens and the directory is unchanged.
	cursorOn(&m, "doc.txt")
	if cmd := m.handleListKey("enter"); cmd != nil {
		t.Error("enter on a file should be a no-op, got a cmd")
	}
	if opened != "" {
		t.Errorf("enter on a file must not open anything, opened %q", opened)
	}
	if m.cur().dir != dir {
		t.Errorf("enter on a file must not change the directory, got %q", m.cur().dir)
	}

	// Enter on the directory → descend, no open.
	cursorOn(&m, "sub")
	if cmd := m.handleListKey("enter"); cmd != nil {
		t.Error("enter on a dir should not return an open cmd")
	}
	if want := filepath.Join(dir, "sub"); m.cur().dir != want {
		t.Errorf("enter on a dir should descend to %q, got %q", want, m.cur().dir)
	}
	if opened != "" {
		t.Errorf("entering a dir must not open anything, opened %q", opened)
	}
}

// TestOpenFileCmd covers the retained OS-open primitive directly. No UI key
// wires it up right now (Enter navigates, edit is gone), but [o]pen / open-with
// will reuse it for the default-app path, so it stays tested.
func TestOpenFileCmd(t *testing.T) {
	var opened string
	old := openFile
	openFile = func(p string) error { opened = p; return nil }
	defer func() { openFile = old }()

	cmd := openFileCmd("/tmp/x.txt")
	if cmd == nil {
		t.Fatal("openFileCmd should return a cmd")
	}
	cmd() // execute the tea.Cmd
	if opened != "/tmp/x.txt" {
		t.Errorf("openFileCmd opened %q, want /tmp/x.txt", opened)
	}
}
