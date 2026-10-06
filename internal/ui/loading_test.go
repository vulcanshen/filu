package ui

import (
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"
)

// pinClock fixes the loading icon's clock at d past the epoch.
func pinClock(t *testing.T, d time.Duration) {
	t.Helper()
	old := loadingNow
	loadingNow = func() time.Time { return time.Unix(0, int64(d)) }
	t.Cleanup(func() { loadingNow = old })
}

// tdp D3: the loading icon is nf-md-circle_slice_1..8 (U+F0A9E–U+F0AA5), 90ms a
// frame, picked by the clock — frames[(now / 90ms) % 8]. Code points written out.
func TestD3LoadingIconFollowsTheClock(t *testing.T) {
	for _, tc := range []struct {
		at   time.Duration
		want rune
	}{
		{0, 0xf0a9e},
		{89 * time.Millisecond, 0xf0a9e},
		{90 * time.Millisecond, 0xf0a9f},
		{630 * time.Millisecond, 0xf0aa5},
		{720 * time.Millisecond, 0xf0a9e}, // a full turn
	} {
		pinClock(t, tc.at)
		if got := loadingIcon(); got != string(tc.want) {
			t.Errorf("at %v the icon is %U, want %U", tc.at, []rune(got)[0], tc.want)
		}
	}
}

// loadingFinder is a finder opened on root whose file walk has sent files but
// not finished.
func loadingFinder(root string, files ...string) searchModel {
	m := newSearch()
	m.setSize(120, 30)
	m.open(root, 120, 30, false, false, nil)
	m.anim.state = popupOpen
	m.onStreamBatch(fileBatchMsg{gen: m.openGen, root: root, batch: files})
	return m
}

// tdp F7: a finder whose results are still coming in shows the loading icon
// after its title; when the walk is done the icon goes, and the box keeps its
// size either way (L2).
func TestF7FinderShowsLoadingInItsTitle(t *testing.T) {
	pinClock(t, 0)
	icon := string(rune(0xf0a9e))
	m := loadingFinder("/root", "/root/a.go")
	loading := ansi.Strip(m.renderFull())
	if top := strings.SplitN(loading, "\n", 2)[0]; !strings.Contains(top, "Search "+icon) {
		t.Errorf("a loading finder should show the icon after its title:\n%s", top)
	}

	m.onStreamBatch(fileBatchMsg{gen: m.openGen, root: "/root", done: true})
	done := ansi.Strip(m.renderFull())
	if strings.Contains(done, icon) {
		t.Errorf("the icon should go once loading is done:\n%s", strings.SplitN(done, "\n", 2)[0])
	}
	if boxWidth(done) != boxWidth(loading) || boxRows(done) != boxRows(loading) {
		t.Errorf("the box changed size with the icon: %dx%d → %dx%d",
			boxWidth(loading), boxRows(loading), boxWidth(done), boxRows(done))
	}

	f := openedSearch("/root", "/root/a.go") // Find: rg running counts as loading
	f.searching = true
	if !strings.Contains(strings.SplitN(ansi.Strip(f.renderFull()), "\n", 2)[0], "Find "+icon) {
		t.Error("a content search in flight should show the icon after the title")
	}
}

// User ruling 2026-09-28: results show as they stream in — the list is not
// held back until the walk ends; "(indexing…)" only while nothing has come.
func TestFinderListsResultsWhileLoading(t *testing.T) {
	empty := loadingFinder("/root")
	if !strings.Contains(ansi.Strip(empty.renderFull()), "(indexing…)") {
		t.Error("with nothing streamed in yet the list should say it is indexing")
	}
	m := loadingFinder("/root", "/root/alpha.go")
	out := ansi.Strip(m.renderFull())
	if !strings.Contains(out, "alpha.go") || strings.Contains(out, "(indexing…)") {
		t.Errorf("results streamed in so far should be listed while loading:\n%s", out)
	}
}

// tdp D3: the icon tick runs only while something loads. It starts when a
// finder opens loading, re-arms on each tick while loading lasts, and stops
// once nothing is loading.
func TestD3LoadingTickOnlyWhileLoading(t *testing.T) {
	m := minModel()
	m.search = loadingFinder("/root", "/root/a.go")
	if m.keepLoading() == nil || !m.loadingTicking {
		t.Fatal("a loading finder should start the icon tick")
	}
	if m.keepLoading() != nil {
		t.Error("a second tick should not start while one is in flight")
	}

	model, cmd := m.Update(loadingTickMsg{})
	m = model.(AppModel)
	if cmd == nil || !m.loadingTicking {
		t.Error("the tick should re-arm while the finder is still loading")
	}

	m.search.onStreamBatch(fileBatchMsg{gen: m.search.openGen, root: "/root", done: true})
	model, cmd = m.Update(loadingTickMsg{})
	m = model.(AppModel)
	if cmd != nil || m.loadingTicking {
		t.Error("the tick should stop once nothing is loading")
	}

	m.tasks = []landTask{{id: 1, status: taskRunning}} // a running task loads too
	if m.keepLoading() == nil {
		t.Error("a running task should start the icon tick")
	}
}

// tdp D3 (user ruling 2026-09-28): a running task in [3] Tasks turns the same
// family icon, not the old braille dots.
func TestD3TasksUseTheLoadingIcon(t *testing.T) {
	pinClock(t, 90*time.Millisecond)
	m := minModel()
	line := ansi.Strip(m.taskLine(landTask{id: 1, action: "cp", dest: "d", total: 2, srcs: []string{"a", "b"}, status: taskRunning}))
	if !strings.Contains(line, string(rune(0xf0a9f))+" Copying") {
		t.Errorf("a running task should show the loading icon before its verb: %q", line)
	}
	for r := rune(0x2800); r <= 0x28ff; r++ {
		if strings.ContainsRune(line, r) {
			t.Errorf("the braille spinner is still there: %q", line)
		}
	}
}

// Every way a finder opens starts the icon tick, and a key typed into a loading
// finder (a new rg search, a re-anchored walk) re-arms it if it had stopped.
func TestD3FinderOpenersStartTheTick(t *testing.T) {
	old := lookPath
	lookPath = func(string) (string, error) { return "/usr/bin/rg", nil }
	defer func() { lookPath = old }()

	for name, open := range map[string]func(m *AppModel){
		"search": func(m *AppModel) { m.openSearch() },
		"find":   func(m *AppModel) { m.openFind() },
		"goto":   func(m *AppModel) { m.openGoto() },
	} {
		m := minModel()
		m.search = newSearch()
		open(&m)
		if !m.loadingTicking {
			t.Errorf("opening the %s finder should start the loading icon tick", name)
		}
	}

	m := minModel()
	m.search = loadingFinder("/root", "/root/a.go")
	m = press(t, m, runes("a"))
	if !m.loadingTicking {
		t.Error("typing into a loading finder should keep the icon turning")
	}
}

// Where icons take two cells the loading icon does too; the top border shortens
// to match, so the box stays square (tdp L4). Measured on one box: beside the
// finder's preview the join pads rows and would hide a border cell too many.
func TestF7LoadingIconKeepsTheBorderOnWideIconFonts(t *testing.T) {
	old := iconCells
	iconCells = 2
	defer func() { iconCells = old }()

	lines := strings.Split(drawPopupBoxPad(popupLayerColor(1), " Search "+loadingIcon(), " hint ", []string{"x"}, 30, false), "\n")
	for i, l := range lines {
		if got := dispWidth(l); got != 32 {
			t.Errorf("row %d is %d cells on a two-cell icon font, want 32: %q", i, got, ansi.Strip(l))
		}
	}
}
