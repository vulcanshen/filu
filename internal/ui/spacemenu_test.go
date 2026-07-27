package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

func TestBracketHotkey(t *testing.T) {
	cases := []struct{ label, key, want string }{
		{"Carry", "C", "[C]arry"},
		{"Copy here", "c", "[c]opy here"},   // case preserved (c vs C)
		{"Move here", "x", "[x] Move here"}, // key not in label → prefixed
		{"Hidden", ".", "[.] Hidden"},
		{"Delete", "D", "[D]elete"},
		{"Jump", "enter", "Jump"}, // multi-char key → plain
	}
	for _, c := range cases {
		if got := bracketHotkey(c.label, c.key); got != c.want {
			t.Errorf("bracketHotkey(%q, %q) = %q, want %q", c.label, c.key, got, c.want)
		}
	}
}

func TestSpaceMenuCommit(t *testing.T) {
	m := newSpaceMenu()
	m.setItems([]menuItem{{label: "Carry", key: "C"}, {label: "Rename", key: "R"}}, "x")
	m.anim.state = popupOpen // skip the open animation for the test

	if _, key, _ := m.update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("R")}); key != "R" {
		t.Errorf("direct hotkey R committed %q, want R", key)
	}
	moved, _, _ := m.update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	if _, key, _ := moved.update(tea.KeyMsg{Type: tea.KeyEnter}); key != "R" {
		t.Errorf("j+Enter committed %q, want R (2nd item)", key)
	}
	if _, key, cmd := m.update(tea.KeyMsg{Type: tea.KeyEsc}); key != "" || cmd == nil {
		t.Errorf("Esc should close without commit: key=%q cmd=%v", key, cmd)
	}
}

func TestSpaceMenuRender(t *testing.T) {
	m := newSpaceMenu()
	m.setSize(100)
	m.setItems([]menuItem{{label: "Carry", key: "C", hint: "add to bucket"}}, "README.md")
	plain := ansi.Strip(m.renderFull())
	for _, want := range []string{"README.md", "[C]arry", "add to bucket", "Space close"} {
		if !strings.Contains(plain, want) {
			t.Errorf("popup missing %q:\n%s", want, plain)
		}
	}
}

func TestQuitMenuSingleGlyphAlign(t *testing.T) {
	// The quit picker (hintRight) uses single-glyph hints (launch icon / tab
	// numeral), right-aligned. Rows with very different path widths must still
	// render to the same width, so the glyphs sit in one clean column on the right.
	m := newQuitMenu()
	m.setSize(120)
	m.setItems([]menuItem{
		{label: "~/Documents/sideproj/filu", key: "1", hint: iconCWD + " "},
		{label: "~/Downloads", key: "2", hint: tabNumeral(1) + " "},
		{label: "~/Documents/sideproj", key: "3", hint: tabNumeral(2) + " "},
	}, "Quit — cd to…")
	lines := strings.Split(ansi.Strip(m.renderFull()), "\n")

	want := ansi.StringWidth(lines[0])
	for i, ln := range lines {
		if got := ansi.StringWidth(ln); got != want {
			t.Errorf("row %d width = %d, want %d (box not rectangular — hints won't line up)\n%s",
				i, got, want, strings.Join(lines, "\n"))
		}
	}
}

func TestBuildSpaceMenuList(t *testing.T) {
	m := AppModel{focus: panelList}
	m.tabs = []listModel{{dir: "/tmp", items: []fileItem{{name: "foo.txt"}}}}
	items, title := m.buildSpaceMenu()
	if title != "foo.txt" {
		t.Errorf("title = %q, want foo.txt", title)
	}
	keys := map[string]bool{}
	headers := map[string]bool{}
	for _, it := range items {
		if it.header {
			headers[it.label] = true
			continue
		}
		keys[it.key] = true
	}
	for _, k := range []string{"p", "r", "a", "D", ".", "z"} {
		if !keys[k] {
			t.Errorf("panel [2] menu missing hotkey %q", k)
		}
	}
	if keys["P"] {
		t.Error("Pin should be hidden for a non-dir cursor item")
	}
	if !headers["item operation"] || !headers["panel operation"] {
		t.Errorf("panel [2] menu should label both regions: %v", headers)
	}
}

func TestGroupedMenu(t *testing.T) {
	itemOps := []menuItem{{label: "Carry", key: "C"}}
	panelOps := []menuItem{{label: "Zoom", key: "z"}}

	both := groupedMenu(itemOps, panelOps)
	if !both[0].header || both[0].label != "item operation" {
		t.Errorf("grouped menu should open with the item-operation header: %+v", both[0])
	}
	sawSep, sawPanelHdr := false, false
	for _, it := range both {
		sawSep = sawSep || it.separator
		sawPanelHdr = sawPanelHdr || (it.header && it.label == "panel operation")
	}
	if !sawSep || !sawPanelHdr {
		t.Error("two-region menu needs a separator and a panel-operation header")
	}

	flat := groupedMenu(nil, panelOps)
	for _, it := range flat {
		if it.header || it.separator {
			t.Errorf("single-region menu should stay flat: %+v", it)
		}
	}
}

func TestToggleZoom(t *testing.T) {
	var m AppModel
	m.toggleZoom(panelList)
	if m.zoom != panelList {
		t.Errorf("zoom = %v, want panelList", m.zoom)
	}
	m.toggleZoom(panelList) // same panel toggles off
	if m.zoom != 0 {
		t.Errorf("zoom = %v, want 0 after toggle off", m.zoom)
	}
	m.toggleZoom(panelDetail)
	m.toggleZoom(panelCarry) // different panel switches target
	if m.zoom != panelCarry {
		t.Errorf("zoom = %v, want panelCarry", m.zoom)
	}
}

func TestZoomFocusSwitch(t *testing.T) {
	// [2]-zoom stacks [2] over [4], so 2/4 switch focus without leaving zoom.
	m := AppModel{focus: panelList, zoom: panelList}
	if !m.zoomVisible(panelList) || !m.zoomVisible(panelCarry) {
		t.Error("[2]-zoom should show [2] and [4]")
	}
	if m.zoomVisible(panelPin) || m.zoomVisible(panelDetail) {
		t.Error("[2]-zoom should hide [1] and [3]")
	}
	m.setFocus(panelCarry)
	if m.zoom != panelList || m.focus != panelCarry {
		t.Errorf("4 in [2]-zoom: zoom=%v focus=%v, want zoom kept + focus [4]", m.zoom, m.focus)
	}
	m.setFocus(panelDetail)
	if m.zoom != 0 || m.focus != panelDetail {
		t.Errorf("3 in [2]-zoom: zoom=%v focus=%v, want zoom cleared + focus [3]", m.zoom, m.focus)
	}

	// [4]-zoom shows only [4]; switching away exits.
	m4 := AppModel{focus: panelCarry, zoom: panelCarry}
	if m4.zoomVisible(panelList) {
		t.Error("[4]-zoom should hide [2]")
	}
	m4.setFocus(panelList)
	if m4.zoom != 0 {
		t.Error("switching away from [4]-zoom should exit zoom")
	}
}
