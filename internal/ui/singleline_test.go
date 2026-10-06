package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

// A single-line value keeps a pasted line break or tab, draws it as a Red \n /
// \t, and refuses to submit while it holds one; every other control character
// is dropped (the family input survey, 2026-10-06).

var redRGB = [3]int{0xf3, 0x8b, 0xa8}

// paste is a bracketed paste: one KeyRunes holding the whole text, control
// characters included.
func paste(s string) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s), Paste: true}
}

// rowWith is the first row of box whose plain text holds sub.
func rowWith(t *testing.T, box, sub string) string {
	t.Helper()
	for _, row := range strings.Split(box, "\n") {
		if strings.Contains(ansi.Strip(row), sub) {
			return row
		}
	}
	t.Fatalf("no row holds %q:\n%s", sub, ansi.Strip(box))
	return ""
}

// escapeIsRed checks both cells of the escape sub in row are Red, and the cell
// before it is not.
func escapeIsRed(t *testing.T, row, sub string) {
	t.Helper()
	fg := cellFG(row)
	i := cellAt(t, row, sub)
	if !near(fg[i], redRGB) || !near(fg[i+1], redRGB) {
		t.Errorf("%s should be Red, got %v %v: %q", sub, fg[i], fg[i+1], row)
	}
	if near(fg[i-1], redRGB) {
		t.Errorf("the cell before %s should be the value's colour: %q", sub, row)
	}
}

func openAdd(t *testing.T, m AppModel) AppModel {
	t.Helper()
	m.handleListKey("a")
	m.inputPopup.anim.state = popupOpen
	m.inputPopup.blink = false
	return m
}

// A paste lands as it came, \r\n as one break that Backspace takes whole; an
// ESC never reaches the value.
func TestInputPasteKeepsBreaks(t *testing.T) {
	m := openAdd(t, f4Model(t))
	m = press(t, m, paste("12\r\n34\x1b[31m\t"))
	if got := m.inputPopup.buffer; got != "12\n34[31m\t" {
		t.Fatalf("value = %q, want %q", got, "12\n34[31m\t")
	}
	for range 8 {
		m = press(t, m, tea.KeyMsg{Type: tea.KeyBackspace})
	}
	if got := m.inputPopup.buffer; got != "12" {
		t.Errorf("Backspace should take a break whole: %q, want %q", got, "12")
	}
}

// The value's breaks and tabs are drawn as a Red \n / \t, on the one field row;
// a \ and n typed by hand stay the value's colour.
func TestInputDrawsBreaksRed(t *testing.T) {
	truecolor(t)
	m := f4Model(t)
	m.width = 80
	m.inputPopup.setSize(80)
	m = openAdd(t, m)
	plain := strings.Count(m.inputPopup.renderFull(), "\n")

	m = press(t, m, paste("x\ny\tz"))
	m = press(t, m, runes(`\n`))
	box := m.inputPopup.renderFull()
	if got := strings.Count(box, "\n"); got != plain {
		t.Fatalf("the box has %d rows, want %d — the break must not split the field:\n%s", got+1, plain+1, ansi.Strip(box))
	}
	row := rowWith(t, box, `x\ny\tz\n`)
	escapeIsRed(t, row, `\ny`)
	escapeIsRed(t, row, `\tz`)
	fg := cellFG(row)
	if i := cellAt(t, row, `z\n`) + 1; near(fg[i], redRGB) || near(fg[i+1], redRGB) {
		t.Errorf("a typed \\n should be the value's colour: %q", row)
	}
}

// Enter refuses a value holding a line break or a tab — even at either end,
// where trimming would have hidden it — and says which field; nothing is
// created. With the break gone it goes through.
func TestInputRefusesBreaks(t *testing.T) {
	for _, v := range []string{"new\n", "ne\rw", "\tnew"} {
		m := openAdd(t, f4Model(t))
		dir := m.tabs[0].dir
		m = press(t, m, paste(v))
		m = press(t, m, enterKey)
		if !m.inputPopup.owns() {
			t.Fatalf("%q should not submit", v)
		}
		if got := ansi.Strip(m.inputPopup.renderFull()); !strings.Contains(got, "A name can't have line breaks or tabs") {
			t.Errorf("%q: the popup should say why:\n%s", v, got)
		}
		if entries, _ := os.ReadDir(dir); len(entries) != 1 {
			t.Errorf("%q: a refused name must create nothing, the dir holds %d entries", v, len(entries))
		}
	}

	m := openAdd(t, f4Model(t))
	m = press(t, m, paste("new\n"))
	m = press(t, m, tea.KeyMsg{Type: tea.KeyBackspace})
	m = press(t, m, enterKey)
	if m.inputPopup.owns() {
		t.Fatal("with the break gone the name should submit")
	}
	if _, err := os.Stat(filepath.Join(m.tabs[0].dir, "new")); err != nil {
		t.Error("the name without the break should be created")
	}
}

