package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestSanitizeLine(t *testing.T) {
	cases := map[string]string{
		"a\tb":         "a    b",    // tab → 4 spaces
		"a\rb":         "ab",        // CR dropped
		"a\x1b[31mred": "a [31mred", // ESC → space (no raw escape leaks)
		"plain":        "plain",
		"nul\x00byte":  "nul byte",
	}
	for in, want := range cases {
		if got := sanitizeLine(in); got != want {
			t.Errorf("sanitizeLine(%q) = %q, want %q", in, got, want)
		}
	}
	// pathological long line is capped
	if got := sanitizeLine(strings.Repeat("x", 5000)); len(got) > 2000 {
		t.Errorf("long line not capped: len=%d", len(got))
	}
}

func TestTruncateWidth(t *testing.T) {
	// a wide-char line must clip to the display width, not the rune count
	s := strings.Repeat("你", 20) // 20 runes, 40 display cells
	got := truncate(s, 10)
	if w := lipgloss.Width(got); w > 10 {
		t.Errorf("truncate width = %d, want <= 10 (%q)", w, got)
	}
}
