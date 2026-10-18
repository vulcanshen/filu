package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	terminu "github.com/vulcanshen/terminu-go"
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
	hs.has("Cannot read state.yaml")
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

// A content search's preview keeps secrets out when hide_secrets_in_preview
// is on, and names control characters in Red (components/input/file-picker).
func TestHitPreview(t *testing.T) {
	hs := newHarness(t, 120, 40, map[string]string{
		"id_ed25519": "-----BEGIN OPENSSH PRIVATE KEY-----\nsecretsecret\n-----END OPENSSH PRIVATE KEY-----\n",
		"own.txt":    "mine\n",
		"bell.txt":   "ding\x07dong\n",
	})
	if err := os.Chmod(hs.path("own.txt"), 0o600); err != nil {
		t.Fatal(err)
	}
	hs.a.t.HideSecretsInPreview = true
	key := strings.Join(hs.a.hitPreview(hs.path("id_ed25519")), "\n")
	if strings.Contains(key, "secretsecret") || !strings.Contains(key, "(sensitive content not shown)") {
		t.Fatalf("the key's preview: %q", key)
	}
	if got := strings.Join(hs.a.hitPreview(hs.path("own.txt")), "\n"); strings.Contains(got, "mine") {
		t.Fatalf("a file only its owner reads: %q", got)
	}
	hs.a.t.HideSecretsInPreview = false
	if got := terminu.StripStyles(strings.Join(hs.a.hitPreview(hs.path("bell.txt")), "\n")); !strings.Contains(got, `ding\x07dong`) {
		t.Fatalf("a control character: %q", got)
	}
	if got := terminu.StripStyles(hs.a.hitPreview(hs.path("gone.txt"))[0]); got != "(unreadable)" {
		t.Fatalf("a file that is not there: %q", got)
	}
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
