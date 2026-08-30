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

// quitTarget is one cd-on-quit destination: a directory plus a short note on
// where it came from (the first source that offered it).
type quitTarget struct {
	dir  string
	hint string
}

// quitTargets is the cd-on-quit menu's distinct destinations: the launch dir
// (the launch dir) then each tab's dir, with duplicates dropped so a directory
// that is open in several places is offered once. Order is launch-first, then
// tab order; the hint names the first source that introduced each dir.
func (m AppModel) quitTargets() []quitTarget {
	var out []quitTarget
	seen := map[string]bool{}
	add := func(dir, hint string) {
		if dir == "" || seen[dir] {
			return
		}
		seen[dir] = true
		out = append(out, quitTarget{dir, hint})
	}
	// Hints are a single glyph + a trailing space: the launch glyph for the
	// launch dir, each tab's mark (the launch glyph again for tab one, Ⅱ..Ⅴ
	// beyond — dedupe usually folds tab one into the launch row anyway). One
	// cell of content keeps the right column aligned regardless of glyph width.
	add(m.launchDir, iconCWD+" ")
	for i, t := range m.tabs {
		add(t.dir, tabNumeral(i)+" ")
	}
	return out
}

// openQuitMenu opens the quit picker: pick where to leave the shell (a number or
// j/k + Enter), or Esc to stay.
func (m *AppModel) openQuitMenu() tea.Cmd {
	targets := m.quitTargets()
	labelMax := max(maxInnerWidth(m.width)-28, 12)
	items := make([]menuItem, 0, len(targets)+2)
	if m.anyRunning() { // quitting abandons an in-flight copy/move
		items = append(items,
			menuItem{header: true, warn: true, label: "!!! a task is still running"},
			menuItem{separator: true})
	}
	for i, t := range targets {
		items = append(items, menuItem{
			label: truncPathLeft(shortPath(t.dir), labelMax),
			key:   strconv.Itoa(i + 1),
			hint:  t.hint,
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
