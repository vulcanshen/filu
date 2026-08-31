package ui

import (
	"strconv"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

// TestHelpPanelDigitsMatchPanels pins the help popup's panel-digit hint to the
// panels that actually exist. The row read "1 2 3 4" for several releases after
// the 3-panel redesign, promising a panel the app has no key for.
func TestHelpPanelDigitsMatchPanels(t *testing.T) {
	var digits []string
	for p := panelList; p <= panelMarks; p++ {
		digits = append(digits, strconv.Itoa(int(p)))
	}
	want := strings.Join(digits, " ")
	for _, r := range helpRows {
		if r.desc == "focus a panel directly" {
			if r.key != want {
				t.Errorf("help lists panel keys %q, but the panels are %q", r.key, want)
			}
			return
		}
	}
	t.Fatal("help popup has no panel-digit row")
}

func TestHelpPopupRender(t *testing.T) {
	m := newHelpPopup()
	m.setSize(100)
	m.open()
	plain := ansi.Strip(m.renderFull())
	for _, want := range []string{"Help", "Tab", "Space", "quit", "esc close"} {
		if !strings.Contains(plain, want) {
			t.Errorf("help popup missing %q:\n%s", want, plain)
		}
	}
}

func TestHelpPopupDismiss(t *testing.T) {
	m := newHelpPopup()
	m.open()
	m.anim.state = popupOpen
	for _, k := range []string{"esc", "?", " ", "q"} {
		mm := m
		if _, cmd := mm.update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(k)}); k != "esc" && cmd == nil {
			t.Errorf("%q should close the help popup", k)
		}
	}
	if _, cmd := m.update(tea.KeyMsg{Type: tea.KeyEsc}); cmd == nil {
		t.Error("esc should close the help popup")
	}
}
