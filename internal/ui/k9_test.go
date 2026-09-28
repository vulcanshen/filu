package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	tea "github.com/charmbracelet/bubbletea"
)

var (
	qKey     = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")}
	ctrlCKey = tea.KeyMsg{Type: tea.KeyCtrlC}
)

// k9Model is minModel with every key-routed popup constructed, so any of them
// can be opened under the quit picker.
func k9Model() AppModel {
	m := minModel()
	m.toast = newToast()
	m.spaceMenu = newSpaceMenu()
	return m
}

// openUnder puts the named popup up, fully open, as the layer the leave flow
// has to stack on. The finder is put in its result list (not typing).
func openUnder(m *AppModel, name string) {
	for _, p := range keyedPopups {
		if p.name == name {
			p.set(m, popupOpen)
		}
	}
	if name == "search" {
		m.search.mode = searchNav
	}
}

// tdp K9: Ctrl-C on a panel enters the leave flow instead of quitting outright.
func TestK9CtrlCOnPanelOpensQuitPicker(t *testing.T) {
	model, cmd := k9Model().Update(ctrlCKey)
	if isQuitCmd(cmd) {
		t.Fatal("Ctrl-C on a panel should open the quit picker, not quit")
	}
	if !model.(AppModel).quitMenu.owns() {
		t.Error("Ctrl-C on a panel should open the quit picker")
	}
}

// tdp K9: Ctrl-C while the leave flow is already up leaves at once — even
// before the picker has finished opening.
func TestK9CtrlCOnQuitPickerQuits(t *testing.T) {
	for _, s := range []popupAnimState{popupOpeningExpand, popupOpen} {
		m := k9Model()
		m.openQuitMenu()
		m.quitMenu.anim.state = s
		if _, cmd := m.Update(ctrlCKey); !isQuitCmd(cmd) {
			t.Errorf("Ctrl-C on the quit picker (state %d) should quit", s)
		}
	}
}

// tdp K9 / D3: from any popup, q and Ctrl-C open the quit picker on top of it,
// leaving the popup underneath in place so Esc on the picker returns there.
func TestK9LeaveFlowStacksOnEveryPopup(t *testing.T) {
	for _, p := range keyedPopups {
		if p.name == "quitMenu" {
			continue
		}
		for _, key := range []tea.KeyMsg{qKey, ctrlCKey} {
			if key.String() == "q" && p.name == "inputPopup" {
				continue // typing: q is a character (TestK9QIsACharacterWhileTyping)
			}
			t.Run(p.name+"/"+key.String(), func(t *testing.T) {
				m := k9Model()
				openUnder(&m, p.name)
				model, cmd := m.Update(key)
				got := model.(AppModel)
				if isQuitCmd(cmd) {
					t.Fatal("should open the quit picker, not quit")
				}
				if !got.quitMenu.owns() {
					t.Fatalf("%s on %s should open the quit picker", key.String(), p.name)
				}
				if !stillOpen(got, p.name) {
					t.Errorf("the quit picker should stack on %s, not close it", p.name)
				}
			})
		}
	}
}

// stillOpen reports whether the named popup still holds its place in the stack.
func stillOpen(m AppModel, name string) bool {
	switch name {
	case "detailYank":
		return m.detailYank.owns()
	case "meta":
		return m.meta.owns()
	case "search":
		return m.search.owns()
	case "breadcrumb":
		return m.breadcrumb.owns()
	case "help":
		return m.help.owns()
	case "inputPopup":
		return m.inputPopup.owns()
	case "confirm":
		return m.confirm.owns()
	case "spaceMenu":
		return m.spaceMenu.owns()
	case "globalMenu":
		return m.globalMenu.owns()
	case "sortMenu":
		return m.sortMenu.owns()
	case "gotoMenu":
		return m.gotoMenu.owns()
	case "gotoFavMenu":
		return m.gotoFavMenu.owns()
	case "sortDirMenu":
		return m.sortDirMenu.owns()
	case "openInMenu":
		return m.openInMenu.owns()
	case "searchMenu":
		return m.searchMenu.owns()
	case "openWithMenu":
		return m.openWithMenu.owns()
	}
	return false
}

// tdp K8: while typing, q is a character — it lands in the text, and the leave
// flow stays shut. Ctrl-C still works there (covered above).
func TestK9QIsACharacterWhileTyping(t *testing.T) {
	m := k9Model()
	m.inputPopup = newInputPopup()
	m.inputPopup.open(inputRename, "Rename", "a", fileItem{name: "a"})
	m.inputPopup.anim.state = popupOpen
	model, _ := m.Update(qKey)
	got := model.(AppModel)
	if got.quitMenu.owns() {
		t.Error("q in the input popup should not open the quit picker")
	}
	if got.inputPopup.buffer != "aq" {
		t.Errorf("q should be typed into the input: buffer %q, want %q", got.inputPopup.buffer, "aq")
	}

	f := k9Model()
	f.search.anim.state = popupOpen
	f.search.mode = searchInput
	model, _ = f.Update(qKey)
	got = model.(AppModel)
	if got.quitMenu.owns() {
		t.Error("q on the finder's query line should not open the quit picker")
	}
	if got.search.query != "q" {
		t.Errorf("q should be typed into the query: %q", got.search.query)
	}
}

// tdp K9: q on the leave flow does not stack a second one (or replay the
// picker's open animation — the only visible trace of a re-open).
func TestK9QOnQuitPickerDoesNotReopen(t *testing.T) {
	m := k9Model()
	m.openQuitMenu()
	m.quitMenu.anim.state = popupOpen
	model, cmd := m.Update(qKey)
	got := model.(AppModel)
	if isQuitCmd(cmd) {
		t.Fatal("q on the quit picker should not quit")
	}
	if !got.quitMenu.isInteractive() {
		t.Error("q on the quit picker should leave it as it is, not replay its opening")
	}
}

// tdp D3 / F4: Esc on the quit picker closes only the picker; the popup it was
// opened over is back in charge.
func TestK9EscOnQuitPickerReturnsToPopup(t *testing.T) {
	m := k9Model()
	m.confirm.open("Delete a?", "trash")
	m.confirm.anim.state = popupOpen
	model, _ := m.Update(qKey)
	m = model.(AppModel)
	m.quitMenu.anim.state = popupOpen

	model, _ = m.Update(escKey)
	got := model.(AppModel)
	if got.quitMenu.owns() {
		t.Error("Esc should close the quit picker")
	}
	if !got.confirm.owns() {
		t.Error("Esc on the quit picker should return to the confirm beneath it")
	}
}

// tdp D3: the leave flow is drawn over every other popup.
func TestK9QuitPickerDrawnOnTop(t *testing.T) {
	m := k9Model()
	m.width, m.height = 100, 40
	m.confirm.setSize(m.width)
	m.confirm.open(strings.Repeat("delete everything ", 12), "trash")
	m.confirm.anim.state = popupOpen
	m.openQuitMenu()
	m.quitMenu.anim.state = popupOpen

	if out := ansi.Strip(m.View()); !strings.Contains(out, "Quit — cd to…") {
		t.Errorf("the quit picker should be drawn over the confirm:\n%s", out)
	}
}
