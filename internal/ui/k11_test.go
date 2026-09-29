package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

// yankModel is minModel with the yank viewport open on a few lines, selecting
// when select is true.
func yankModel(t *testing.T, selecting bool) AppModel {
	t.Helper()
	m := k9Model()
	m.help, m.detailYank = newHelpPopup(), newDetailYank()
	m.width, m.height = 100, 40
	m.detailYank.setSize(100, 40)
	m.detailYank.open("notes.txt", []string{"line one", "line two", "line three"}, false, nil)
	m.detailYank.anim.state = popupOpen
	if selecting {
		m = press(t, m, runes("v"))
		if !m.detailYank.visual {
			t.Fatal("setup: v should start selecting")
		}
	}
	return m
}

// tdp K11 (v0.1.10) / K5: a mode has no Space menu and no key list — Space
// while selecting does nothing: no popup opens and the selection stays as it
// is. Outside the mode the viewport is a plain popup and Space does nothing
// either.
func TestK11SpaceDoesNothingInTheMode(t *testing.T) {
	for _, selecting := range []bool{true, false} {
		m := yankModel(t, selecting)
		m = press(t, m, runes("l"))
		line, col := m.detailYank.cursorLine, m.detailYank.cursorCol
		m = press(t, m, runes(" "))
		for _, a := range m.stackOrder() {
			if a != &m.detailYank.anim && a.owns() {
				t.Errorf("selecting=%v: Space opened a popup", selecting)
			}
		}
		if !m.detailYank.owns() || m.detailYank.visual != selecting ||
			m.detailYank.cursorLine != line || m.detailYank.cursorCol != col {
			t.Errorf("selecting=%v: Space should leave the viewport as it was", selecting)
		}
	}
}

// tdp K11 / K6: ? while selecting is the mode's help; outside it is the
// viewport's key reference. Both come from the same table.
func TestK11QuestionIsModeHelp(t *testing.T) {
	m := press(t, yankModel(t, true), qmKey)
	if m.help.title != "Selection keys" {
		t.Fatalf("? while selecting should open the mode help, got %q", m.help.title)
	}
	have := map[string]bool{}
	for _, r := range m.help.rows {
		have[r.key] = true
	}
	for _, k := range selectKeys {
		if !have[k.keys] {
			t.Errorf("the mode help is missing %q", k.keys)
		}
	}
	if have["Space"] {
		t.Error("Space does nothing in the mode; its help should not list it (tdp K11)")
	}
	o := press(t, yankModel(t, false), qmKey)
	if o.help.title != "Preview viewport keys" {
		t.Errorf("? outside the selection should be the viewport keys, got %q", o.help.title)
	}
}

// tdp K11: Tab is suspended in the mode but answers.
func TestK11TabAnswersWhileSelecting(t *testing.T) {
	m := press(t, yankModel(t, true), tea.KeyMsg{Type: tea.KeyTab})
	if !m.toast.owns() || !strings.Contains(m.toast.message, "[Esc] leaves the selection") {
		t.Errorf("Tab while selecting should say how to leave: toast %v %q", m.toast.owns(), m.toast.message)
	}
	if !m.detailYank.visual {
		t.Error("Tab should not leave the selection")
	}
	// With that toast up, the first Esc takes the toast (K4, F3); the next one
	// leaves the selection.
	if m = press(t, m, tea.KeyMsg{Type: tea.KeyEsc}); m.toast.owns() || !m.detailYank.visual {
		t.Fatalf("the first Esc should close only the toast: toast %v, selecting %v", m.toast.owns(), m.detailYank.visual)
	}
	if m = press(t, m, tea.KeyMsg{Type: tea.KeyEsc}); m.detailYank.visual {
		t.Error("the second Esc should leave the selection")
	}
}

// The hint follows the state; the box width does not (tdp L2).
func TestK11HintFollowsStateWidthHolds(t *testing.T) {
	m := yankModel(t, false)
	out := ansi.Strip(m.detailYank.renderFull())
	w := ansi.StringWidth(strings.Split(out, "\n")[0])
	if !strings.Contains(out, "v:select") {
		t.Errorf("outside the selection the hint should offer v:\n%s", out)
	}
	m = press(t, m, runes("v"))
	sel := ansi.Strip(m.detailYank.renderFull())
	if !strings.Contains(sel, "?:keys") || !strings.Contains(sel, "Esc:leave") || strings.Contains(sel, "Space") {
		t.Errorf("while selecting the hint should offer ? and Esc, not Space:\n%s", sel)
	}
	if got := ansi.StringWidth(strings.Split(sel, "\n")[0]); got != w {
		t.Errorf("the box width changed with the state: %d → %d", w, got)
	}
}

// tdp M3 / K11: every key detailYank.update handles while selecting has a row
// in the table — a key the viewport answers to but the mode's key reference
// leaves out would be one only a memorised hotkey reaches. Keep this list in
// step with update.
func TestK11TableCoversViewportKeys(t *testing.T) {
	triggers := map[string]bool{}
	for _, k := range selectKeys {
		triggers[k.trigger()] = true
	}
	for _, key := range []string{"h", "l", "j", "k", "0", "$", "g", "G", "u", "d", "y", "v"} {
		if !triggers[key] {
			t.Errorf("the viewport answers to %q while selecting, but the mode's key reference has no row for it", key)
		}
	}
}

