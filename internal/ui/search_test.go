package ui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// openedSearch builds an interactive finder with the all-files list already
// loaded (skipping the async fd + open animation). Paths need not exist — the
// preview just notes them unreadable, which the tests don't assert on.
func openedSearch(root string, files ...string) searchModel {
	m := newSearch()
	m.setSize(120, 30)
	m.open(root, 120, 30)
	m.anim.state = popupOpen // skip the open animation so update() is interactive
	m.onFilesLoaded(filesLoadedMsg{gen: m.gen, root: root, files: files})
	return m
}

func TestParseGrepLine(t *testing.T) {
	p, n, ok := parseGrepLine("internal/ui/app.go:100:func New() AppModel {")
	if !ok || p != "internal/ui/app.go" || n != 100 {
		t.Errorf("parse = (%q,%d,%v)", p, n, ok)
	}
	// a colon inside the matched text must stay with the text, not split
	p, n, ok = parseGrepLine("a.go:10:x := map[string]int{}")
	if !ok || p != "a.go" || n != 10 {
		t.Errorf("colon-in-text parse = (%q,%d,%v)", p, n, ok)
	}
	if _, _, ok := parseGrepLine("notaline"); ok {
		t.Error("a line without path:line:text must not parse")
	}
	if _, _, ok := parseGrepLine("a.go:notanumber:x"); ok {
		t.Error("a non-numeric line field must not parse")
	}
}

func TestSearchShowsAllFilesOnOpen(t *testing.T) {
	m := openedSearch("/root", "a.go", "b.go", "sub/c.go")
	if m.loading {
		t.Error("files loaded → not loading")
	}
	if len(m.files) != 3 {
		t.Fatalf("empty query should list all 3 files, got %d", len(m.files))
	}
}

func TestSearchTypingFiltersByContent(t *testing.T) {
	m := openedSearch("/root", "a.go", "b.go", "c.go")

	m, cmd := m.update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})
	if m.query != "x" || cmd == nil || !m.searching {
		t.Fatalf("typing should schedule a grep + mark searching (query=%q searching=%v)", m.query, m.searching)
	}

	// rg comes back with the subset of files that matched
	m.onGrepResult(grepFilesMsg{gen: m.gen, root: "/root", matches: []fileMatch{{path: "b.go", line: 12}}})
	if len(m.files) != 1 || m.files[0].path != "b.go" {
		t.Fatalf("grep result should replace the list with matched files, got %v", m.files)
	}
	if m.selectedLine() != 12 {
		t.Errorf("selectedLine = %d, want 12 (the first-match line)", m.selectedLine())
	}

	// deleting back to empty restores the full list instantly
	m, _ = m.update(tea.KeyMsg{Type: tea.KeyBackspace})
	if m.query != "" || m.searching {
		t.Errorf("empty query should stop searching; query=%q searching=%v", m.query, m.searching)
	}
	if len(m.files) != 3 {
		t.Errorf("empty query should restore all 3 files, got %d", len(m.files))
	}
}

func TestSearchModalFlow(t *testing.T) {
	m := openedSearch("/root", "a.go", "b.go")

	m, _ = m.update(tea.KeyMsg{Type: tea.KeyEnter}) // → nav
	if m.mode != searchNav {
		t.Fatal("Enter should switch to nav mode")
	}
	if m.selectedAbs() != "/root/a.go" {
		t.Errorf("selectedAbs = %q, want /root/a.go", m.selectedAbs())
	}

	m, _ = m.update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	if m.cursor != 1 || m.selectedAbs() != "/root/b.go" {
		t.Errorf("j → cursor %d (%q)", m.cursor, m.selectedAbs())
	}

	if _, cmd := m.update(tea.KeyMsg{Type: tea.KeyEnter}); cmd == nil {
		t.Error("Enter in nav mode should return a confirm cmd")
	}

	m, _ = m.update(tea.KeyMsg{Type: tea.KeyEsc})
	if m.mode != searchInput {
		t.Error("Esc in nav mode should return to input mode")
	}
}

func TestSearchEnterNeedsFiles(t *testing.T) {
	m := openedSearch("/root") // no files
	m, _ = m.update(tea.KeyMsg{Type: tea.KeyEnter})
	if m.mode != searchInput {
		t.Error("Enter with no files must stay in input mode")
	}
}

func TestSearchStaleGrepDropped(t *testing.T) {
	m := openedSearch("/root", "a.go")
	m.gen = 5
	m.onGrepResult(grepFilesMsg{gen: 4, root: "/root", matches: []fileMatch{{path: "b.go"}}})
	if m.files[0].path != "a.go" {
		t.Error("a stale (older gen) grep result must be dropped")
	}
	m.onGrepResult(grepFilesMsg{gen: 5, root: "/other", matches: []fileMatch{{path: "c.go"}}})
	if m.files[0].path != "a.go" {
		t.Error("a grep result for a different root must be dropped")
	}
	m.onGrepResult(grepFilesMsg{gen: 5, root: "/root", matches: []fileMatch{{path: "d.go"}}})
	if len(m.files) != 1 || m.files[0].path != "d.go" {
		t.Error("the current-gen result should install")
	}
}
