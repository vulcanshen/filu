package ui

import (
	"os"
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func minModel() AppModel {
	m := AppModel{
		focus: panelList, confirm: newConfirmPopup(), quitMenu: newQuitMenu(),
		pty: newPtyPopup(), taskCh: make(chan landMsg, 1), watched: map[string]bool{},
	}
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

func TestQuitOpensCdMenu(t *testing.T) {
	m := minModel()
	m.launchDir = "/launch"

	model, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	got := model.(AppModel)
	if !got.quitMenu.isActive() {
		t.Fatal("q should open the cd-on-quit menu")
	}
	if len(got.quitMenu.items) != 4 {
		t.Errorf("menu should offer 4 targets (launch + 3 tabs), got %d", len(got.quitMenu.items))
	}
}

func TestQuitMenuSelectWritesDirAndQuits(t *testing.T) {
	dirFile := filepath.Join(t.TempDir(), "cwd")
	t.Setenv(envLastDirFile, dirFile)

	m := minModel()
	m.launchDir = "/launch"
	m.tabs[0].dir = "/tab-one"
	m.openQuitMenu()
	m.quitMenu.anim.state = popupOpen // make it interactive so the commit lands

	// option 2 = tab 1 (tabs[0])
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("2")})
	if !isQuitCmd(cmd) {
		t.Error("selecting a target should quit")
	}
	if data, _ := os.ReadFile(dirFile); string(data) != "/tab-one" {
		t.Errorf("cd-on-quit wrote %q, want /tab-one", string(data))
	}
}

func TestCtrlCForceQuits(t *testing.T) {
	m := minModel()
	if _, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC}); !isQuitCmd(cmd) {
		t.Error("ctrl+c should quit immediately")
	}
}

func TestWriteLastDirNoopWithoutEnv(t *testing.T) {
	t.Setenv(envLastDirFile, "") // feature off
	writeLastDir("/somewhere")   // must not panic or write anywhere
}
