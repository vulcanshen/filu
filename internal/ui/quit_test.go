package ui

import (
	"os"
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func minModel() AppModel {
	m := AppModel{
		focus: panelList, confirm: newConfirmPopup(), quitMenu: newQuitMenu(), gotoMenu: newGotoMenu(),
		pty: newPtyPopup(), taskCh: make(chan landMsg, 1), watched: map[string]bool{},
	}
	m.tabs = []listModel{{dir: "/tmp"}, {dir: "/tmp"}, {dir: "/tmp"}}
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
	// /launch + three /tmp tabs → duplicates dropped → 2 distinct targets.
	if len(got.quitMenu.items) != 2 {
		t.Errorf("menu should offer 2 distinct targets, got %d", len(got.quitMenu.items))
	}
}

func TestQuitTargetsDedup(t *testing.T) {
	m := AppModel{launchDir: "/a"}
	m.tabs = []listModel{{dir: "/a"}, {dir: "/b"}, {dir: "/b"}} // /a == launch, /b twice
	tg := m.quitTargets()
	if len(tg) != 2 {
		t.Fatalf("want 2 distinct targets, got %d: %+v", len(tg), tg)
	}
	if tg[0].dir != "/a" || tg[1].dir != "/b" {
		t.Errorf("targets = %+v, want /a then /b", tg)
	}
	if tg[0].hint != iconCWD+" " { // launch glyph + trailing space
		t.Errorf("launch hint = %q, want %q", tg[0].hint, iconCWD+" ")
	}
	if tg[1].hint != tabMark(1)+" " { // /b first appeared as tab Ⅱ (index 1)
		t.Errorf("/b hint = %q, want %q", tg[1].hint, tabMark(1)+" ")
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

func TestQuitMenuWarnsWhenTaskRunning(t *testing.T) {
	m := minModel()
	m.tasks = []landTask{{id: 1, status: taskRunning}}
	m.openQuitMenu()
	warned := false
	for _, it := range m.quitMenu.items {
		warned = warned || it.warn
	}
	if !warned {
		t.Error("quit menu should carry a warning while a task is running")
	}

	idle := minModel()
	idle.openQuitMenu()
	for _, it := range idle.quitMenu.items {
		if it.warn {
			t.Error("quit menu should not warn when idle")
		}
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
