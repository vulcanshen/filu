package ui

import (
	"os"
	"path/filepath"
	"slices"
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

// TestAbsAnchor: a leading-/ query resolves to (deepest existing directory the
// path names, scan depth). Uses a real temp tree so the os.Stat probes are stable.
func TestAbsAnchor(t *testing.T) {
	base := t.TempDir()
	if err := os.MkdirAll(filepath.Join(base, "usr", "local"), 0o755); err != nil {
		t.Fatal(err)
	}
	usr := filepath.Join(base, "usr")

	cases := []struct {
		query     string
		wantRoot  string
		wantDepth int
	}{
		{base + "/usr/lo", usr, absAnchorDepth},                                // /usr exists; "lo" is the fuzzy needle below it
		{base + "/u/lo", base, absAnchorDepth},                                 // /u doesn't exist → anchor at base, needle "u/lo"
		{base + "/usr/", usr, 1},                                               // trailing slash → rest on /usr, list its level
		{base + "/usr", base, absAnchorDepth},                                  // last segment is the needle, never folded into root
		{base + "/usr/local/x/y", filepath.Join(usr, "local"), absAnchorDepth}, // stops at the first missing segment
	}
	for _, c := range cases {
		root, depth := absAnchor(c.query)
		if root != c.wantRoot || depth != c.wantDepth {
			t.Errorf("absAnchor(%q) = (%q, %d), want (%q, %d)", c.query, root, depth, c.wantRoot, c.wantDepth)
		}
	}
}

func TestEffectiveFilter(t *testing.T) {
	cases := []struct {
		query, root string
		byContent   bool
		want        string
	}{
		{"/usr/lo", "/usr", false, "lo"},
		{"/u/lo", "/", false, "u/lo"}, // the needle spans segments when the anchor stays at /
		{"/usr", "/", false, "usr"},
		{"/usr/", "/usr", false, ""},         // rests on the anchor dir → empty needle
		{"proj", "/whatever", false, "proj"}, // no leading slash → the whole query
		{"/foo", "/", true, "/foo"},          // by-content is never re-anchored
	}
	for _, c := range cases {
		m := searchModel{query: c.query, root: c.root, byContent: c.byContent}
		if got := m.effectiveFilter(); got != c.want {
			t.Errorf("effectiveFilter(query=%q root=%q byContent=%v) = %q, want %q", c.query, c.root, c.byContent, got, c.want)
		}
	}
}

// TestGotoSlashReanchors: a leading / in Goto (or filename Search) re-anchors the
// scan onto that absolute path (the deepest existing prefix), fuzzy on the
// remainder — so directories outside the home base root become reachable.
func TestGotoSlashReanchors(t *testing.T) {
	base := t.TempDir()
	if err := os.MkdirAll(filepath.Join(base, "usr", "local"), 0o755); err != nil {
		t.Fatal(err)
	}
	usr := filepath.Join(base, "usr")

	m := openedFinder("/home/x", false, true, "a", "b") // Goto, based at home
	if m.root != "/home/x" || m.curDepth != 0 {
		t.Fatalf("open should root at home recursively (root=%q depth=%d)", m.root, m.curDepth)
	}
	m, cmd := m.update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(usr + "/lo")})
	if cmd == nil {
		t.Fatal("crossing to a new root should return a restream cmd")
	}
	if m.root != usr || m.curDepth != absAnchorDepth {
		t.Fatalf("leading / should anchor at %q depth %d, got root=%q depth=%d", usr, absAnchorDepth, m.root, m.curDepth)
	}
	if !m.loading {
		t.Error("a rescan should re-enter the loading state")
	}
	// the anchor's listing streams in; the trailing "lo" fuzzy-filters it
	m.onStreamBatch(fileBatchMsg{gen: m.openGen, root: usr, batch: []string{"local/", "lib/"}, done: true})
	if len(m.files) != 1 || m.files[0].path != "local/" {
		t.Fatalf(`"…/usr/lo" should show only local/, got %v`, m.files)
	}
	if want := filepath.Join(usr, "local"); m.selectedAbs() != want {
		t.Errorf("selectedAbs = %q, want %q", m.selectedAbs(), want)
	}
}

// TestSlashFuzzyAcrossSegments: the (b) behaviour — the leading-/ needle
// fuzzy-matches across path segments, so "/<base>/u/lo" still finds usr/local
// even though /u doesn't exist (anchor stays at base, needle "u/lo").
func TestSlashFuzzyAcrossSegments(t *testing.T) {
	base := t.TempDir() // nothing named "u" under base → anchor stays at base
	m := openedFinder("/home/x", false, true, "seed")
	m, _ = m.update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(base + "/u/lo")})
	if m.root != base || m.curDepth != absAnchorDepth {
		t.Fatalf("anchor should stay at base depth %d, got root=%q depth=%d", absAnchorDepth, m.root, m.curDepth)
	}
	m.onStreamBatch(fileBatchMsg{gen: m.openGen, root: base, batch: []string{"usr/", "usr/local/", "var/"}, done: true})
	if !slices.ContainsFunc(m.files, func(f fileMatch) bool { return f.path == "usr/local/" }) {
		t.Fatalf(`needle "u/lo" should fuzzy-match usr/local across segments, got %v`, m.files)
	}
}

