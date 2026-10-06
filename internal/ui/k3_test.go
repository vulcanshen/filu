package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

// nameCheck is the input popup's validation: which names each kind refuses,
// and that it says why.
func TestK3NameCheck(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(dir, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		kind   inputKind
		target string
		name   string
		want   string // "" = accepted; else a substring of the reason
	}{
		{inputRename, "a.txt", "", "Type a name"},
		{inputRename, "a.txt", "x/y", "can't contain /"},
		{inputRename, "a.txt", "sub", "sub already exists"},
		{inputRename, "a.txt", "a.txt", ""}, // unchanged
		{inputRename, "a.txt", "b.txt", ""},
		{inputAdd, "", "", "Type a name"},
		{inputAdd, "", ".", "Not a name"},
		{inputAdd, "", "..", "Not a name"},
		{inputAdd, "", "a.txt", "a.txt already exists"},
		{inputAdd, "", "sub/", "sub already exists"},
		{inputAdd, "", "new/deep.txt", ""},
		{inputAdd, "", "../out.txt", "../out.txt is outside this directory"},
		{inputAdd, "", "/out/", "/out is outside this directory"},
		{inputAdd, "", "new/../../out", "outside this directory"},
		{inputAdd, "", "new/../../", "outside this directory"},
		{inputAdd, "", "new/../b.txt", ""}, // still in here once cleaned
		{inputZip, "", "", "Type a name"},
		{inputZip, "", "bundle", ""},
	} {
		got := nameCheck(tc.kind, dir, tc.target)(tc.name)
		if (tc.want == "") != (got == "") || !strings.Contains(got, tc.want) {
			t.Errorf("kind %d, %q → %q, want %q", tc.kind, tc.name, got, tc.want)
		}
	}
}

// Add creates under the current directory only: a value that climbs out with
// ../ or starts at / is refused on the spot and creates nothing, here or above;
// a ../ that stays inside still goes through.
func TestK3AddStaysInThisDirectory(t *testing.T) {
	m := f4Model(t)
	root := m.tabs[0].dir
	sub := filepath.Join(root, "sub")
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	m.tabs[0] = newList(sub)
	for _, v := range []string{"../out.txt", "/out.txt", "x/../../out.txt"} {
		m := press(t, openAdd(t, m), runes(v))
		m = press(t, m, enterKey)
		if !m.inputPopup.owns() || !strings.Contains(m.inputPopup.errMsg, "outside this directory") {
			t.Errorf("%q: owns %v, reason %q — want it refused", v, m.inputPopup.owns(), m.inputPopup.errMsg)
		}
		for _, p := range []string{filepath.Join(root, "out.txt"), filepath.Join(sub, "out.txt")} {
			if _, err := os.Lstat(p); err == nil {
				t.Errorf("%q created %s", v, p)
				_ = os.Remove(p)
			}
		}
	}
	m = press(t, openAdd(t, m), runes("x/../in.txt"))
	m = press(t, m, enterKey)
	if _, err := os.Lstat(filepath.Join(sub, "in.txt")); err != nil {
		t.Error("x/../in.txt stays in the directory and should be created there")
	}
}

// tdp K3: a failed submit keeps the popup and the focus on the field and says
// why — and nothing on disk changes. Fixing the name and pressing Enter again
// goes through.
func TestK3FailedSubmitStaysAndSaysWhy(t *testing.T) {
	m := f4Model(t)
	dir := m.tabs[0].dir
	if err := os.WriteFile(filepath.Join(dir, "b.txt"), []byte("keep me"), 0o644); err != nil {
		t.Fatal(err)
	}
	m.handleListKey("r") // rename a.txt
	m.inputPopup.anim.state = popupOpen
	m.inputPopup.buffer = "b.txt"

	m = press(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if !m.inputPopup.owns() {
		t.Fatal("renaming onto an existing name should not submit")
	}
	if !strings.Contains(ansi.Strip(m.inputPopup.renderFull()), "b.txt already exists here") {
		t.Errorf("the popup should say why:\n%s", ansi.Strip(m.inputPopup.renderFull()))
	}
	if data, _ := os.ReadFile(filepath.Join(dir, "b.txt")); string(data) != "keep me" {
		t.Error("a refused rename must not touch the existing file")
	}

	m = press(t, m, runes("x")) // typing clears the reason
	if m.inputPopup.errMsg != "" {
		t.Error("editing the name should clear the error")
	}
	m.inputPopup.buffer = "c.txt"
	m = press(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.inputPopup.owns() {
		t.Error("a valid name should submit and close the popup")
	}
	if _, err := os.Stat(filepath.Join(dir, "c.txt")); err != nil {
		t.Error("the valid rename should have happened")
	}
}

// tdp K3: an empty name is a failed check with a reason, not a silent close.
func TestK3EmptyNameIsRefused(t *testing.T) {
	m := f4Model(t)
	m.handleListKey("a")
	m.inputPopup.anim.state = popupOpen
	m = press(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if !m.inputPopup.owns() || m.inputPopup.errMsg == "" {
		t.Errorf("Enter on an empty name should stay open and say why: owns %v, err %q",
			m.inputPopup.owns(), m.inputPopup.errMsg)
	}
}

// tdp L2: the box width is set when the popup opens and holds while the value
// grows and an error line appears.
func TestL2InputWidthHolds(t *testing.T) {
	m := f4Model(t)
	m.width = 120
	m.inputPopup.setSize(120)
	m.handleListKey("a")
	m.inputPopup.anim.state = popupOpen
	width := func() int { return ansi.StringWidth(strings.Split(m.inputPopup.renderFull(), "\n")[0]) }
	w0 := width()

	m.inputPopup.buffer = strings.Repeat("long-name-", 20)
	if w := width(); w != w0 {
		t.Errorf("typing a long name changed the box width %d → %d", w0, w)
	}
	m.inputPopup.buffer = ""
	m = press(t, m, tea.KeyMsg{Type: tea.KeyEnter}) // empty → error line
	if w := width(); w != w0 {
		t.Errorf("the error line changed the box width %d → %d", w0, w)
	}
}
