package ui

import (
	"os"
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// A confirm opened from the Space menu sits over it: Esc goes back to the
// menu, not to the panel (tdp F4, T1).
func TestConfirmFromTheMenuKeepsTheMenu(t *testing.T) {
	hs := newHarness(t, 120, 40, map[string]string{"a.txt": ""})
	hs.to("a.txt")
	hs.keys("Space", "D")
	hs.has("Move a.txt to the trash?", "ITEM OPERATION")
	hs.keys("Esc")
	hs.has("ITEM OPERATION")
	hs.hasNot("Move a.txt to the trash?")
	if !hs.exists("a.txt") {
		t.Fatal("Esc deleted it")
	}
	hs.keys("D", "Enter") // accepting closes the whole stack
	hs.hasNot("ITEM OPERATION")
	if hs.exists("a.txt") {
		t.Fatal("Enter did not delete it")
	}
}

// The key reference lists Shift-Tab, which goes to the previous panel
// (tdp K2, K6).
func TestKeyReferenceListsShiftTab(t *testing.T) {
	hs := newHarness(t, 120, 40, nil)
	hs.keys("?")
	hs.has("Tab|Shift-Tab")
	hs.keys("Esc", "Shift-Tab")
	if hs.a.screen.Focused() != hs.a.drawer {
		t.Fatal("Shift-Tab on [1] did not go to [3]")
	}
}

// A hotkey goes in place in its label, in either case: [go]to, Mo[v]e here
// (tdp M5, S9).
func TestHotkeysInTheirLabels(t *testing.T) {
	hs := newHarness(t, 120, 40, map[string]string{"a.txt": ""})
	hs.to("a.txt")
	hs.keys("Space")
	hs.has("[go]to", "Mo[v]e here", "[o]pen", "New [t]ab", "[w] Close tab")
}

// From the Space menu, what writes a value back leaves the menu to come back
// to; a drill-down and a shell clear it (tdp T1).
func TestWhatTheMenuIsLeftWith(t *testing.T) {
	hs := newHarness(t, 120, 40, map[string]string{"a.txt": "", "sub/": ""})
	hs.to("a.txt")
	hs.keys("Space", "r", "b.txt", "Enter")
	if !hs.exists("b.txt") {
		t.Fatal("the rename did not happen")
	}
	hs.has("ITEM OPERATION")
	hs.keys("Esc")

	hs.keys("Space", "g", "o", "/")
	hs.drain()
	hs.keys("sub", "Enter", "Enter")
	if hs.a.cur().dir != hs.path("sub") {
		t.Fatalf("[1] is on %s", hs.a.cur().dir)
	}
	hs.hasNot("OPERATION", "GOTO") // sub is empty: its menu would have only the panel's section

	t.Setenv("SHELL", "/bin/sh")
	hs.keys("Space", "s", "Enter")
	if hs.a.term == nil {
		t.Fatal("no shell")
	}
	hs.send(tea.KeyMsg{Type: tea.KeyEsc, Alt: true}) // the terminal's exit key, which asks first
	hs.keys("Enter")
	hs.hasNot("OPERATION", "SHELL")
}

// The question names the item under the cursor as it is when the action runs.
func TestConfirmNamesTheItemNow(t *testing.T) {
	hs := newHarness(t, 120, 40, map[string]string{"a.txt": "", "b.txt": ""})
	hs.to("a.txt")
	hs.keys("D", "Esc")
	hs.to("b.txt")
	hs.keys("D")
	hs.has("Move b.txt to the trash?")
}

// A state.yaml that cannot be read is said, and not written over (tdp F5).
func TestBrokenStateIsSaidAndKept(t *testing.T) {
	hs := newHarness(t, 120, 40, map[string]string{"a.txt": ""})
	if err := os.MkdirAll(filepath.Dir(statePathOverride), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(statePathOverride, []byte("tabs: [::"), 0o644); err != nil {
		t.Fatal(err)
	}
	hs.a = NewApp(hs.root, "")
	hs.send(tea.WindowSizeMsg{Width: 120, Height: 40})
	hs.run(hs.a.start())
	hs.has("Cannot read the session")
	hs.to("a.txt")
	hs.keys("m")
	hs.a.shutdown()
	if b, _ := os.ReadFile(statePathOverride); string(b) != "tabs: [::" {
		t.Fatalf("state.yaml was written over: %q", b)
	}
}

// At / there is no directory above: the breadcrumb row is not there (tdp M6).
func TestNoBreadcrumbAtTheTop(t *testing.T) {
	hs := newHarness(t, 120, 40, nil)
	hs.a.cd("/", "")
	hs.keys("Space")
	hs.hasNot("[b]readcrumb")
	hs.keys("Esc")
	hs.a.cd(hs.root, "")
	hs.keys("Space")
	hs.has("[b]readcrumb")
}

// A file [2] cannot read is the panel's error: the fact and a hint naming a
// key (components/layout/panel).
func TestPreviewOfAnUnreadableFile(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("root reads everything")
	}
	hs := newHarness(t, 120, 40, map[string]string{"locked.txt": "x"})
	if err := os.Chmod(hs.path("locked.txt"), 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(hs.path("locked.txt"), 0o644) })
	hs.to("locked.txt")
	hs.has("Cannot read locked.txt", "Press [1], then [Enter]")
	if hs.a.preview.Fail == nil {
		t.Fatal("[2] has no Fail")
	}
}

// An empty directory's preview, and an empty [1], say what to do.
func TestEmptyStatesNameAKey(t *testing.T) {
	hs := newHarness(t, 120, 40, nil)
	hs.has("Nothing to preview", "Move the cursor in [1]")
	hs.keys("3", "l")
	hs.has("No tasks yet", "press [c] or [v]")
}

// With hide_secrets_in_preview in config.yaml a finder's preview keeps a
// private key out (components/input/file-picker).
func TestHideSecretsInPreview(t *testing.T) {
	hs := newHarness(t, 120, 40, map[string]string{
		"id_ed25519": "-----BEGIN OPENSSH PRIVATE KEY-----\nsecretsecret\n-----END OPENSSH PRIVATE KEY-----\n",
	})
	if err := os.MkdirAll(filepath.Dir(configPathOverride), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(configPathOverride, []byte("hide_secrets_in_preview: true\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	hs.a = NewApp(hs.root, "")
	hs.send(tea.WindowSizeMsg{Width: 120, Height: 40})
	hs.keys("/", "f")
	hs.drain()
	hs.keys("id_ed")
	hs.has("(sensitive content not shown)")
	hs.hasNot("secretsecret")
}

// The viewer gets the file as it is: a copy of all of it gives back the tabs
// and the control characters (components/dialog/note).
func TestViewerGetsTheFileAsItIs(t *testing.T) {
	text := "a\tb\nding\x07dong"
	hs := newHarness(t, 120, 40, map[string]string{"t.txt": text + "\n"})
	hs.to("t.txt")
	hs.keys("2", "Enter", "y")
	if got := hs.copied(); got != text {
		t.Fatalf("copied %q", got)
	}
}