// TestSlashSameLevelFiltersInMemory: extending the needle without crossing an
// anchor/depth boundary filters in memory — no openGen bump, no stream reload.
func TestSlashSameLevelFiltersInMemory(t *testing.T) {
	base := t.TempDir()
	if err := os.MkdirAll(filepath.Join(base, "usr"), 0o755); err != nil {
		t.Fatal(err)
	}
	usr := filepath.Join(base, "usr")

	m := openedFinder("/home/x", false, true, "a")
	m, _ = m.update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(usr + "/lo")})
	m.onStreamBatch(fileBatchMsg{gen: m.openGen, root: usr, batch: []string{"local/", "lib/", "lost/"}, done: true})
	gen, n := m.openGen, len(m.allFiles)

	m, cmd := m.update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("c")}) // → "…/usr/loc"
	if cmd != nil {
		t.Error("same-level typing should not restream")
	}
	if m.openGen != gen || len(m.allFiles) != n {
		t.Errorf("same-level typing must not rescan (openGen %d→%d, allFiles %d→%d)", gen, m.openGen, n, len(m.allFiles))
	}
	if len(m.files) != 1 || m.files[0].path != "local/" {
		t.Fatalf(`"…/usr/loc" should narrow to local/, got %v`, m.files)
	}
}

// TestSlashBackToBase: deleting the leading-/ query returns the scan to the
// original base root, recursively. Uses a single-segment "/usr" (no os.Stat, so
// no dependency on the host filesystem) that anchors at / depth absAnchorDepth.
func TestSlashBackToBase(t *testing.T) {
	m := openedFinder("/home/x", false, true, "a")
	m, _ = m.update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/usr")})
	if m.root != "/" || m.curDepth != absAnchorDepth {
		t.Fatalf(`"/usr" should anchor at / depth %d (root=%q depth=%d)`, absAnchorDepth, m.root, m.curDepth)
	}
	for range "/usr" { // backspace the whole query away
		m, _ = m.update(tea.KeyMsg{Type: tea.KeyBackspace})
	}
	if m.query != "" {
		t.Fatalf("query should be empty, got %q", m.query)
	}
	if m.root != "/home/x" || m.curDepth != 0 {
		t.Errorf("empty query should return to the base root recursively, got root=%q depth=%d", m.root, m.curDepth)
	}
}

// TestWalkDirFilesDepth drives the fd-less fallback over a real tree: maxDepth 1
// lists only the root's direct children (no descent); maxDepth 0 recurses. This
// is the depth limit that keeps a / anchor at "/" from walking the whole disk.
func TestWalkDirFilesDepth(t *testing.T) {
	root := t.TempDir()
	for _, d := range []string{"a/a1", "b"} {
		if err := os.MkdirAll(filepath.Join(root, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "c.txt"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	a1 := filepath.Join("a", "a1")

	// depth 1, dirs only → the direct child dirs, no descent, no files
	got := walkDirFiles(root, true, 1)
	if !slices.Contains(got, "a") || !slices.Contains(got, "b") || slices.Contains(got, a1) || slices.Contains(got, "c.txt") {
		t.Errorf("depth-1 dirsOnly = %v, want [a b] only", got)
	}
	// depth 1, all → direct children incl. the file, still no descent
	got = walkDirFiles(root, false, 1)
	if !slices.Contains(got, "c.txt") || slices.Contains(got, a1) {
		t.Errorf("depth-1 all = %v, want c.txt present and a/a1 absent", got)
	}
	// depth 0 → recursive, descends into a/a1
	if got = walkDirFiles(root, true, 0); !slices.Contains(got, a1) {
		t.Errorf("recursive dirsOnly = %v, want a/a1 present", got)
	}
}

// TestWalkDirFilesIncludesHidden: Goto (dirsOnly) now lists hidden directories
// and descends into them, so ~/.m2 & friends are reachable (the ignore list still
// filters tool dirs, but a plain hidden dir is kept).
func TestWalkDirFilesIncludesHidden(t *testing.T) {
	root := t.TempDir()
	for _, d := range []string{".hidden/inner", "proj"} {
		if err := os.MkdirAll(filepath.Join(root, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	got := walkDirFiles(root, true, 0) // dirsOnly, recursive
	for _, want := range []string{".hidden", filepath.Join(".hidden", "inner"), "proj"} {
		if !slices.Contains(got, want) {
			t.Errorf("walkDirFiles(dirsOnly) should list hidden dir %q; got %v", want, got)
		}
	}
}

// TestResolvedQuery: a leading ~ / ~/ expands to $HOME so a ~/… path is treated
// as the absolute path it names; everything else is left untouched.
func TestResolvedQuery(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		t.Skip("no home dir")
	}
	cases := []struct {
		query string
		want  string
	}{
		{"~", home},
		{"~/.m2", home + "/.m2"},
		{"~/proj/x", home + "/proj/x"},
		{"/etc", "/etc"}, // already absolute → unchanged
		{".m2", ".m2"},   // bare name → unchanged
		{"~foo", "~foo"}, // ~ not followed by / → literal
	}
	for _, c := range cases {
		m := searchModel{query: c.query}
		if got := m.resolvedQuery(); got != c.want {
			t.Errorf("resolvedQuery(%q) = %q, want %q", c.query, got, c.want)
		}
	}
}
