package ui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func minModel() AppModel {
	m := AppModel{focus: panelList, confirm: newConfirmPopup(), pty: newPtyPopup(), taskCh: make(chan landMsg, 1), watched: map[string]bool{}}
	for i := range m.tabs {
		m.tabs[i] = listModel{dir: "/tmp"}
	}
	return m
}

func isQuitCmd(cmd tea.Cmd) bool {
	if cmd == nil {
		return false
	}
	_, ok := cmd().(tea.QuitMsg)
	return ok
}

func TestQuitConfirmsWhenTaskRunning(t *testing.T) {
	m := minModel()
	m.tasks = []landTask{{id: 1, status: taskRunning}}

	model, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	got := model.(AppModel)
	if !got.confirm.isActive() {
		t.Error("q with a running task should open the quit confirm, not quit")
	}
	if got.confirmAction != confirmQuit {
		t.Errorf("confirmAction = %v, want confirmQuit", got.confirmAction)
	}
}

func TestQuitImmediateWhenIdle(t *testing.T) {
	m := minModel()
	if _, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")}); !isQuitCmd(cmd) {
		t.Error("q with no running task should quit immediately")
	}
}

func TestCtrlCForceQuitsDespiteRunningTask(t *testing.T) {
	m := minModel()
	m.tasks = []landTask{{id: 1, status: taskRunning}}
	if _, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC}); !isQuitCmd(cmd) {
		t.Error("ctrl+c should force quit even with a running task")
	}
}
