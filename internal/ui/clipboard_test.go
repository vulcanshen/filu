package ui

import (
	"os"
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// runQuiet executes a tea.Cmd with stderr discarded, so the OSC 52 clipboard
// escape a copy writes doesn't leak into test output or the real clipboard.
func runQuiet(cmd tea.Cmd) tea.Msg {
	old := os.Stderr
	if devnull, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0); err == nil {
		os.Stderr = devnull
		defer func() { os.Stderr = old; devnull.Close() }()
	}
	return cmd()
}

func TestCopyToClipboardCmd(t *testing.T) {
	if _, ok := runQuiet(copyToClipboardCmd("", "x")).(clipboardFailedMsg); !ok {
		t.Error("empty text should report failure")
	}
	msg := runQuiet(copyToClipboardCmd("hello", "note"))
	cp, ok := msg.(clipboardCopiedMsg)
	if !ok || cp.note != "note" {
		t.Errorf("expected clipboardCopiedMsg{note}, got %#v", msg)
	}
}

func TestYankPanel2ReturnsCopy(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "f.txt"))
	m := AppModel{focus: panelList, taskCh: make(chan landMsg, 1), watched: map[string]bool{}}
	m.tabs = []listModel{newList(dir)}
	cursorOn(&m, "f.txt")

	cmd := m.handleListKey("y")
	if cmd == nil {
		t.Fatal("y should return a copy cmd")
	}
	if _, ok := runQuiet(cmd).(clipboardCopiedMsg); !ok {
		t.Error("the copy cmd should report success")
	}
}

func TestYankMarksReturnsCopy(t *testing.T) {
	m := AppModel{focus: panelMarks, watched: map[string]bool{}}
	m.marks.items = []string{"/a/b.txt"}
	m.marks.cursor = 0

	cmd := m.handleMarksKey("y")
	if cmd == nil {
		t.Fatal("y in Marks should return a copy cmd")
	}
	if _, ok := runQuiet(cmd).(clipboardCopiedMsg); !ok {
		t.Error("the copy cmd should report success")
	}
}

func TestToastShowDismiss(t *testing.T) {
	m := newToast()
	m.show("hi")
	if !m.isActive() {
		t.Error("toast should be active after show")
	}
	if m.dismiss(toastDismissMsg{id: m.id + 1}) != nil {
		t.Error("a stale dismiss (superseded id) must be ignored")
	}
	if m.dismiss(toastDismissMsg{id: m.id}) == nil {
		t.Error("a matching dismiss should close the toast")
	}
}
