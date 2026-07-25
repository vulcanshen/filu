package ui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// openedSearch builds an interactive Find (by-content) finder with the all-files
// list already loaded (skipping the async fd + open animation). Paths need not
// exist — the preview just notes them unreadable, which the tests don't assert.
func openedSearch(root string, files ...string) searchModel {
	return openedFinder(root, true, false, files...)
}

// openedFinder is openedSearch with an explicit mode (byContent / dirsOnly).
func openedFinder(root string, byContent, dirsOnly bool, files ...string) searchModel {
	m := newSearch()
	m.setSize(120, 30)
	m.open(root, 120, 30, byContent, dirsOnly, nil) // discard the stream cmd; inject below
	m.anim.state = popupOpen                        // skip the open animation so update() is interactive
	m.onStreamBatch(fileBatchMsg{gen: m.openGen, root: root, batch: files, done: true})
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

// TestStreamBatchesAccumulate: streamed batches append to the list, a stale-gen
// batch is dropped, and the done batch ends the loading state.
func TestStreamBatchesAccumulate(t *testing.T) {
	m := newSearch()
	m.setSize(120, 30)
	m.open("/root", 120, 30, false, false, nil) // discard the stream cmd
	m.anim.state = popupOpen
	gen := m.openGen

	m.onStreamBatch(fileBatchMsg{gen: gen, root: "/root", batch: []string{"a.go", "b.go"}})
	if !m.loading || len(m.files) != 2 {
		t.Fatalf("first (non-done) batch: loading=%v files=%d, want true/2", m.loading, len(m.files))
	}
	m.onStreamBatch(fileBatchMsg{gen: gen - 1, root: "/root", batch: []string{"stale.go"}})
	if len(m.files) != 2 {
		t.Errorf("a stale-gen batch must be dropped, got %d files", len(m.files))
	}
	m.onStreamBatch(fileBatchMsg{gen: gen, root: "/root", batch: []string{"c.go"}, done: true})
	if m.loading || len(m.files) != 3 {
		t.Errorf("after done: loading=%v files=%d, want false/3", m.loading, len(m.files))
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

func TestFuzzyMatch(t *testing.T) {
	// non-contiguous subsequence across path segments
	if _, ok := fuzzyMatch("internal/ui/search.go", "uisrch"); !ok {
		t.Error("uisrch should fuzzy-match internal/ui/search.go")
	}
	if _, ok := fuzzyMatch("search.go", "xyz"); ok {
		t.Error("xyz is not a subsequence of search.go")
	}
	// a prefix / boundary match outranks the same query mid-word
	pre, _ := fuzzyMatch("app.go", "app")
	mid, _ := fuzzyMatch("myapp.go", "app")
	if pre <= mid {
		t.Errorf("prefix match (%d) should outrank mid-word (%d)", pre, mid)
	}
}

func TestSearchByNameFiltersInMemory(t *testing.T) {
	m := openedFinder("/root", false, false, "app.go", "search.go", "sub/app_test.go")

	// typing narrows the list in-memory (no rg cmd, not "searching")
	m, cmd := m.update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("app")})
	if cmd != nil || m.searching {
		t.Errorf("by-name filter should be synchronous (cmd=%v searching=%v)", cmd != nil, m.searching)
	}
	// fuzzy: "app" matches app.go and sub/app_test.go, but not search.go
	if len(m.files) != 2 {
		t.Fatalf("query %q should match app.go + sub/app_test.go, got %d: %v", "app", len(m.files), m.files)
	}
	if m.files[0].path != "app.go" {
		t.Errorf("best match for %q should rank app.go first, got %q", "app", m.files[0].path)
	}
	// the finder shows no preview in by-name mode
	if m.renderFull() == "" {
		t.Error("by-name finder should still render a list box")
	}

	// backspace to empty restores everything
	m, _ = m.update(tea.KeyMsg{Type: tea.KeyBackspace})
	m, _ = m.update(tea.KeyMsg{Type: tea.KeyBackspace})
	m, _ = m.update(tea.KeyMsg{Type: tea.KeyBackspace})
	if m.query != "" || len(m.files) != 3 {
		t.Errorf("empty query should restore all 3, got query=%q n=%d", m.query, len(m.files))
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

	// q (nav mode) returns to the input to refine the query
	m, _ = m.update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	if m.mode != searchInput {
		t.Fatal("q in nav mode should return to input mode")
	}

	// Esc closes the finder from either mode (like every other popup) — never
	// dropping back to input from nav.
	if _, cmd := m.update(tea.KeyMsg{Type: tea.KeyEsc}); cmd == nil {
		t.Error("Esc in input mode should close the finder")
	}
	m, _ = m.update(tea.KeyMsg{Type: tea.KeyEnter}) // back to nav
	if m.mode != searchNav {
		t.Fatal("Enter should switch back to nav mode")
	}
	m, cmd := m.update(tea.KeyMsg{Type: tea.KeyEsc})
	if cmd == nil {
		t.Error("Esc in nav mode should close the finder")
	}
	if m.mode != searchNav {
		t.Error("Esc in nav mode must close, not drop back to input")
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
