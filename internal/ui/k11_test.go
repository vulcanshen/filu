package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

// yankModel is minModel with the yank viewport open on a few lines, selecting
// when select is true.
func yankModel(t *testing.T, selecting bool) AppModel {
	t.Helper()
	m := k9Model()
	m.help, m.modeList, m.detailYank = newHelpPopup(), newModeList(), newDetailYank()
	m.width, m.height = 100, 40
	m.detailYank.setSize(100, 40)
	m.detailYank.open("notes.txt", []string{"line one", "line two", "line three"}, false, nil)
	m.detailYank.anim.state = popupOpen
	if selecting {
		m = press(t, m, runes("v"))
		if !m.detailYank.visual {
			t.Fatal("setup: v should start selecting")
		}
	}
	return m
}

// tdp K11 / K5: Space lists the mode's keys while selecting; outside the mode
// the viewport is a plain popup and Space does nothing.
func TestK11SpaceListsModeKeysOnlyWhileSelecting(t *testing.T) {
	m := press(t, yankModel(t, true), runes(" "))
	if !m.modeList.owns() || !m.detailYank.owns() {
		t.Errorf("Space while selecting should open the key list over the viewport: list %v, viewport %v",
			m.modeList.owns(), m.detailYank.owns())
	}
	o := press(t, yankModel(t, false), runes(" "))
	if o.modeList.owns() || o.spaceMenu.owns() {
		t.Error("Space outside the selection should do nothing (tdp K5)")
	}
}

// tdp K11: in the key list the arrows move; every other key runs its row, and
// running a row closes the list.
func TestK11KeyListRunsRows(t *testing.T) {
	m := press(t, yankModel(t, true), runes(" "))

	m = press(t, m, tea.KeyMsg{Type: tea.KeyDown})
	if !m.modeList.owns() || m.detailYank.cursorLine != 0 || m.modeList.cursor != 1 {
		t.Fatalf("↓ should move the list, not the viewport: list open %v, list cursor %d, viewport line %d",
			m.modeList.owns(), m.modeList.cursor, m.detailYank.cursorLine)
	}
	m = press(t, m, runes("j")) // j runs "move down" — it does not move the list
	if m.modeList.owns() || m.detailYank.cursorLine != 1 {
		t.Errorf("j in the list should run move-down and close it: list %v, viewport line %d",
			m.modeList.owns(), m.detailYank.cursorLine)
	}

	m = press(t, m, runes(" "))
	m.modeList.cursor = 7 // G — bottom
	m = press(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.modeList.owns() || m.detailYank.cursorLine != 2 {
		t.Errorf("Enter should run the highlighted row (G): list %v, viewport line %d",
			m.modeList.owns(), m.detailYank.cursorLine)
	}

	m = press(t, m, runes(" "))
	m = press(t, m, runes(" "))
	if m.modeList.owns() || !m.detailYank.visual {
		t.Error("Space on the list should close it without running anything")
	}

	m = press(t, m, runes(" "))
	m = press(t, m, runes("v")) // v runs "leave the selection"
	if m.detailYank.visual {
		t.Error("v in the list should run leave-the-selection")
	}
}

// tdp K11 / M3: every key of the mode is in its key list.
func TestK11KeyListHasEveryModeKey(t *testing.T) {
	l := newModeList()
	l.setSize(100)
	out := ansi.Strip(l.renderFull())
	for _, k := range selectKeys {
		if !strings.Contains(out, k.keys) || !strings.Contains(out, k.desc) {
			t.Errorf("the key list is missing %q (%s):\n%s", k.keys, k.desc, out)
		}
	}
}

// tdp K11 / K6: ? while selecting is the mode's help; outside it is the
// viewport's key reference. Both come from the same table.
func TestK11QuestionIsModeHelp(t *testing.T) {
	m := press(t, yankModel(t, true), qmKey)
	if m.help.title != "Selection keys" {
		t.Fatalf("? while selecting should open the mode help, got %q", m.help.title)
	}
	have := map[string]bool{}
	for _, r := range m.help.rows {
		have[r.key] = true
	}
	for _, k := range selectKeys {
		if !have[k.keys] {
			t.Errorf("the mode help is missing %q", k.keys)
		}
	}
	o := press(t, yankModel(t, false), qmKey)
	if o.help.title != "Preview viewport keys" {
		t.Errorf("? outside the selection should be the viewport keys, got %q", o.help.title)
	}
}

// tdp K11: Tab is suspended in the mode but answers.
func TestK11TabAnswersWhileSelecting(t *testing.T) {
	m := press(t, yankModel(t, true), tea.KeyMsg{Type: tea.KeyTab})
	if !m.toast.owns() || !strings.Contains(m.toast.message, "Esc leaves the selection") {
		t.Errorf("Tab while selecting should say how to leave: toast %v %q", m.toast.owns(), m.toast.message)
	}
	if !m.detailYank.visual {
		t.Error("Tab should not leave the selection")
	}
}

// The hint follows the state; the box width does not (tdp L2).
func TestK11HintFollowsStateWidthHolds(t *testing.T) {
	m := yankModel(t, false)
	out := ansi.Strip(m.detailYank.renderFull())
	w := ansi.StringWidth(strings.Split(out, "\n")[0])
	if !strings.Contains(out, "v select") {
		t.Errorf("outside the selection the hint should offer v:\n%s", out)
	}
	m = press(t, m, runes("v"))
	sel := ansi.Strip(m.detailYank.renderFull())
	if !strings.Contains(sel, "Space keys") || !strings.Contains(sel, "Esc leave") {
		t.Errorf("while selecting the hint should offer Space and Esc:\n%s", sel)
	}
	if got := ansi.StringWidth(strings.Split(sel, "\n")[0]); got != w {
		t.Errorf("the box width changed with the state: %d → %d", w, got)
	}
}

// The key list is wired like every other popup.
func TestK11KeyListWiring(t *testing.T) {
	m := yankModel(t, true)
	model, _ := m.Update(tea.KeyMsg{Type: tea.KeySpace, Runes: []rune(" ")})
	m = model.(AppModel)
	for i := 0; i < 20 && !m.modeList.isInteractive(); i++ {
		model, _ = m.Update(AnimTickMsg{Target: "modelist"})
		m = model.(AppModel)
	}
	if !m.modeList.isInteractive() {
		t.Fatal("the key list never finished opening: AnimTickMsg is not reaching it")
	}
	if out := ansi.Strip(m.View()); !strings.Contains(out, "Selection keys") {
		t.Errorf("View should draw the key list over the viewport:\n%s", out)
	}
	model, _ = m.Update(tea.WindowSizeMsg{Width: 90, Height: 40})
	if model.(AppModel).modeList.screenW != 90 {
		t.Error("WindowSizeMsg should size the key list")
	}
}

// tdp M3 / K11: every key detailYank.update handles while selecting has a row
// in the table — a key the viewport answers to but the list leaves out would be
// one only a memorised hotkey reaches. Keep this list in step with update.
func TestK11TableCoversViewportKeys(t *testing.T) {
	triggers := map[string]bool{}
	for _, k := range selectKeys {
		triggers[k.trigger()] = true
	}
	for _, key := range []string{"h", "l", "j", "k", "0", "$", "g", "G", "u", "d", "y", "v"} {
		if !triggers[key] {
			t.Errorf("the viewport answers to %q while selecting, but the key list has no row for it", key)
		}
	}
}
