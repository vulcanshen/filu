package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

var (
	m6Dim    = [3]int{0x58, 0x5b, 0x70} // disabledColor, Surface2: a row that can't run now
	m6Header = [3]int{0x7f, 0x84, 0x9c} // a region header, Overlay1
)

// keyRefRow finds the key reference row for key and reports whether it is
// marked disabled.
func keyRefRow(t *testing.T, rows []helpRow, key string) helpRow {
	t.Helper()
	for _, r := range rows {
		if !r.header && r.key == key {
			return r
		}
	}
	t.Fatalf("no key reference row for %q", key)
	return helpRow{}
}

// keyRefColours draws rows as the key reference does and returns the colour
// of key and of its description in the row for key.
func keyRefColours(t *testing.T, rows []helpRow, key string) (k, d [3]int) {
	t.Helper()
	h := newHelpPopup()
	h.setSize(100, 80)
	h.open("Keys", rows)
	for _, line := range strings.Split(h.renderFull(), "\n") {
		plain := ansi.Strip(line)
		if !strings.HasPrefix(plain, "│  "+key+" ") {
			continue
		}
		fg := cellFG(line)
		at := cellAt(t, line, key)
		rest := strings.TrimLeft(plain[len("│  "+key):], " ")
		return fg[at], fg[cellAt(t, line, rest[:3])]
	}
	t.Fatalf("no drawn row for %q", key)
	return
}

// tdp M6 (v0.1.14): a key whose target is there but can't run now is listed in
// the key reference dimmed — key and description — as its menu row is.
func TestM6KeyReferenceDimsWhatCannotRun(t *testing.T) {
	truecolor(t)
	m := f4Model(t)
	m.tabs = m.tabs[:1] // one tab: nothing to switch to or close
	_, rows := m.panelKeyRef()
	for _, key := range []string{"l", "w"} {
		if !keyRefRow(t, rows, key).disabled {
			t.Errorf("with one tab, %q should be dimmed in the [1] key reference", key)
		}
		k, d := keyRefColours(t, rows, key)
		if !near(k, m6Dim) || !near(d, m6Dim) {
			t.Errorf("%q row is %v / %v, want both dimmed %v", key, k, d, m6Dim)
		}
	}
	if keyRefRow(t, rows, "t").disabled {
		t.Error("with one tab, a new tab can be made: t should not be dimmed")
	}
	if k, d := keyRefColours(t, rows, "t"); !near(k, m5Blue) || !near(d, m5Text) {
		t.Errorf("t row is %v / %v, want Blue / Text", k, d)
	}
	h := newHelpPopup()
	h.setSize(100, 80)
	h.open("Keys", rows)
	headers := 0
	for _, line := range strings.Split(h.renderFull(), "\n") {
		if plain := ansi.Strip(line); strings.HasPrefix(plain, "│ panel operation") {
			headers++
			if got := cellFG(line)[cellAt(t, line, "panel")]; !near(got, m6Header) {
				t.Errorf("a region header is %v, want it left as it is %v", got, m6Header)
			}
		}
	}
	if headers != 1 {
		t.Errorf("found %d panel operation headers, want 1", headers)
	}

	for len(m.tabs) < maxTabs {
		m.tabs = append(m.tabs, m.tabs[0])
	}
	_, rows = m.panelKeyRef()
	if !keyRefRow(t, rows, "t").disabled {
		t.Error("with every tab in use, t should be dimmed")
	}

	m.setSortColumnItems()
	if !keyRefRow(t, menuKeyRef(m.sortMenu, nil), "r").disabled {
		t.Error("with no sort, the Sort key reference should dim r")
	}

	m.places.pinned = []place{{path: "/tmp"}}
	m.openOpenInMenu()
	if !keyRefRow(t, menuKeyRef(m.openInMenu, nil), "n").disabled {
		t.Error("with every tab in use, the Open in key reference should dim n")
	}
}

// tdp M6: on an empty preview (an empty directory, an unreadable file, an empty
// [1]) there is nothing to open or copy — Yank is dimmed in the menu and y and
// Enter in the key reference; with content they are not.
func TestM6EmptyPreviewDimsYankAndEnter(t *testing.T) {
	m := f4Model(t)
	m.focus = panelDetail
	m.refreshPreview()
	if !m.previewHasBody() {
		t.Fatal("the fixture's a.txt should preview")
	}
	check := func(want bool) {
		t.Helper()
		items, _ := m.buildSpaceMenu()
		for _, it := range items {
			if it.key == "y" && it.disabled != want {
				t.Errorf("Yank disabled = %v, want %v", it.disabled, want)
			}
		}
		_, rows := m.panelKeyRef()
		for _, key := range []string{"y", "Enter"} {
			if got := keyRefRow(t, rows, key).disabled; got != want {
				t.Errorf("%s dimmed = %v, want %v", key, got, want)
			}
		}
	}
	check(false)

	m.tabs[0] = newList(t.TempDir()) // an empty directory: no selection
	m.refreshPreview()
	if m.previewHasBody() {
		t.Fatal("an empty [1] should leave the preview empty")
	}
	check(true)

	m.focus = panelList // Enter on [1] still goes into a directory
	if _, rows := m.panelKeyRef(); keyRefRow(t, rows, "Enter").disabled {
		t.Error("Enter on [1] is not the preview's: it should not dim")
	}
}

// tdp M6: the bottom-border hints name only keys that work now (the app's
// choice): h/l with one tab, and o / D with no favorites, are left out.
func TestM6HintsListWhatWorks(t *testing.T) {
	if got := ansi.Strip(listNavHint(true, 1)); strings.Contains(got, "h/l") {
		t.Errorf("with one tab there is no tab to switch to: %q", got)
	}
	if got := ansi.Strip(listNavHint(true, 2)); !strings.Contains(got, "h/l:switch tab") {
		t.Errorf("with two tabs h/l switches: %q", got)
	}
	if got := favoritesHint(false); got != "" {
		t.Errorf("with no favorites the edge should stay clean, got %q", got)
	}
}
