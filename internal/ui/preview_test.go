package ui

import (
	"strconv"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

// A text's lines are as it has them: only a CRLF line's \r goes with the
// line break; tabs, control characters and colour codes are terminu's to draw.
func TestTextLines(t *testing.T) {
	got := textLines("a\tb\r\nc\x1b[31mred\nd\x00e\r10%\r100%")
	want := []string{"a\tb", "c\x1b[31mred", "d\x00e\r10%\r100%"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("textLines = %q, want %q", got, want)
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
