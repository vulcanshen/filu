package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

func TestAncestorChain(t *testing.T) {
	// An absolute path outside $HOME so shortPath leaves it untouched (machine-
	// independent). Chain is root-first, given dir last.
	got := ancestorChain("/a/b/c")
	wantPaths := []string{"/", "/a", "/a/b", "/a/b/c"}
	if len(got) != len(wantPaths) {
		t.Fatalf("len = %d, want %d: %+v", len(got), len(wantPaths), got)
	}
	for i, w := range wantPaths {
		if got[i].path != w {
			t.Errorf("level %d path = %q, want %q", i, got[i].path, w)
		}
		if got[i].label != w { // outside home, label == path
			t.Errorf("level %d label = %q, want %q", i, got[i].label, w)
		}
	}
	// A trailing slash must not add an empty level.
	if c := ancestorChain("/a/b/c/"); len(c) != 4 {
		t.Errorf("trailing slash changed chain length: %d", len(c))
	}
	// The filesystem root is a single level.
	if c := ancestorChain("/"); len(c) != 1 || c[0].path != "/" {
		t.Errorf("root chain = %+v, want single /", c)
	}
}

func TestBreadcrumbPopupFlow(t *testing.T) {
	open := func() breadcrumbPopup {
		m := newBreadcrumbPopup()
		m.open("/a/b/c")
		m.anim.state = popupOpen // skip the open animation
		return m
	}

	// Cursor starts on the current (deepest) level → Enter jumps to it.
	m := open()
	if m.cursor != 3 {
		t.Fatalf("cursor starts at %d, want 3 (current)", m.cursor)
	}
	_, path, cmd := m.update(tea.KeyMsg{Type: tea.KeyEnter})
	if path != "/a/b/c" || cmd == nil {
		t.Errorf("Enter on current: path=%q cmd=%v, want /a/b/c + close", path, cmd)
	}

	// k moves up the chain; Enter jumps to the selected ancestor.
	m = open()
	m, _, _ = m.update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")}) // → /a/b
	m, _, _ = m.update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")}) // → /a
	_, path, _ = m.update(tea.KeyMsg{Type: tea.KeyEnter})
	if path != "/a" {
		t.Errorf("two k + Enter: path=%q, want /a", path)
	}

	// g jumps to root, G back to current.
	m = open()
	m, _, _ = m.update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("g")})
	if m.cursor != 0 {
		t.Errorf("g cursor = %d, want 0", m.cursor)
	}
	m, _, _ = m.update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("G")})
	if m.cursor != 3 {
		t.Errorf("G cursor = %d, want 3", m.cursor)
	}

	// Esc / b close without a jump.
	for _, key := range []tea.KeyMsg{{Type: tea.KeyEsc}, {Type: tea.KeyRunes, Runes: []rune("b")}} {
		m = open()
		_, path, cmd = m.update(key)
		if path != "" || cmd == nil {
			t.Errorf("%v should close without a jump: path=%q cmd=%v", key, path, cmd)
		}
	}
}

// TestBreadcrumbKeyOpensAndJumps wires the whole path: [b] on panel [2] opens
// the popup, k moves up a level, Enter teleports the active tab to that ancestor.
func TestBreadcrumbKeyOpensAndJumps(t *testing.T) {
	root := t.TempDir()
	nested := filepath.Join(root, "a", "b")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	m := minModel()
	m.width, m.height = 80, 24
	m.tabs = []listModel{newList(nested)}
	m.tab = 0

	// b: open the breadcrumb popup on the active tab.
	model, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("b")})
	m = model.(AppModel)
	if !m.breadcrumb.isActive() {
		t.Fatal("b should open the breadcrumb popup")
	}
	m.breadcrumb.anim.state = popupOpen // skip the open animation → interactive

	// k moves the cursor up one level (from .../a/b to .../a), Enter jumps there.
	model, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	m = model.(AppModel)
	model, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = model.(AppModel)

	if got, want := m.cur().dir, filepath.Join(root, "a"); got != want {
		t.Errorf("after k+Enter the tab dir = %q, want ancestor %q", got, want)
	}
	if m.breadcrumb.isInteractive() {
		t.Error("Enter should start closing the popup (no longer interactive)")
	}
}

func TestBreadcrumbRender(t *testing.T) {
	m := newBreadcrumbPopup()
	m.setSize(100)
	m.open("/a/b/c")
	m.anim.state = popupOpen
	plain := ansi.Strip(m.renderFull())
	for _, want := range []string{"Breadcrumb", "/a/b/c", "jump", "close"} {
		if !strings.Contains(plain, want) {
			t.Errorf("breadcrumb popup missing %q:\n%s", want, plain)
		}
	}
}