// A prefilled value goes through the same filter: macOS's custom-icon file
// Icon\r opens Rename with a Red \n that has to go before Enter renames, and an
// ESC in a name never reaches the field.
func TestRenamePrefillKeepsBreaks(t *testing.T) {
	truecolor(t)
	m := f4Model(t)
	dir := m.tabs[0].dir
	for _, name := range []string{"Icon\r", "e\x1bx"} {
		if err := os.WriteFile(filepath.Join(dir, name), nil, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	m.tabs[0] = newList(dir)
	m.width = 80
	m.inputPopup.setSize(80)
	rename := func(name string) AppModel {
		l := m.cur()
		for i, it := range l.items {
			if it.name == name {
				l.cursor = i
			}
		}
		m.handleListKey("r")
		m.inputPopup.anim.state = popupOpen
		m.inputPopup.blink = false
		return m
	}

	m = rename("e\x1bx")
	if got := m.inputPopup.buffer; got != "ex" {
		t.Errorf("a prefilled ESC should be dropped: %q", got)
	}
	m.inputPopup.anim.state = popupClosed

	m = rename("Icon\r")
	if got := m.inputPopup.buffer; got != "Icon\r" {
		t.Fatalf("the prefilled value should keep its break: %q", got)
	}
	escapeIsRed(t, rowWith(t, m.inputPopup.renderFull(), `Icon\n`), `\n`)
	m = press(t, m, enterKey)
	if !m.inputPopup.owns() {
		t.Fatal("Icon\\r should not rename while the break is in the value")
	}
	if _, err := os.Lstat(filepath.Join(dir, "Icon\r")); err != nil {
		t.Error("the refused rename must leave Icon\\r alone")
	}
	m = press(t, m, tea.KeyMsg{Type: tea.KeyBackspace})
	m = press(t, m, enterKey)
	if _, err := os.Lstat(filepath.Join(dir, "Icon")); err != nil {
		t.Error("with the break deleted, Icon\\r should rename to Icon")
	}
}

// Zip's suggested name is prefilled through the same filter.
func TestZipPrefillDropsControls(t *testing.T) {
	m := minModel()
	m.inputPopup = newInputPopup()
	m.focus = panelMarks
	m.marks.items = []string{"/tmp/pi\x1bcs.png"}
	m.handleMarksKey("Z")
	if got := m.inputPopup.buffer; got != "pics.zip" {
		t.Errorf("the suggestion should drop the ESC: %q", got)
	}
}

// The finder's query keeps a pasted break or tab, drawn Red on its one row; on
// the list, the whole row is one grey, the \n too. It is never refused.
func TestFinderQueryKeepsBreaks(t *testing.T) {
	truecolor(t)
	s := newSearch()
	s.anim.state = popupOpen
	s.mode, s.blink = searchInput, false
	s, _ = s.update(paste("a\r\nb\x1b\tc"))
	if s.query != "a\nb\tc" {
		t.Fatalf("query = %q, want %q", s.query, "a\nb\tc")
	}
	bar := s.inputBar(40)
	if strings.Contains(bar, "\n") || strings.Contains(bar, "\t") {
		t.Fatalf("the query row must stay one row: %q", bar)
	}
	escapeIsRed(t, bar, `\nb`)
	escapeIsRed(t, bar, `\tc`)

	s.mode = searchNav
	bar = s.inputBar(40)
	for i, c := range cellFG(bar) {
		if !near(c, hexRGB(dimColor)) {
			t.Fatalf("cell %d is %v; on the list the query row is one grey: %q", i, c, bar)
		}
	}
	if !strings.Contains(ansi.Strip(bar), `a\nb\tc`) {
		t.Errorf("the grey row should still show the escapes: %q", ansi.Strip(bar))
	}
}

// singleLine: a break or tab stays, \r\n as one; C0, DEL and C1 go.
func TestSingleLine(t *testing.T) {
	for in, want := range map[string]string{
		"plain":               "plain",
		"12\r\n34":            "12\n34",
		"a\rb\nc\td":          "a\rb\nc\td",
		"x\x1b[31my":          "x[31my",
		"a\x00b\x7fc\u0085dé": "abcdé",
	} {
		if got := singleLine(in); got != want {
			t.Errorf("singleLine(%q) = %q, want %q", in, got, want)
		}
	}
}

// valueTail keeps the tail in w cells and never cuts a \n in half: a cut
// landing inside one drops it, and a space keeps its cell.
func TestValueTailNeverCutsAnEscape(t *testing.T) {
	for w, want := range map[int]string{
		9: ` ab\ncd█`,
		8: ` ab\ncd█`,
		7: `…b\ncd█`,
		6: `…\ncd█`,
		5: `… cd█`,
		4: `…cd█`,
		0: ``,
	} {
		got := valueTail(" ", "ab\ncd", "█", w, false)
		if got != want {
			t.Errorf("w %d: %q, want %q", w, got, want)
		}
		if w > 0 && w < 8 && dispWidth(got) != w {
			t.Errorf("w %d: drawn %d cells wide", w, dispWidth(got))
		}
	}
	if got := valueTail("", "a\tb\rc", "", 9, false); got != `a\tb\nc` {
		t.Errorf("a tab is \\t and a lone \\r a \\n: %q", got)
	}
}
