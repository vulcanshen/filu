package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

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
