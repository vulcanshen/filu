package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

var qmKey = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("?")}

// tdp K6 / M4: a panel's key reference lists every key its Space menu offers —
// built from the same rows, so the two cannot drift — plus the core keys. The
// Global operation row has no key to press, so it is not listed.
func TestK6PanelKeyRefListsMenuKeys(t *testing.T) {
	am := AppModel{focus: panelList}
	am.tabs = []listModel{{dir: "/tmp", items: []fileItem{{name: "foo.txt"}}}, {dir: "/tmp"}}
	items, _ := am.buildSpaceMenu()
	_, rows := am.panelKeyRef()
	have := map[string]bool{}
	for _, r := range rows {
		have[r.key] = true
		if strings.Contains(r.desc, "Global operation") || r.key == globalOpKey {
			t.Errorf("the Global operation row has no key and should not be listed: %+v", r)
		}
	}
	for _, it := range items {
		if it.key != "" && it.key != globalOpKey && !have[it.key] {
			t.Errorf("menu key %q (%s) is missing from the key reference", it.key, it.label)
		}
	}
	for _, k := range []string{"Tab", "Enter", "Esc", "Space", "?", "q"} {
		if !have[k] {
			t.Errorf("core key %q is missing from the key reference", k)
		}
	}
}

// tdp K6 / D3 / F4: ? on any popup opens that popup's own key reference over
// it; the popup stays beneath, and Esc returns to it.
func TestK6QuestionOnEveryPopup(t *testing.T) {
	titles := map[string]string{
		"detailYank": "Preview viewport keys", "meta": "File information keys", "search": "Finder keys", "breadcrumb": "Breadcrumb keys",
		"confirm": "Confirm keys", "spaceMenu": "Space menu keys", "globalMenu": "Global operation keys",
		"sortMenu": "Sort keys", "sortDirMenu": "Sort direction keys", "gotoMenu": "Goto keys", "gotoFavMenu": "Favorites keys", "openInMenu": "Open in keys",
		"searchMenu": "Search keys", "openWithMenu": "Open with keys",
	}
	for name, want := range titles {
		t.Run(name, func(t *testing.T) {
			m := k9Model()
			m.help = newHelpPopup()
			openUnder(&m, name)
			m = press(t, m, qmKey)
			if !m.help.owns() || m.help.title != want {
				t.Fatalf("? on %s should open %q, got owns %v title %q", name, want, m.help.owns(), m.help.title)
			}
			if !stillOpen(m, name) {
				t.Fatalf("the key reference should stack on %s, not replace it", name)
			}
			m = press(t, m, escKey)
			if m.help.owns() || !stillOpen(m, name) {
				t.Errorf("Esc on the key reference should return to %s", name)
			}
		})
	}
}

// ? on a panel opens that panel's key reference.
func TestK6QuestionOnPanel(t *testing.T) {
	m := f4Model(t)
	m.help = newHelpPopup()
	m = press(t, m, qmKey)
	if !m.help.owns() || m.help.title != "[1] a.txt keys" {
		t.Errorf("? on panel [1] should open its key reference, got %q", m.help.title)
	}
	m = press(t, m, qmKey)
	if m.help.owns() {
		t.Error("? again should close the key reference (tdp K6)")
	}
}

// tdp K8: while typing, ? is a character, not the key reference.
func TestK6QuestionIsTypedWhileTyping(t *testing.T) {
	m := f4Model(t)
	m.help = newHelpPopup()
	m.handleListKey("a")
	m.inputPopup.anim.state = popupOpen
	m = press(t, m, qmKey)
	if m.help.owns() || m.inputPopup.buffer != "?" {
		t.Errorf("? in the input should be typed: help %v, buffer %q", m.help.owns(), m.inputPopup.buffer)
	}
	f := f4Model(t)
	f.help = newHelpPopup()
	f.search.anim.state = popupOpen
	f.search.mode = searchInput
	f = press(t, f, qmKey)
	if f.help.owns() || f.search.query != "?" {
		t.Errorf("? on the finder query should be typed: help %v, query %q", f.help.owns(), f.search.query)
	}
}

