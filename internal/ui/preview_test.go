package ui

import (
	"strconv"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
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

func TestWithLineNumbers(t *testing.T) {
	in := []string{"alpha", "\x1b[32mbeta\x1b[0m", "gamma"}
	out := withLineNumbers(in)
	if len(out) != len(in) {
		t.Fatalf("line count changed: %d -> %d", len(in), len(out))
	}
	for i, l := range out {
		if !strings.HasSuffix(l, in[i]) { // content preserved verbatim after the gutter
			t.Errorf("line %d lost its content: %q", i+1, l)
		}
		if fields := strings.Fields(ansi.Strip(l)); len(fields) == 0 || fields[0] != strconv.Itoa(i+1) {
			t.Errorf("line %d gutter should start with %d: %q", i+1, i+1, ansi.Strip(l))
		}
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
