package ui

import (
	"os"
	"strconv"

	tea "github.com/charmbracelet/bubbletea"
)

// envLastDirFile names the file filu writes the chosen directory to on quit, so
// a shell wrapper can cd there — superfile's cd_on_quit. Unset = feature off.
const envLastDirFile = "FILU_LAST_DIR_FILE"

// writeLastDir records dir for the shell wrapper's cd-on-quit, when enabled.
func writeLastDir(dir string) {
	path := os.Getenv(envLastDirFile)
	if path == "" || dir == "" {
		return
	}
	_ = os.WriteFile(path, []byte(dir), 0o644)
}

// quitDirs is the cd-on-quit menu's four targets: the launch dir (panel [1]'s
// CWD) then the three list-tab directories.
func (m AppModel) quitDirs() []string {
	return []string{m.launchDir, m.tabs[0].dir, m.tabs[1].dir, m.tabs[2].dir}
}

// openQuitMenu opens the quit picker: pick where to leave the shell (1–4 or
// j/k + Enter), or Esc to stay.
func (m *AppModel) openQuitMenu() tea.Cmd {
	dirs := m.quitDirs()
	hints := []string{"panel 1 (launch)", "tab 1", "tab 2", "tab 3"}
	labelMax := max(maxInnerWidth(m.width)-28, 12)
	items := make([]menuItem, 0, len(dirs)+2)
	if m.anyRunning() { // quitting abandons an in-flight copy/move
		items = append(items,
			menuItem{header: true, warn: true, label: "!!! a task is still running"},
			menuItem{separator: true})
	}
	for i, d := range dirs {
		items = append(items, menuItem{
			label: truncPathLeft(shortPath(d), labelMax),
			key:   strconv.Itoa(i + 1),
			hint:  hints[i],
		})
	}
	m.quitMenu.setItems(items, "Quit — cd to…")
	m.quitMenu.setSize(m.width)
	return m.quitMenu.open()
}

// quitTo records the chosen dir for the shell wrapper, then shuts down.
func (m *AppModel) quitTo(dir string) tea.Cmd {
	writeLastDir(dir)
	return m.shutdown()
}
