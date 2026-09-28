package ui

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// tdp F5: a failed OS open comes back as a message naming the file and the
// reason, instead of vanishing.
func TestF5OpenFailureIsReported(t *testing.T) {
	old := openFile
	openFile = func(string) error { return errors.New("no application knows this file") }
	defer func() { openFile = old }()

	msg, ok := openFileCmd("/tmp/report.pdf")().(opFailedMsg)
	if !ok || msg.text != "Cannot open report.pdf: no application knows this file" {
		t.Errorf("a failed open should report itself, got %#v", msg)
	}
}

// tdp F5: an open-with app that cannot start is reported too.
func TestF5OpenWithFailureIsReported(t *testing.T) {
	msg, ok := openWithCmd("filu-no-such-editor --flag", "/tmp/x.txt")().(opFailedMsg)
	if !ok || !strings.HasPrefix(msg.text, "Cannot run filu-no-such-editor: ") {
		t.Errorf("an app that cannot start should report itself, got %#v", msg)
	}
}

// tdp F5: the report reaches the screen as a toast, which Esc can close.
func TestF5FailureMessageShowsToast(t *testing.T) {
	m := minModel()
	m.toast = newToast()
	model, _ := m.Update(opFailedMsg{text: "Cannot open x: boom"})
	got := model.(AppModel)
	if !got.toast.owns() || got.toast.message != "Cannot open x: boom" {
		t.Errorf("a failure should show as a toast: owns %v, %q", got.toast.owns(), got.toast.message)
	}
}

// opFailedText drops the path the OS error repeats and keeps the reason.
func TestF5FailedTextKeepsTheReason(t *testing.T) {
	_, err := os.Stat(filepath.Join(t.TempDir(), "missing"))
	if got := opFailedText("open missing", err); got != "Cannot open missing: no such file or directory" {
		t.Errorf("opFailedText = %q", got)
	}
}

// tdp F5: the synchronous file operations — trash, rename, add — show their
// failure as a toast right away.
func TestF5FileOperationFailuresShowToast(t *testing.T) {
	for _, tc := range []struct {
		name string
		run  func(m *AppModel) tea.Cmd
		want string
	}{
		{"trash", func(m *AppModel) tea.Cmd {
			t.Setenv("HOME", t.TempDir())
			m.pendingDelete = filepath.Join(m.tabs[0].dir, "gone.txt") // not there → the move fails
			m.confirmAction = confirmDelete
			m.confirm.open("Move gone.txt to the trash?", "trash")
			m.confirm.anim.state = popupOpen
			model, cmd := m.Update(runes("y"))
			*m = model.(AppModel)
			return cmd
		}, "Cannot move gone.txt to the trash: "},
		{"rename", func(m *AppModel) tea.Cmd {
			m.inputPopup.open(inputRename, "Rename", "b.txt", fileItem{name: "missing.txt"})
			return m.performInput()
		}, "Cannot rename missing.txt: "},
		{"add", func(m *AppModel) tea.Cmd {
			m.inputPopup.open(inputAdd, "Add", "a.txt", fileItem{}) // a.txt already exists
			return m.performInput()
		}, "Cannot create a.txt: "},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := f4Model(t)
			if cmd := tc.run(&m); cmd == nil {
				t.Fatal("a failed operation should return the toast cmd")
			}
			if !m.toast.owns() || !strings.HasPrefix(m.toast.message, tc.want) {
				t.Errorf("toast owns %v, message %q, want prefix %q", m.toast.owns(), m.toast.message, tc.want)
			}
		})
	}
}
