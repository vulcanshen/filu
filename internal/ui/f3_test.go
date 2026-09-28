package ui

import (
	"os"
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

var escKey = tea.KeyMsg{Type: tea.KeyEsc}

// f3Model is minModel with the active tab one level below a real directory, so
// an Esc that reaches the panel is visible as the tab moving up to the parent.
func f3Model(t *testing.T) (AppModel, string, string) {
	t.Helper()
	parent := t.TempDir()
	child := filepath.Join(parent, "child")
	if err := os.Mkdir(child, 0o755); err != nil {
		t.Fatal(err)
	}
	m := minModel()
	m.toast = newToast()
	m.tabs[0].dir = child
	return m, parent, child
}

// tdp F3: Esc closes a visible toast — even one that would close itself — and
// the key stops there instead of also sending the tab up a directory.
func TestF3EscClosesToastFirst(t *testing.T) {
	m, _, child := f3Model(t)
	m.toast.show("copied")
	m.toast.anim.state = popupOpen

	model, _ := m.Update(escKey)
	got := model.(AppModel)
	if got.toast.owns() {
		t.Error("Esc should start closing the toast")
	}
	if got.tabs[0].dir != child {
		t.Errorf("Esc on a toast leaked to the panel: tab moved to %q", got.tabs[0].dir)
	}
}

// tdp F3: only Esc belongs to the toast; other keys go on to the panel.
func TestF3ToastLetsOtherKeysThrough(t *testing.T) {
	m, _, _ := f3Model(t)
	m.toast.show("copied")
	m.toast.anim.state = popupOpen

	model, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	got := model.(AppModel)
	if !got.toast.owns() {
		t.Error("a non-Esc key should not close the toast")
	}
	if !got.quitMenu.isActive() {
		t.Error("q under a toast should still reach the panel and open the quit picker")
	}
}

// keyedPopups sets one key-routed popup to a given animation state. Each entry
// is one routing site in Update, so a regression at any of them shows up here.
var keyedPopups = []struct {
	name string
	set  func(m *AppModel, s popupAnimState)
}{
	{"detailYank", func(m *AppModel, s popupAnimState) { m.detailYank.anim.state = s }},
	{"meta", func(m *AppModel, s popupAnimState) { m.meta.anim.state = s }},
	{"search", func(m *AppModel, s popupAnimState) { m.search.anim.state = s }},
	{"breadcrumb", func(m *AppModel, s popupAnimState) { m.breadcrumb.anim.state = s }},
	{"help", func(m *AppModel, s popupAnimState) { m.help.anim.state = s }},
	{"inputPopup", func(m *AppModel, s popupAnimState) { m.inputPopup.anim.state = s }},
	{"confirm", func(m *AppModel, s popupAnimState) { m.confirm.anim.state = s }},
	{"spaceMenu", func(m *AppModel, s popupAnimState) { m.spaceMenu.anim.state = s }},
	{"globalMenu", func(m *AppModel, s popupAnimState) { m.globalMenu.anim.state = s }},
	{"sortMenu", func(m *AppModel, s popupAnimState) { m.sortMenu.anim.state = s }},
	{"gotoMenu", func(m *AppModel, s popupAnimState) { m.gotoMenu.anim.state = s }},
	{"openInMenu", func(m *AppModel, s popupAnimState) { m.openInMenu.anim.state = s }},
	{"searchMenu", func(m *AppModel, s popupAnimState) { m.searchMenu.anim.state = s }},
	{"quitMenu", func(m *AppModel, s popupAnimState) { m.quitMenu.anim.state = s }},
	{"openWithMenu", func(m *AppModel, s popupAnimState) { m.openWithMenu.anim.state = s }},
}

// tdp F3 / K4: a popup already running its close animation no longer takes
// keys, so the next Esc closes the layer beneath it (here: the panel, which
// goes up a directory) instead of being swallowed.
func TestF3ClosingPopupHandsEscDown(t *testing.T) {
	for _, tc := range keyedPopups {
		t.Run(tc.name, func(t *testing.T) {
			m, parent, _ := f3Model(t)
			tc.set(&m, popupClosingCompress)

			model, _ := m.Update(escKey)
			if got := model.(AppModel).tabs[0].dir; got != parent {
				t.Errorf("Esc while %s is closing was swallowed: tab at %q, want %q", tc.name, got, parent)
			}
		})
	}
}

// The other half of owns(): a popup still opening keeps the keyboard (it just
// does not act yet), so a key pressed mid-open never reaches the panel.
func TestF3OpeningPopupStillSwallowsKeys(t *testing.T) {
	for _, tc := range keyedPopups {
		t.Run(tc.name, func(t *testing.T) {
			m, _, child := f3Model(t)
			tc.set(&m, popupOpeningExpand)

			model, _ := m.Update(escKey)
			if got := model.(AppModel).tabs[0].dir; got != child {
				t.Errorf("Esc while %s is opening leaked to the panel: tab at %q", tc.name, got)
			}
		})
	}
}
