package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

func TestConfirmPopup(t *testing.T) {
	m := newConfirmPopup()
	m.open("Delete foo?")
	m.anim.state = popupOpen // skip the open animation

	if _, ok, _ := m.update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")}); !ok {
		t.Error("y should confirm")
	}
	if _, ok, cmd := m.update(tea.KeyMsg{Type: tea.KeyEsc}); ok || cmd == nil {
		t.Errorf("esc should cancel and close: ok=%v cmd=%v", ok, cmd)
	}
	if _, ok, _ := m.update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")}); ok {
		t.Error("n should not confirm")
	}
}

func TestConfirmRender(t *testing.T) {
	m := newConfirmPopup()
	m.setSize(100)
	m.open("Move README.md to the trash?")
	plain := ansi.Strip(m.renderFull())
	for _, want := range []string{"Confirm", "README.md", "trash", "confirm", "cancel"} {
		if !strings.Contains(plain, want) {
			t.Errorf("confirm popup missing %q:\n%s", want, plain)
		}
	}
}

func TestWrapWords(t *testing.T) {
	got := wrapWords("the quick brown fox", 10)
	for _, l := range got {
		if len(l) > 10 {
			t.Errorf("line exceeds width 10: %q", l)
		}
	}
	if strings.Join(got, " ") != "the quick brown fox" {
		t.Errorf("words lost/reordered: %v", got)
	}
}
