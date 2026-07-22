package ui

import (
	"strings"
	"testing"
)

func TestHighlightKnown(t *testing.T) {
	cases := []struct{ name, src string }{
		{"config.json", `{"a": 1, "b": [true, null]}`},
		{"app.yaml", "key: value\nlist:\n  - one\n"},
		{"main.go", "package main\n\nfunc main() {}\n"},
	}
	for _, tc := range cases {
		lines, ok := highlight(tc.name, tc.src)
		if !ok {
			t.Errorf("%s: expected a lexer match", tc.name)
			continue
		}
		joined := strings.Join(lines, "\n")
		if !strings.Contains(joined, "\x1b[") {
			t.Errorf("%s: expected ANSI colour codes, got %q", tc.name, joined)
		}
		if strings.Contains(joined, "48;2;") { // truecolor background — must be stripped
			t.Errorf("%s: output carries a background colour, want foreground-only", tc.name)
		}
		for i, l := range lines {
			if !strings.HasSuffix(l, ansiReset) {
				t.Errorf("%s line %d not reset-terminated: %q", tc.name, i, l)
			}
		}
	}
}

func TestHighlightUnknown(t *testing.T) {
	if _, ok := highlight("mystery", "just some words"); ok {
		t.Error("a name with no recognised type should fall back to plain text")
	}
}
