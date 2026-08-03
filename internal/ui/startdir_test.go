package ui

import (
	"os"
	"path/filepath"
	"testing"
)

// TestResolveStartDir: a directory arg opens as-is (no focus); a file arg opens
// its parent and focuses the file; a missing path errors.
func TestResolveStartDir(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "file.txt")
	mustWrite(t, file)

	t.Run("dir", func(t *testing.T) {
		gotDir, gotFocus, err := ResolveStartDir(dir)
		if err != nil {
			t.Fatal(err)
		}
		if gotDir != dir || gotFocus != "" {
			t.Errorf("dir arg -> (%q, %q), want (%q, \"\")", gotDir, gotFocus, dir)
		}
	})
	t.Run("file", func(t *testing.T) {
		gotDir, gotFocus, err := ResolveStartDir(file)
		if err != nil {
			t.Fatal(err)
		}
		if gotDir != dir || gotFocus != "file.txt" {
			t.Errorf("file arg -> (%q, %q), want (%q, \"file.txt\")", gotDir, gotFocus, dir)
		}
	})
	t.Run("missing", func(t *testing.T) {
		if _, _, err := ResolveStartDir(filepath.Join(dir, "nope")); err == nil {
			t.Error("a missing path should error")
		}
	})
}

// TestResolveStartDirRelative: a relative arg is made absolute against the CWD.
func TestResolveStartDirRelative(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	wd, _ := os.Getwd()
	defer os.Chdir(wd)
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	gotDir, gotFocus, err := ResolveStartDir("sub")
	if err != nil {
		t.Fatal(err)
	}
	if !filepath.IsAbs(gotDir) {
		t.Errorf("relative arg should resolve to an absolute dir, got %q", gotDir)
	}
	if gotFocus != "" {
		t.Errorf("a directory arg should have no focus, got %q", gotFocus)
	}
}

// TestFocusEntry: focusEntry lands the cursor on a listed entry (reporting true),
// and is a no-op reporting false for an entry that isn't listed.
func TestFocusEntry(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	mustWrite(t, filepath.Join(dir, "file.txt"))

	l := newList(dir)
	if !l.focusEntry("file.txt") {
		t.Fatal("focusEntry should find and report file.txt")
	}
	if got := l.cursorItem().name; got != "file.txt" {
		t.Fatalf("focusEntry should land on file.txt, cursor on %q", got)
	}
	before := l.cursor
	if l.focusEntry("ghost.txt") { // not listed -> false, no-op
		t.Error("focusEntry on a missing entry should report false")
	}
	if l.cursor != before {
		t.Errorf("focusEntry on a missing entry should be a no-op; cursor moved to %d", l.cursor)
	}
}

// TestFocusEntryRevealsHidden: a dotfile is hidden by default (focusEntry misses),
// but once hidden entries are shown it's found — the retry path New() uses for
// `filu ~/.dotfile`.
func TestFocusEntryRevealsHidden(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, ".env"))

	l := newList(dir)
	if l.focusEntry(".env") {
		t.Fatal("a dotfile should be hidden by default -> focusEntry misses")
	}
	l.showHidden = true
	l.reload()
	if !l.focusEntry(".env") {
		t.Fatal("with hidden shown, .env should be found")
	}
	if got := l.cursorItem().name; got != ".env" {
		t.Errorf("cursor should land on .env, got %q", got)
	}
}
