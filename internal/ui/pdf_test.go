package ui

import (
	"strings"
	"testing"
)

func TestPdfTextMissing(t *testing.T) {
	if _, _, ok := pdfText("/no/such/file.pdf"); ok {
		t.Error("a missing PDF must return ok=false, not succeed")
	}
}

func TestPdfLines(t *testing.T) {
	lines := pdfLines("hello\nworld", 3)
	if !strings.Contains(lines[0], "3 pages") {
		t.Errorf("header should show the page count: %q", lines[0])
	}
	joined := strings.Join(lines, "\n")
	if !strings.Contains(joined, "hello") || !strings.Contains(joined, "world") {
		t.Errorf("body text was lost: %q", joined)
	}
}

func TestPdfLinesEmpty(t *testing.T) {
	joined := strings.Join(pdfLines("  \n\n", 1), "\n")
	if !strings.Contains(joined, "1 page") {
		t.Errorf("want singular '1 page': %q", joined)
	}
	if !strings.Contains(joined, "no extractable text") {
		t.Errorf("an empty text layer should be noted: %q", joined)
	}
}
