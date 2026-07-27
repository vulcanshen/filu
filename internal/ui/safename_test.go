package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSafeNameStripsControl(t *testing.T) {
	cases := map[string]string{
		"Icon\r":     "Icon",   // macOS custom-icon file — the reported bug
		"a\x1b[31mb": "a[31mb", // ESC stripped (no ANSI injection)
		"tab\ttab":   "tabtab",
		"normal.txt": "normal.txt",
	}
	for in, want := range cases {
		if got := safeName(in); got != want {
			t.Errorf("safeName(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestListViewNoCursorResetOnControlName(t *testing.T) {
	dir := t.TempDir()
	// a file literally named "Icon\r" (like macOS custom-icon files)
	if err := os.WriteFile(filepath.Join(dir, "Icon\r"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "z.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	out := newList(dir).view(40, 10, true, nil, nil)
	if strings.ContainsRune(out, '\r') {
		t.Error("list view must not emit a raw CR from a filename (would reset the cursor)")
	}
}
