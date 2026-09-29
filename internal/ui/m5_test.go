package ui

import (
	"math"
	"regexp"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

// bottomHint is the hint in a box's bottom border: what sits between "╰─" and
// the run of dashes that closes the border.
func bottomHint(box string) string {
	lines := strings.Split(ansi.Strip(box), "\n")
	row := strings.TrimPrefix(lines[len(lines)-1], "╰─")
	if i := strings.Index(row, "─"); i >= 0 {
		row = row[:i]
	}
	return row
}

// cellAt is the cell index where sub first appears in the plain text of line.
func cellAt(t *testing.T, line, sub string) int {
	t.Helper()
	plain := ansi.Strip(line)
	i := strings.Index(plain, sub)
	if i < 0 {
		t.Fatalf("%q not in %q", sub, plain)
	}
	return ansi.StringWidth(plain[:i])
}

var (
	m5Blue     = [3]int{0x89, 0xb4, 0xfa} // keys in hints, footer and key references
	m5Overlay0 = [3]int{0x6c, 0x70, 0x86} // a hint's colon and description
	m5Text     = [3]int{0xcd, 0xd6, 0xf4} // a key reference's description
)

// tdp M5: every hint and the footer read "key:what", one space between the
// items — each written out in full, so a stray separator anywhere shows.
func TestM5HintsAndFooter(t *testing.T) {
	boxes := f7Popups(t, 100, 40)
	cases := map[string]string{
		"menu":          " j/k:move Enter:run Esc:close ",
		"confirm":       " Enter:delete Esc:cancel ",
		"input":         " Enter:create Esc:cancel ",
		"key reference": " j/k:scroll ?/Esc:close ",
		"metadata":      " j/k:scroll Esc:close ",
		"breadcrumb":    " j/k:move Enter:jump Esc:close ",
		"viewport":      " v:select y:copy all Esc:close ",
	}
	for name, want := range cases {
		if got := bottomHint(boxes[name]); got != want {
			t.Errorf("%s hint = %q, want %q", name, got, want)
		}
	}

	vp := newDetailYank()
	vp.visual = true
	fs := newSearch()
	typing := fs.hint(200)
	fs.mode = searchNav
	m := f4Model(t)
	m.width = 100
	m.gotoFavMenu.setSize(100, 40)
	m.places.pinned = []place{{path: "/tmp"}}
	m.setGotoPinnedItems()
	for name, c := range map[string]struct{ got, want string }{
		"selecting":      {vp.hint(200), " y:copy Esc:leave ?:keys "},
		"finder typing":  {typing, " ↑/↓:move Enter:go Tab:list Esc:close "},
		"finder list":    {fs.hint(200), " j/k/u/d:move Enter:go Tab:query Esc:close "},
		"favorites menu": {bottomHint(m.gotoFavMenu.renderFull()), " j/k:move Enter:run f:unfavorite Esc:close "},
		"list panel":     {keyLegend(listNavHint(true, 2)), " Enter:into Esc:back j/k/u/d:move h/l:switch tab "},
		"marks panel":    {keyLegend(marksHint(true)), " p:pick m:unmark Z:zip C:clear "},
		"favorites tab":  {keyLegend(favoritesHint(true)), " o:open in D:remove "},
		"footer":         {strings.TrimRight(m.footerBar(60), " "), " Space:menu ?:help Tab/1–3:panels q:quit"},
	} {
		if got := ansi.Strip(c.got); got != c.want {
			t.Errorf("%s hint = %q, want %q", name, got, c.want)
		}
	}
	// The key belongs in the hint, not the title (a title is a label, M5).
	if m.gotoFavMenu.title != "Favorites" {
		t.Errorf("the Favorites menu title = %q, want just Favorites", m.gotoFavMenu.title)
	}
	m.places.pinned = nil
	m.setGotoPinnedItems()
	if got := bottomHint(m.gotoFavMenu.renderFull()); strings.Contains(got, "f:") {
		t.Errorf("with nothing favorited there is nothing to unfavorite: %q", got)
	}
}

// oldKeyNotation matches the ways keys were written before tdp M5: "+" for a
// modifier, lowercase key-cap names, "=" or " · " between items, and keys run
// together or spaced in one column ("j k", "jkud").
var oldKeyNotation = regexp.MustCompile(`(Alt|Ctrl|Shift)\+|\b(enter|esc|space|tab)\b| · |=|^\S+ \S+$|jkud|^hl$`)

// tdp M5: key reference key columns name keys as they are on the keyboard:
// "/" between keys that do the same thing, "–" for a range, "-" after a modifier.
func TestM5KeyReferenceKeys(t *testing.T) {
	m := f4Model(t)
	var rows []helpRow
	for _, f := range []panelID{panelList, panelDetail, panelMarks} {
		for tab := range 3 {
			m.focus, m.marksTab = f, tab
			_, r := m.panelKeyRef()
			rows = append(rows, r...)
		}
	}
	menu := newSpaceMenu()
	menu.setItems([]menuItem{{label: "Copy", key: "c"}}, "t")
	rows = append(rows, menuKeyRef(menu, nil)...)
	rows = append(rows, quitKeyRef(3)...)
	rows = append(rows, confirmKeyRef("delete")...)
	rows = append(rows, breadcrumbKeyRef()...)
	rows = append(rows, finderKeyRef()...)
	rows = append(rows, yankKeyRef()...)
	rows = append(rows, metaKeyRef()...)
	rows = append(rows, selectHelpRows()...)
	for _, r := range rows {
		if !r.header && oldKeyNotation.MatchString(r.key) {
			t.Errorf("key column %q (%s) is not written as tdp M5 has it", r.key, r.desc)
		}
	}
	want := map[string]string{"1–3": "", "j/k": "", "gg/G": "", "u/d": "", "Ctrl-C": "", "Enter/y": "", "Esc/n": "", "h/←": "", "v/Esc": ""}
	for _, r := range rows {
		if _, ok := want[r.key]; ok {
			want[r.key] = "seen"
		}
	}
	for k, seen := range want {
		if seen == "" {
			t.Errorf("no key reference row keyed %q", k)
		}
	}
}

// tdp M5: a key named in a sentence — a toast, an empty state, a menu or key
// reference description — is bracketed.
func TestM5KeysInSentences(t *testing.T) {
	m := f4Model(t)
	m.focus = panelList
	m.tabs = append(m.tabs, m.tabs[0], m.tabs[0])
	items, _ := m.buildSpaceMenu()
	var hints []string
	for _, it := range items {
		hints = append(hints, it.hint)
	}
	m.focus = panelMarks
	items, _ = m.buildSpaceMenu()
	for _, it := range items {
		hints = append(hints, it.hint)
	}
	_, rows := m.panelKeyRef()
	for _, r := range append(rows, yankKeyRef()...) {
		hints = append(hints, r.desc)
	}
	m.focus = panelDetail
	hints = append(hints, m.enterDesc())

	joined := strings.Join(hints, "\n")
	for _, want := range []string{"next tab [h]/[l]", "Marks / Tasks / Favorites [h]/[l]", "same as [q], even while typing",
		"open the scrollable view (same as [y])", "start selecting (then [?] lists its keys)"} {
		if !strings.Contains(joined, want) {
			t.Errorf("no description reads %q", want)
		}
	}

	var places placesModel
	m.places.pinned = nil
	m.setGotoPinnedItems()
	m.toast = newToast()
	m.showInTabs("/nowhere", "")
	if cfg := string(defaultConfigBytes()); !strings.Contains(cfg, "press [O] on a file") || !strings.Contains(cfg, "plain [o] just opens") {
		t.Error("the config template should bracket the keys it names")
	}
	for name, s := range map[string]string{
		"favorites tab":  ansi.Strip(places.view(60, 3, false)),
		"favorites menu": m.gotoFavMenu.items[0].label,
		"tabs full":      m.toast.message,
	} {
		if !regexp.MustCompile(`\[[a-z]\]`).MatchString(s) {
			t.Errorf("%s names its key without brackets: %q", name, s)
		}
	}
}

// tdp M5, D2: a hint's key is Blue and its colon and description Overlay0; a
// key reference's key is Blue and its description Text.
func TestM5KeyColours(t *testing.T) {
	truecolor(t)
	c := newConfirmPopup()
	c.setSize(100)
	c.open("Delete a?", "delete")
	lines := strings.Split(c.renderFull(), "\n")
	bottom := lines[len(lines)-1]
	fg := cellFG(bottom)
	for sub, want := range map[string][3]int{"Enter": m5Blue, ":delete": m5Overlay0, "delete": m5Overlay0, "Esc": m5Blue} {
		if got := fg[cellAt(t, bottom, sub)]; !near(got, want) {
			t.Errorf("confirm hint %q is %v, want %v", sub, got, want)
		}
	}

	var m AppModel
	footer := m.footerBar(60)
	fg = cellFG(footer)
	for sub, want := range map[string][3]int{"Space": m5Blue, ":menu": m5Overlay0, "Tab/1–3": m5Blue} {
		if got := fg[cellAt(t, footer, sub)]; !near(got, want) {
			t.Errorf("footer %q is %v, want %v", sub, got, want)
		}
	}

	h := newHelpPopup()
	h.setSize(100, 40)
	h.open("Keys", []helpRow{{header: true, desc: "keys"}, {key: "Tab", desc: "focus the next panel"}})
	for _, line := range strings.Split(h.renderFull(), "\n") {
		if !strings.Contains(ansi.Strip(line), "Tab") {
			continue
		}
		fg = cellFG(line)
		if got := fg[cellAt(t, line, "Tab")]; !near(got, m5Blue) {
			t.Errorf("key reference key is %v, want Blue %v", got, m5Blue)
		}
		if got := fg[cellAt(t, line, "focus")]; !near(got, m5Text) {
			t.Errorf("key reference description is %v, want Text %v", got, m5Text)
		}
		return
	}
	t.Fatal("no Tab row in the key reference")
}

// keyLegend is keyLegendFit with no width limit: every pair, for the tests
// that check what a hint says rather than what fits.
func keyLegend(pairs [][2]string) string { return keyLegendFit(pairs, math.MaxInt) }

// tdp D3, D1: a hint or footer that does not fit leaves pairs out from the end,
// whole — never cut in the middle, never an ellipsis.
func TestD3HintsDropWholePairs(t *testing.T) {
	m, _ := d6App(t) // 100 columns: the finder's list box is 37 inside
	m.search.open(t.TempDir(), m.width, m.height, false, false, make(chan fileBatchMsg, 1))
	m.search.anim.state = popupOpen
	typing := bottomHint(m.search.renderFull())
	m.search.mode = searchNav
	list := bottomHint(m.search.renderFull())
	var mm AppModel
	panel := strings.Split(mm.panelBoxHint(true, singleChip("[1]", true), listNavHint(true, 2), 40, 4, "body"), "\n")
	for _, c := range []struct{ name, got, want string }{
		{"finder typing", typing, " ↑/↓:move Enter:go Tab:list "},
		{"finder list", list, " j/k/u/d:move Enter:go Tab:query "},
		{"[1] at 40 columns", ansi.Strip(panel[3]), "╚ Enter:into Esc:back j/k/u/d:move ════╝"},
		{"footer at 30", ansi.Strip(mm.footerBar(30)), " Space:menu ?:help " + strings.Repeat(" ", 11)},
		{"footer just fits", ansi.Strip(mm.footerBar(41)), " Space:menu ?:help Tab/1–3:panels q:quit "},
		{"footer one short", ansi.Strip(mm.footerBar(40)), " Space:menu ?:help Tab/1–3:panels " + strings.Repeat(" ", 6)},
	} {
		if c.got != c.want {
			t.Errorf("%s = %q, want %q", c.name, c.got, c.want)
		}
	}
}
