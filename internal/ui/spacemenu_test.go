package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

func TestBracketHotkey(t *testing.T) {
	cases := []struct{ label, key, want string }{
		{"Carry", "C", "[C]arry"},
		{"Copy here", "c", "[c]opy here"},   // case preserved (c vs C)
		{"Move here", "x", "[x] Move here"}, // key not in label → prefixed
		{"Hidden", ".", "[.] Hidden"},
		{"Delete", "D", "[D]elete"},
		{"Jump", "enter", "Jump"}, // multi-char key → plain
	}
	for _, c := range cases {
		if got := bracketHotkey(c.label, c.key); got != c.want {
			t.Errorf("bracketHotkey(%q, %q) = %q, want %q", c.label, c.key, got, c.want)
		}
	}
}

func TestSpaceMenuCommit(t *testing.T) {
	m := newSpaceMenu()
	m.setItems([]menuItem{{label: "Carry", key: "C"}, {label: "Rename", key: "R"}}, "x")
	m.anim.state = popupOpen // skip the open animation for the test

	if _, key, _ := m.update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("R")}); key != "R" {
		t.Errorf("direct hotkey R committed %q, want R", key)
	}
	moved, _, _ := m.update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	if _, key, _ := moved.update(tea.KeyMsg{Type: tea.KeyEnter}); key != "R" {
		t.Errorf("j+Enter committed %q, want R (2nd item)", key)
	}
	if _, key, cmd := m.update(tea.KeyMsg{Type: tea.KeyEsc}); key != "" || cmd == nil {
		t.Errorf("Esc should close without commit: key=%q cmd=%v", key, cmd)
	}
}

func TestSpaceMenuRender(t *testing.T) {
	m := newSpaceMenu()
	m.setSize(100)
	m.setItems([]menuItem{{label: "Carry", key: "C", hint: "add to bucket"}}, "README.md")
	plain := ansi.Strip(m.renderFull())
	for _, want := range []string{"README.md", "[C]arry", "add to bucket", "Space close"} {
		if !strings.Contains(plain, want) {
			t.Errorf("popup missing %q:\n%s", want, plain)
		}
	}
}

func TestBuildSpaceMenuList(t *testing.T) {
	m := AppModel{focus: panelList}
	m.tabs[0] = listModel{dir: "/tmp", items: []fileItem{{name: "foo.txt"}}}
	items, title := m.buildSpaceMenu()
	if title != "foo.txt" {
		t.Errorf("title = %q, want foo.txt", title)
	}
	keys := map[string]bool{}
	for _, it := range items {
		keys[it.key] = true
	}
	for _, k := range []string{"C", "R", "A", "D", "."} {
		if !keys[k] {
			t.Errorf("panel [2] menu missing hotkey %q", k)
		}
	}
	if keys["P"] {
		t.Error("Pin should be hidden for a non-dir cursor item")
	}
}
