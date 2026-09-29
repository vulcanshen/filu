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
	m.help, m.detailYank = newHelpPopup(), newDetailYank()
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

// tdp K11 (v0.1.10) / K5: a mode has no Space menu and no key list — Space
// while selecting does nothing: no popup opens and the selection stays as it
// is. Outside the mode the viewport is a plain popup and Space does nothing
// either.
func TestK11SpaceDoesNothingInTheMode(t *testing.T) {
	for _, selecting := range []bool{true, false} {
		m := yankModel(t, selecting)
		m = press(t, m, runes("l"))
		line, col := m.detailYank.cursorLine, m.detailYank.cursorCol
		m = press(t, m, runes(" "))
		for _, a := range m.stackOrder() {
			if a != &m.detailYank.anim && a.owns() {
				t.Errorf("selecting=%v: Space opened a popup", selecting)
			}
		}
		if !m.detailYank.owns() || m.detailYank.visual != selecting ||
			m.detailYank.cursorLine != line || m.detailYank.cursorCol != col {
			t.Errorf("selecting=%v: Space should leave the viewport as it was", selecting)
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
	if have["Space"] {
		t.Error("Space does nothing in the mode; its help should not list it (tdp K11)")
	}
	o := press(t, yankModel(t, false), qmKey)
	if o.help.title != "Preview viewport keys" {
		t.Errorf("? outside the selection should be the viewport keys, got %q", o.help.title)
	}
}

// tdp K11: Tab is suspended in the mode but answers.
func TestK11TabAnswersWhileSelecting(t *testing.T) {
	m := press(t, yankModel(t, true), tea.KeyMsg{Type: tea.KeyTab})
	if !m.toast.owns() || !strings.Contains(m.toast.message, "[Esc] leaves the selection") {
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
	if !strings.Contains(out, "v:select") {
		t.Errorf("outside the selection the hint should offer v:\n%s", out)
	}
	m = press(t, m, runes("v"))
	sel := ansi.Strip(m.detailYank.renderFull())
	if !strings.Contains(sel, "?:keys") || !strings.Contains(sel, "Esc:leave") || strings.Contains(sel, "Space") {
		t.Errorf("while selecting the hint should offer ? and Esc, not Space:\n%s", sel)
	}
	if got := ansi.StringWidth(strings.Split(sel, "\n")[0]); got != w {
		t.Errorf("the box width changed with the state: %d → %d", w, got)
	}
}

// tdp M3 / K11: every key detailYank.update handles while selecting has a row
// in the table — a key the viewport answers to but the mode's key reference
// leaves out would be one only a memorised hotkey reaches. Keep this list in
// step with update.
func TestK11TableCoversViewportKeys(t *testing.T) {
	triggers := map[string]bool{}
	for _, k := range selectKeys {
		triggers[k.trigger()] = true
	}
	for _, key := range []string{"h", "l", "j", "k", "0", "$", "g", "G", "u", "d", "y", "v"} {
		if !triggers[key] {
			t.Errorf("the viewport answers to %q while selecting, but the mode's key reference has no row for it", key)
		}
	}
}
