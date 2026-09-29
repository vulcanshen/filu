package ui

import (
	"strconv"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

// panel1KeyRef is panel [1]'s key reference with one file under the cursor.
func panel1KeyRef() (string, []helpRow) {
	m := AppModel{focus: panelList}
	m.tabs = []listModel{{dir: "/tmp", items: []fileItem{{name: "foo.txt"}}}}
	return m.panelKeyRef()
}

// TestHelpPanelDigitsMatchPanels pins the key reference's panel-digit row to
// the panels that actually exist. The row read "1 2 3 4" for several releases
// after the 3-panel redesign, promising a panel the app has no key for. A range
// is written first–last with an en dash (tdp M5).
func TestHelpPanelDigitsMatchPanels(t *testing.T) {
	want := strconv.Itoa(int(panelList)) + "–" + strconv.Itoa(int(panelMarks))
	_, rows := panel1KeyRef()
	for _, r := range rows {
		if r.desc == "focus a panel directly" {
			if r.key != want {
				t.Errorf("key reference lists panel keys %q, but the panels are %q", r.key, want)
			}
			return
		}
	}
	t.Fatal("key reference has no panel-digit row")
}

func TestHelpPopupRender(t *testing.T) {
	m := newHelpPopup()
	m.setSize(100, 60)
	m.open(panel1KeyRef())
	plain := ansi.Strip(m.renderFull())
	for _, want := range []string{"[1] foo.txt keys", "Tab", "Space", "quit", "?/Esc:close"} {
		if !strings.Contains(plain, want) {
			t.Errorf("key reference missing %q:\n%s", want, plain)
		}
	}
}

func TestHelpPopupDismiss(t *testing.T) {
	m := newHelpPopup()
	m.open("x keys", []helpRow{{key: "a", desc: "b"}})
	m.anim.state = popupOpen
	if _, cmd := m.update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("?")}); cmd == nil {
		t.Error("? should close the key reference (tdp K6)")
	}
	if _, cmd := m.update(tea.KeyMsg{Type: tea.KeyEsc}); cmd == nil {
		t.Error("esc should close the key reference")
	}
	// q is the leave flow (tdp K9) and Space only toggles the Space menu (K5).
	for _, key := range []tea.KeyMsg{{Type: tea.KeyRunes, Runes: []rune("q")}, {Type: tea.KeySpace, Runes: []rune(" ")}} {
		if _, cmd := m.update(key); cmd != nil {
			t.Errorf("%q should not close the key reference", key.String())
		}
	}
}
