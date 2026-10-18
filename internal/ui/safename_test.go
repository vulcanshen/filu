package ui

import (
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