// tdp K11, D2, D3: a mode labels itself — its name between two junctions at the
// top right of the box it is in, the frame in the mode colour (Yellow) — and
// leaving it puts both back. The box keeps its width either way (L2), and a
// narrow one keeps the name.
func TestK11ModeLabelsItself(t *testing.T) {
	truecolor(t)
	yellow := [3]int{0xf9, 0xe2, 0xaf}
	layer1 := [3]int{0xa4, 0xc0, 0xfa} // Lavenphire25
	m := yankModel(t, true)
	box := strings.Split(m.detailYank.renderFull(), "\n")
	top := box[0]
	if !strings.HasSuffix(ansi.Strip(top), "┤Selection├─╮") {
		t.Errorf("selecting, the top border should end with the mode name: %q", ansi.Strip(top))
	}
	fg := cellFG(top)
	for _, at := range []int{0, len(fg) - 1, cellAt(t, top, "Selection"), cellAt(t, top, "notes")} {
		if !near(fg[at], yellow) {
			t.Errorf("selecting, top border cell %d is %v, want Yellow %v", at, fg[at], yellow)
		}
	}
	if got := cellFG(box[1])[0]; !near(got, yellow) {
		t.Errorf("selecting, the side border is %v, want Yellow", got)
	}
	width := dispWidth(top)

	m = press(t, m, tea.KeyMsg{Type: tea.KeyEsc})
	if m.detailYank.visual {
		t.Fatal("Esc should leave the selection")
	}
	top = strings.Split(m.detailYank.renderFull(), "\n")[0]
	if strings.Contains(ansi.Strip(top), "Selection") {
		t.Errorf("out of the mode, no mode name: %q", ansi.Strip(top))
	}
	if got := cellFG(top)[0]; !near(got, layer1) {
		t.Errorf("out of the mode, the frame is %v, want its layer colour %v", got, layer1)
	}
	if got := dispWidth(top); got != width {
		t.Errorf("the box is %d wide out of the mode, %d in it", got, width)
	}

	m = press(t, m, runes("v"))
	m.detailYank.setSize(24, 40) // the narrowest box: 20 inside
	top = ansi.Strip(strings.Split(m.detailYank.renderFull(), "\n")[0])
	if !strings.HasSuffix(top, "┤Selection├─╮") || dispWidth(top) != 22 {
		t.Errorf("a narrow box should cut the title and keep the mode name: %q", top)
	}
}

// tdp D2: the selection is Yellow, the colour of the mode frame around it
// (user ruling 2026-09-29: Lavender clashed with the Yellow frame; Lavender
// stays the user's footprint).
func TestK11SelectionIsYellow(t *testing.T) {
	truecolor(t)
	m := press(t, yankModel(t, true), runes("l")) // two characters selected
	row := strings.Split(m.detailYank.renderFull(), "\n")[1]
	if bg, _, ok := firstBG(row); !ok || !near(bg, [3]int{0xf9, 0xe2, 0xaf}) {
		t.Errorf("the selection is %v, want Yellow #f9e2af: %q", bg, row)
	}
}

// sgrBefore is the parameters of the last SGR sequence before sub in line.
func sgrBefore(t *testing.T, line, sub string) []string {
	t.Helper()
	i := strings.Index(line, sub)
	if i < 0 {
		t.Fatalf("%q not in %q", sub, line)
	}
	ps := sgrParams(line[:i])
	if len(ps) == 0 {
		return nil
	}
	return ps[len(ps)-1]
}

func hasParam(ps []string, p string) bool {
	for _, x := range ps {
		if x == p {
			return true
		}
	}
	return false
}

// tdp D3, D6: the mode tag's junctions are drawn like the frame (not bold), its
// name bold; with icons two cells wide and an icon in the title, the top border
// still measures the box.
func TestK11ModeTagJunctions(t *testing.T) {
	truecolor(t)
	defer restoreIconCells(iconCells)
	m := yankModel(t, true)
	top := strings.Split(m.detailYank.renderFull(), "\n")[0]
	if hasParam(sgrBefore(t, top, "┤"), "1") || hasParam(sgrBefore(t, top, "├"), "1") {
		t.Errorf("the junctions should be plain frame lines, not bold: %q", top)
	}
	if !hasParam(sgrBefore(t, top, "Selection"), "1") {
		t.Errorf("the mode name should be bold: %q", top)
	}
	for _, cells := range []int{1, 2} {
		iconCells = cells
		m.detailYank.title = "notes " + wideIcon + ".txt"
		lines := strings.Split(m.detailYank.renderFull(), "\n")
		if got, want := dispWidth(lines[0]), dispWidth(lines[1]); got != want {
			t.Errorf("icons %d: the top border is %d wide, the box %d: %q", cells, got, want, ansi.Strip(lines[0]))
		}
	}
}
