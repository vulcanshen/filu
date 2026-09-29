package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	overlay "github.com/rmhubbert/bubbletea-overlay"
)

// wideIcon is a Nerd Font icon: one cell on a normal font, two on a CJK icon font.
var wideIcon = string(rune(0xf015))

// d6App is the whole app, 100 × 30, on a directory whose names, previews and
// popups carry icons: a sub-directory (its tree previews with icons), a file,
// and a directory whose own name holds an icon.
func d6App(t *testing.T) (AppModel, string) {
	t.Helper()
	oldState, oldCfg := statePathOverride, configPathOverride
	statePathOverride = filepath.Join(t.TempDir(), "state.yaml")
	configPathOverride = filepath.Join(t.TempDir(), "config.yaml")
	t.Cleanup(func() { statePathOverride, configPathOverride = oldState, oldCfg })

	dir := t.TempDir()
	iconDir := filepath.Join(dir, "d"+wideIcon)
	for _, d := range []string{filepath.Join(dir, "subdir", "inner"), iconDir} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	for _, f := range []string{filepath.Join(dir, "subdir", "a.go"), filepath.Join(dir, "file.go"), filepath.Join(iconDir, "f"+wideIcon+".txt")} {
		if err := os.WriteFile(f, []byte("package x\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	m := New(dir, "")
	if m.watcher != nil {
		t.Cleanup(func() { m.watcher.Close() })
	}
	model, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	m = model.(AppModel)
	for i, it := range m.tabs[0].items { // the cursor on subdir: [2] previews its tree
		if it.name == "subdir" {
			m.tabs[0].cursor = i
		}
	}
	m.refreshPreview()
	return m, dir
}

// d6Popup opens one kind of popup and returns the box as it is drawn alone.
type d6Popup struct {
	name string
	open func(t *testing.T, m *AppModel, dir string) []string
}

func itemNamed(t *testing.T, l listModel, name string) fileItem {
	t.Helper()
	for _, it := range l.items {
		if it.name == name {
			return it
		}
	}
	t.Fatalf("no %q in %s", name, l.dir)
	return fileItem{}
}

var d6Popups = []d6Popup{
	{"Space menu", func(t *testing.T, m *AppModel, _ string) []string {
		*m = press(t, *m, runes(" "))
		return []string{m.spaceMenu.renderFull()}
	}},
	{"quit picker", func(t *testing.T, m *AppModel, _ string) []string {
		*m = press(t, *m, runes("q"))
		return []string{m.quitMenu.renderFull()}
	}},
	{"Open in", func(t *testing.T, m *AppModel, dir string) []string {
		m.places.pinned = []place{{path: filepath.Join(dir, "subdir")}}
		m.focus, m.marksTab = panelMarks, 2
		m.openOpenInMenu()
		m.openInMenu.anim.state = popupOpen
		return []string{m.openInMenu.renderFull()}
	}},
	{"finder", func(t *testing.T, m *AppModel, dir string) []string {
		m.search.open(dir, m.width, m.height, false, false, make(chan fileBatchMsg, 1))
		m.search.anim.state = popupOpen
		m.search.onStreamBatch(fileBatchMsg{gen: m.search.openGen, root: dir, batch: []string{"subdir", "file.go"}, done: true})
		_, sW, sRows, pW, pRows := m.search.geometry() // each box alone: joined, one box's error hides
		bc := popupLayerColor(1)
		return []string{
			drawPopupBoxPad(bc, " Search", m.search.hint(sW-1), m.search.listColumn(sW, sRows), sW, false),
			drawPopupBoxPad(bc, m.search.previewTitle(), "", m.search.previewColumn(pW, pRows), pW, false),
		}
	}},
	{"input", func(t *testing.T, m *AppModel, _ string) []string {
		m.openInput(inputRename, "Rename", "file.go", itemNamed(t, m.tabs[0], "file.go"))
		m.inputPopup.anim.state = popupOpen
		return []string{m.inputPopup.renderFull()}
	}},
	{"viewport", func(t *testing.T, m *AppModel, _ string) []string {
		m.focus = panelDetail
		m.openDetailYank()
		m.detailYank.anim.state = popupOpen
		return []string{m.detailYank.renderFull()}
	}},
	{"confirm", func(t *testing.T, m *AppModel, _ string) []string {
		m.confirm.open("Move file.go to the trash?", "trash")
		m.confirm.anim.state = popupOpen
		return []string{m.confirm.renderFull()}
	}},
	{"key reference", func(t *testing.T, m *AppModel, _ string) []string {
		*m = press(t, *m, runes("?"))
		return []string{m.help.renderFull()}
	}},
	{"metadata", func(t *testing.T, m *AppModel, dir string) []string {
		m.meta.open(filepath.Join(dir, "d"+wideIcon, "f"+wideIcon+".txt"))
		m.meta.anim.state = popupOpen
		return []string{m.meta.renderFull()}
	}},
	{"breadcrumb", func(t *testing.T, m *AppModel, dir string) []string {
		m.breadcrumb.open(filepath.Join(dir, "d"+wideIcon))
		m.breadcrumb.anim.state = popupOpen
		return []string{m.breadcrumb.renderFull()}
	}},
	{"toast", func(t *testing.T, m *AppModel, _ string) []string {
		m.toast.show("Copied " + wideIcon + " path")
		m.toast.anim.state = popupOpen
		return []string{m.toast.renderFull()}
	}},
}

// tdp D6, L4: with icons one cell wide and two, every popup — alone, and laid
// over the screen — keeps every line exactly as wide as it should be: the box
// its own width, the screen the terminal's.
func TestD6EveryPopupEveryLineExact(t *testing.T) {
	defer restoreIconCells(iconCells)
	for _, cells := range []int{1, 2} {
		for _, p := range d6Popups {
			iconCells = cells
			m, dir := d6App(t)
			for bi, box := range p.open(t, &m, dir) {
				lines := strings.Split(box, "\n")
				want := dispWidth(lines[0])
				for r, line := range lines {
					if got := dispWidth(line); got != want {
						t.Errorf("icons %d, %s box %d alone: row %d is %d wide, row 0 is %d\n  %q", cells, p.name, bi, r, got, want, ansi.Strip(line))
					}
				}
			}
			for r, line := range strings.Split(m.View(), "\n") {
				if got := dispWidth(line); got != m.width {
					t.Errorf("icons %d, %s on screen: row %d is %d wide, want %d\n  %q", cells, p.name, r, got, m.width, ansi.Strip(line))
				}
			}
		}
	}
}

// tdp D6: the quit picker keeps its right-hand glyph with icons two cells wide
// (measured as one, the room for it came out a cell short and it was cut to …).
func TestD6QuitPickerKeepsItsGlyph(t *testing.T) {
	defer restoreIconCells(iconCells)
	iconCells = 2
	m, _ := d6App(t)
	m = press(t, m, runes("q"))
	if out := ansi.Strip(m.quitMenu.renderFull()); !strings.Contains(out, iconCWD) {
		t.Errorf("the launch directory's glyph should show:\n%s", out)
	}
}

// tdp D6: a menu whose labels carry icons keeps its description column lined up,
// and a description too long for the row still ends in "…" (not cut off by the
// box one cell early).
func TestD6MenuColumnsWithIcons(t *testing.T) {
	defer restoreIconCells(iconCells)
	iconCells = 2
	menu := newSpaceMenu()
	menu.setSize(60, 30)
	menu.setItems([]menuItem{
		{label: "a" + wideIcon, key: "a", hint: "one"},
		{label: "bb", key: "b", hint: "two"},
		{label: "c" + wideIcon, key: "c", hint: strings.Repeat("long ", 20)},
	}, "t")
	menu.cursor = 1
	var cols []int
	var long string
	for _, line := range strings.Split(menu.renderFull(), "\n") {
		plain := ansi.Strip(line)
		for _, h := range []string{"one", "two", "long"} {
			if i := strings.Index(plain, h); i >= 0 {
				cols = append(cols, dispWidth(plain[:i]))
				if h == "long" {
					long = plain
				}
				break
			}
		}
	}
	if len(cols) != 3 || cols[0] != cols[1] || cols[1] != cols[2] {
		t.Errorf("the descriptions should start in one column, got cells %v", cols)
	}
	if !strings.HasSuffix(strings.TrimSuffix(long, "│"), "… ") {
		t.Errorf("a cut description should end in … and the row's one-cell margin: %q", long)
	}
}

// tdp D6: a value is wrapped by display width — one that fits when an icon is
// measured as one cell, but not as drawn, still wraps.
func TestD6WrapHardByDisplayWidth(t *testing.T) {
	defer restoreIconCells(iconCells)
	iconCells = 2
	got := wrapHard("a"+wideIcon+wideIcon+"b", 5)
	if want := []string{"a" + wideIcon + wideIcon, "b"}; strings.Join(got, "|") != strings.Join(want, "|") {
		t.Errorf("wrapHard = %q, want %q", got, want)
	}
}

// tdp D6: the finder's query row cuts a long query from the left by display
// width, so the kept tail reaches the count — icons cut away free their cells.
func TestD6FinderQueryCutByDisplayWidth(t *testing.T) {
	defer restoreIconCells(iconCells)
	iconCells = 2
	s := newSearch()
	s.mode, s.blink = searchInput, true
	s.query = strings.Repeat(wideIcon, 10) + strings.Repeat("x", 60)
	bar := ansi.Strip(s.inputBar(30))
	if !strings.HasSuffix(bar, "x█0 ") {
		t.Errorf("the query tail should run up to the count: %q", bar)
	}
	if got := dispWidth(bar); got != 30 {
		t.Errorf("the query row is %d wide, want 30", got)
	}
}

// tdp D6: the file information box wraps a path holding icons without losing a
// character, and every row is the box's width.
func TestD6MetaWrapsIconsWhole(t *testing.T) {
	defer restoreIconCells(iconCells)
	iconCells = 2
	dir := filepath.Join(t.TempDir(), strings.Repeat("d"+wideIcon, 12))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "f"+wideIcon+".txt")
	if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	meta := newMetaPopup()
	meta.setSize(40, 60)
	meta.open(path)
	lines := strings.Split(meta.renderFull(), "\n")
	var text strings.Builder
	for _, line := range lines {
		if got, want := dispWidth(line), dispWidth(lines[0]); got != want {
			t.Errorf("row %d is %d wide, the box %d: %q", len(text.String()), got, want, ansi.Strip(line))
		}
		text.WriteString(strings.TrimSpace(strings.Trim(ansi.Strip(line), "│")))
	}
	if flat := strings.ReplaceAll(text.String(), " ", ""); !strings.Contains(flat, strings.ReplaceAll(shortPath(path), " ", "")) {
		t.Errorf("the whole path should show across the wrapped rows:\n%s", strings.Join(lines, "\n"))
	}
}

// tdp D6: a long value in an input keeps its tail and its glyph; the row is the
// box's width.
func TestD6InputLongValueKeepsGlyph(t *testing.T) {
	defer restoreIconCells(iconCells)
	for _, cells := range []int{1, 2} {
		iconCells = cells
		in := newInputPopup()
		in.setSize(60)
		in.open(inputAdd, "New", strings.Repeat("x", 80)+"END", fileItem{})
		lines := strings.Split(in.renderFull(), "\n")
		row := ansi.Strip(lines[1])
		if !strings.HasPrefix(row, "│"+inputGlyph+"…") || !strings.Contains(row, "END") {
			t.Errorf("icons %d: the input row should keep the glyph and the tail: %q", cells, row)
		}
		if got, want := dispWidth(lines[1]), dispWidth(lines[0]); got != want {
			t.Errorf("icons %d: the input row is %d wide, the box %d", cells, got, want)
		}
	}
}

// tdp D6: the splash's pixels are two cells whatever the icon width — the glyph
// and a space, or the glyph alone where it is drawn two cells wide — so the logo
// rows line up.
func TestD6SplashPixelsTwoCells(t *testing.T) {
	defer restoreIconCells(iconCells)
	for _, cells := range []int{1, 2} {
		iconCells = cells
		s := newSplashModel()
		s.show()
		s.revealedCount = len(s.pixelOrder) / 2 // lit and unlit cells side by side
		logoW := len(logoPixels[0]) * 2
		for r, line := range strings.Split(s.render(logoW, 0), "\n")[:len(logoPixels)] {
			if got := dispWidth(line); got != logoW {
				t.Errorf("icons %d: splash row %d is %d wide, want the logo's %d", cells, r, got, logoW)
			}
		}
		for r, line := range strings.Split(s.render(100, 40), "\n") {
			if got := dispWidth(line); got != 100 {
				t.Errorf("icons %d: splash screen row %d is %d wide, want 100", cells, r, got)
			}
		}
	}
}

// compositeDisp lays fg over bg by display width. Each case is written out: a
// popup row with an icon, an icon under the popup, and an icon cut by the
// popup's left or right edge (the half left outside becomes a space).
func TestD6CompositeDisp(t *testing.T) {
	defer restoreIconCells(iconCells)
	iconCells = 2
	I := wideIcon
	for _, c := range []struct {
		name, fg, bg string
		x            int
		want         string
	}{
		{"icon in the popup", "a" + I + "b", "0123456789", 2, "01a" + I + "b6789"},
		{"icon under the popup", "XY", "ab" + I + "cdefg", 4, "ab" + I + "XYefg"},
		{"icon cut by the left edge", "XY", "a" + I + "bcdef", 2, "a XYcdef"},
		{"icon cut by the right edge", "XY", "abc" + I + "de", 2, "abXY de"},
	} {
		got := compositeDisp(c.fg, c.bg, overlay.Left, overlay.Top, c.x, 0)
		if got != c.want {
			t.Errorf("%s: got %q, want %q", c.name, got, c.want)
		}
		if dispWidth(got) != dispWidth(c.bg) {
			t.Errorf("%s: %d wide, the screen is %d", c.name, dispWidth(got), dispWidth(c.bg))
		}
	}
	// Placement matches overlay's: centred at half the screen less half the box.
	bg := strings.Repeat(strings.Repeat(".", 10)+"\n", 4) + strings.Repeat(".", 10)
	got := strings.Split(compositeDisp("AB\nCD", bg, overlay.Center, overlay.Center, 0, 0), "\n")
	if got[1] != "....AB...." || got[2] != "....CD...." {
		t.Errorf("centred box landed wrong:\n%s", strings.Join(got, "\n"))
	}
	got = strings.Split(compositeDisp("AB", bg, overlay.Center, overlay.Bottom, 0, -1), "\n")
	if got[3] != "....AB...." {
		t.Errorf("bottom box one row up landed wrong:\n%s", strings.Join(got, "\n"))
	}
}

// dispCutLeft drops display cells from the left; centerDisp centres like
// lipgloss.Place when there are no icons.
func TestD6CutLeftAndCenter(t *testing.T) {
	defer restoreIconCells(iconCells)
	iconCells = 2
	for _, c := range []struct {
		s    string
		n    int
		want string
	}{
		{"abcdef", 2, "cdef"},
		{"a" + wideIcon + "bc", 3, "bc"},
		{"a" + wideIcon + "bc", 2, " bc"}, // the icon's second half is all that is left of it
		{"ab", 5, ""},
	} {
		if got := dispCutLeft(c.s, c.n); got != c.want {
			t.Errorf("dispCutLeft(%q, %d) = %q, want %q", c.s, c.n, got, c.want)
		}
	}
	// An icon past the cut costs nothing; one before it costs its extra cell.
	for _, c := range []struct{ got, want string }{
		{dispClip("abc"+wideIcon+"de", 2), "ab"},
		{dispClip("a"+wideIcon+"bcd", 4), "a" + wideIcon + "b"},
		{truncate("abcdef"+wideIcon, 4), "abc…"},
		{truncate(wideIcon+"abcdef", 5), wideIcon + "ab…"},
		{truncPathLeft("d"+wideIcon+"/abcdef", 5), "…cdef"}, // the icon cut away frees two cells
	} {
		if c.got != c.want {
			t.Errorf("got %q, want %q", c.got, c.want)
		}
	}
	iconCells = 1
	for _, s := range []string{"ab", "abc\nd", "x\nlonger"} {
		for _, wh := range [][2]int{{7, 5}, {8, 4}, {3, 1}} {
			if got, want := centerDisp(wh[0], wh[1], s), lipgloss.Place(wh[0], wh[1], lipgloss.Center, lipgloss.Center, s); got != want {
				t.Errorf("centerDisp(%d, %d, %q) = %q, lipgloss.Place gives %q", wh[0], wh[1], s, got, want)
			}
		}
	}
}

// tdp L4, D6: in the preview viewport the cursor and the selection land on the
// right characters when a line holds wide (CJK) characters: the text is drawn
// as it is, and the marked cells are the ones asked for.
func TestD6ViewportCursorOnWideCharacters(t *testing.T) {
	truecolor(t)
	mark := lipgloss.NewStyle().Reverse(true)
	sel := lipgloss.NewStyle().Underline(true)
	for _, c := range []struct {
		name, line string
		out        string
		marked     string // the character wearing the cursor
		selS, selE int    // selection runes, -1 for a cursor alone
		cursorAt   int
	}{
		{"cursor after CJK", "中文abc", "中文abc", "a", -1, -1, 2},
		{"cursor between CJK", "a中b", "a中b", "b", -1, -1, 2},
		{"cursor on the last CJK", "中文", "中文", "文", -1, -1, 1},
		{"selection after CJK", "中文abc", "中文abc", "b", 2, 3, 3},
		{"selection over CJK", "x中文y", "x中文y", "文", 1, 2, 2},
	} {
		var got string
		if c.selS < 0 {
			got = overlayCursorOnStyledLine(c.line, c.line, c.cursorAt, mark)
		} else {
			got = overlaySelectionOnStyledLine(c.line, c.line, c.selS, c.selE, true, c.cursorAt, sel, mark)
		}
		if plain := ansi.Strip(got); plain != c.out {
			t.Errorf("%s: drawn as %q, want %q", c.name, plain, c.out)
		}
		if want := mark.Render(c.marked); !strings.Contains(got, want) {
			t.Errorf("%s: the cursor should be on %q: %q", c.name, c.marked, got)
		}
	}
}