// tdp K6 / D3: the quit picker's ? opens a key reference of its own over it;
// Esc returns to the picker, and Ctrl-C there still leaves at once (K9).
func TestK6QuitPickerHasItsOwnKeyRef(t *testing.T) {
	m := k9Model()
	m.help, m.quitHelp = newHelpPopup(), newQuitHelp()
	m.openQuitMenu()
	m.quitMenu.anim.state = popupOpen
	m = press(t, m, qmKey)
	if !m.quitHelp.owns() || m.help.owns() || !m.quitMenu.owns() {
		t.Fatalf("? on the quit picker should open quitHelp over it: quitHelp %v, help %v, picker %v",
			m.quitHelp.owns(), m.help.owns(), m.quitMenu.owns())
	}
	if _, cmd := m.Update(ctrlCKey); !isQuitCmd(cmd) {
		t.Error("Ctrl-C on the quit picker's key reference should leave at once")
	}
	m = press(t, m, escKey)
	if m.quitHelp.owns() || !m.quitMenu.owns() {
		t.Error("Esc on the quit picker's key reference should return to the picker")
	}
}

// tdp D3: the key reference is drawn over the popup it describes, and the quit
// picker's over the quit picker.
func TestK6KeyRefDrawnOnTop(t *testing.T) {
	m := f4Model(t)
	m.help, m.quitHelp, m.globalMenu = newHelpPopup(), newQuitHelp(), newGlobalMenu()
	model, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 50})
	m = model.(AppModel)
	m = press(t, m, runes(" "))
	m = press(t, m, qmKey)
	if out := ansi.Strip(m.View()); !strings.Contains(out, "Space menu keys") {
		t.Errorf("the key reference should be drawn over the Space menu:\n%s", out)
	}

	q := k9Model()
	q.help, q.quitHelp = newHelpPopup(), newQuitHelp()
	q.width, q.height = 100, 50
	q.quitHelp.setSize(100, 50)
	q.openQuitMenu()
	q.quitMenu.anim.state = popupOpen
	q = press(t, q, qmKey)
	if out := ansi.Strip(q.View()); !strings.Contains(out, "Quit keys") {
		t.Errorf("quitHelp should be drawn over the quit picker:\n%s", out)
	}
}

// tdp K6: the key reference scrolls — no cursor, the window moves — and never
// outgrows the screen.
func TestK6KeyRefScrolls(t *testing.T) {
	m := newHelpPopup()
	m.setSize(80, 12)
	m.open(panel1KeyRef())
	m.anim.state = popupOpen
	first := ansi.Strip(m.renderFull())
	if n := len(strings.Split(first, "\n")); n > 12-2 {
		t.Fatalf("key reference is %d rows on a 12-row screen", n)
	}
	m, _ = m.update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("G")})
	last := ansi.Strip(m.renderFull())
	if first == last || !strings.Contains(last, "Ctrl-C") {
		t.Errorf("G should scroll to the last rows:\n%s", last)
	}
	m, _ = m.update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("g")})
	if got := ansi.Strip(m.renderFull()); got != first {
		t.Errorf("g should scroll back to the top:\n%s", got)
	}
}

// tdp D4: on a wide screen every description shows in full; the box is sized
// by the longest one.
func TestK6KeyRefWidthFitsLongest(t *testing.T) {
	m := newHelpPopup()
	m.setSize(200, 80)
	title, rows := panel1KeyRef()
	m.open(title, rows)
	out := ansi.Strip(m.renderFull())
	for _, r := range rows {
		if !strings.Contains(out, r.desc) {
			t.Errorf("description %q is cut on a wide screen", r.desc)
		}
	}
}

// tdp K6, M4: the panel key reference names the keys the panels take — gg to
// the top (a lone g only waits for the next key), and Shift-Tab, which focuses
// the previous panel and was written nowhere.
func TestK6PanelKeyRefMatchesTheKeys(t *testing.T) {
	m := f4Model(t)
	_, rows := m.panelKeyRef()
	keys := map[string]string{}
	for _, r := range rows {
		keys[r.key] = r.desc
	}
	if _, ok := keys["gg/G"]; !ok {
		t.Error("the panel key reference should list gg/G for top / bottom")
	}
	for k := range keys {
		if k == "g" || strings.HasPrefix(k, "g/") || strings.HasPrefix(k, "g ") {
			t.Errorf("a lone g does not move on a panel, yet the key reference lists %q", k)
		}
	}
	if keys["Shift-Tab"] != "focus the previous panel" {
		t.Fatalf("the panel key reference should list Shift-Tab, got %q", keys["Shift-Tab"])
	}
	m.focus = panelList
	if m = press(t, m, tea.KeyMsg{Type: tea.KeyShiftTab}); m.focus != panelMarks {
		t.Errorf("Shift-Tab from [1] should focus the previous panel, [3]; got %v", m.focus)
	}
}
