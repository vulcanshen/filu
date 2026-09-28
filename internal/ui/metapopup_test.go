package ui

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

func factsMap(rows []metaRow) map[string]string {
	out := map[string]string{}
	for _, r := range rows {
		out[r.label] = r.value
	}
	return out
}

// The information box shows every fact the decision asked for.
func TestMetaFactsForAFile(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "notes.txt")
	if err := os.WriteFile(p, []byte(strings.Repeat("x", 12601)), 0o644); err != nil {
		t.Fatal(err)
	}
	f := factsMap(fileFacts(p))
	abs, _ := filepath.Abs(p)
	third := "Changed"
	if runtime.GOOS == "darwin" {
		third = "Created"
	}
	for label, want := range map[string]string{
		"Path":        abs,
		"Type":        "Text",
		"Size":        "12.3 KB (12,601 bytes)",
		"Permissions": "-rw-r--r-- (0644)",
	} {
		if f[label] != want {
			t.Errorf("%s = %q, want %q", label, f[label], want)
		}
	}
	for _, label := range []string{"Modified", "Accessed", third} {
		if len(f[label]) != len("2006-01-02 15:04:05") {
			t.Errorf("%s should be a full date and time, got %q", label, f[label])
		}
	}
	if !strings.Contains(f["Owner"], ":") {
		t.Errorf("Owner should be user:group, got %q", f["Owner"])
	}
}

// A symlink names where it points; one that can't be read says why.
func TestMetaFactsLinksAndErrors(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "real.txt")
	if err := os.WriteFile(target, []byte("hi"), 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "link")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	f := factsMap(fileFacts(link))
	if f["Link to"] != target || f["Type"] != "Symbolic link to a file" {
		t.Errorf("symlink facts: Link to %q, Type %q", f["Link to"], f["Type"])
	}
	broken := filepath.Join(dir, "broken")
	if err := os.Symlink(filepath.Join(dir, "nowhere"), broken); err != nil {
		t.Fatal(err)
	}
	if got := factsMap(fileFacts(broken))["Type"]; got != "Symbolic link (broken)" {
		t.Errorf("broken symlink Type = %q", got)
	}
	gone := factsMap(fileFacts(filepath.Join(dir, "gone")))
	if !strings.Contains(gone["Error"], "no such file or directory") {
		t.Errorf("an unreadable file should say why in the box, got %+v", gone)
	}
}

// Long values wrap instead of being cut: every character of the path shows,
// and the box stays inside the screen.
func TestMetaWrapsLongValues(t *testing.T) {
	deep := filepath.Join(t.TempDir(), strings.Repeat("a-very-long-directory-name/", 6))
	if err := os.MkdirAll(deep, 0o755); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(deep, "file.txt")
	if err := os.WriteFile(p, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	m := newMetaPopup()
	m.setSize(80, 60)
	m.open(p)
	out := ansi.Strip(m.renderFull())
	lines := strings.Split(out, "\n")
	for _, l := range lines {
		if ansi.StringWidth(l) > 80 {
			t.Fatalf("a line is wider than the screen:\n%s", out)
		}
	}
	abs, _ := filepath.Abs(p)
	var joined strings.Builder
	for _, l := range lines {
		joined.WriteString(strings.TrimRight(strings.Trim(l, "│"), " "))
	}
	if !strings.Contains(strings.ReplaceAll(joined.String(), " ", ""), strings.ReplaceAll(abs, " ", "")) {
		t.Errorf("the full path should show across the wrapped lines:\n%s", out)
	}
}

// The box scrolls when taller than the screen, and its width holds (tdp L2).
func TestMetaScrollsAndKeepsWidth(t *testing.T) {
	p := filepath.Join(t.TempDir(), "f.txt")
	if err := os.WriteFile(p, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	m := newMetaPopup()
	m.setSize(80, 10)
	m.open(p)
	m.anim.state = popupOpen
	first := ansi.Strip(m.renderFull())
	if n := len(strings.Split(first, "\n")); n > 10-2 {
		t.Fatalf("box is %d rows on a 10-row screen", n)
	}
	w := ansi.StringWidth(strings.Split(first, "\n")[0])
	m, _ = m.update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("G")})
	last := ansi.Strip(m.renderFull())
	if last == first || !strings.Contains(last, "Owner") {
		t.Errorf("G should scroll to the last facts:\n%s", last)
	}
	if got := ansi.StringWidth(strings.Split(last, "\n")[0]); got != w {
		t.Errorf("scrolling changed the width %d → %d", w, got)
	}
}

// tdp K3 / F1: Enter on a file opens its information box over the panel; it
// is read only — Space does nothing, ? lists its keys, Esc closes it.
func TestMetaFromEnter(t *testing.T) {
	m := f4Model(t)
	m.meta, m.help = newMetaPopup(), newHelpPopup()
	m = press(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if !m.meta.owns() || m.meta.title != "a.txt" {
		t.Fatalf("Enter on a.txt should open its information box: owns %v, title %q", m.meta.owns(), m.meta.title)
	}
	m = press(t, m, runes(" "))
	if !m.meta.owns() || m.spaceMenu.owns() {
		t.Error("Space on the information box should do nothing (tdp K5)")
	}
	m = press(t, m, qmKey)
	if m.help.title != "File information keys" {
		t.Errorf("? on the information box should list its keys, got %q", m.help.title)
	}
	m = press(t, m, escKey)
	m = press(t, m, escKey)
	if m.meta.owns() {
		t.Error("Esc should close the information box")
	}
}

// The box is wired like every other popup: sized, animated, drawn.
func TestMetaWiring(t *testing.T) {
	m := f4Model(t)
	m.meta = newMetaPopup()
	model, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	m = model.(AppModel)
	if m.meta.screenW != 100 || m.meta.screenH != 40 {
		t.Errorf("WindowSizeMsg should size the information box: %dx%d", m.meta.screenW, m.meta.screenH)
	}
	model, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = model.(AppModel)
	for i := 0; i < 20 && !m.meta.isInteractive(); i++ {
		model, _ = m.Update(AnimTickMsg{Target: "meta"})
		m = model.(AppModel)
	}
	if !m.meta.isInteractive() {
		t.Fatal("the information box never finished opening: AnimTickMsg is not reaching it")
	}
	if out := ansi.Strip(m.View()); !strings.Contains(out, "Permissions") {
		t.Errorf("View should draw the information box:\n%s", out)
	}
}
