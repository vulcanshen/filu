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

func TestEnterOpensFileNavigatesDir(t *testing.T) {
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

	// Enter on the file → OS open, dir unchanged.
	cursorOn(&m, "doc.txt")
	cmd := m.handleListKey("enter")
	if cmd == nil {
		t.Fatal("enter on a file should return an open cmd")
	}
	cmd() // execute the tea.Cmd
	if want := filepath.Join(dir, "doc.txt"); opened != want {
		t.Errorf("opened %q, want %q", opened, want)
	}
	if m.cur().dir != dir {
		t.Errorf("opening a file must not change the directory, got %q", m.cur().dir)
	}

	// Enter on the directory → descend, no open.
	opened = ""
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
