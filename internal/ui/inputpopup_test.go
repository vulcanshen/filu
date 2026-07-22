package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

func TestInputPopup(t *testing.T) {
	m := newInputPopup()
	m.open(inputRename, "Rename", "ab", "ab")
	m.anim.state = popupOpen // skip the open animation

	m, _, _ = m.update(tea.KeyMsg{Type: tea.KeyBackspace})
	if m.buffer != "a" {
		t.Errorf("backspace: buffer=%q, want a", m.buffer)
	}
	m, _, _ = m.update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("XY")})
	if m.buffer != "aXY" {
		t.Errorf("runes: buffer=%q, want aXY", m.buffer)
	}
	if _, committed, cmd := m.update(tea.KeyMsg{Type: tea.KeyEnter}); !committed || cmd == nil {
		t.Errorf("enter should commit and close: committed=%v cmd=%v", committed, cmd)
	}
	if _, committed, _ := m.update(tea.KeyMsg{Type: tea.KeyEsc}); committed {
		t.Error("esc should not commit")
	}
}

func TestInputPopupRender(t *testing.T) {
	m := newInputPopup()
	m.setSize(100)
	m.open(inputAdd, "New (trailing / = dir)", "hello", "")
	plain := ansi.Strip(m.renderFull())
	for _, want := range []string{"New", "hello", "confirm", "cancel"} {
		if !strings.Contains(plain, want) {
			t.Errorf("input popup missing %q:\n%s", want, plain)
		}
	}
}
