package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTreeLines(t *testing.T) {
	dir := t.TempDir()
	mustMkdir(t, filepath.Join(dir, "a", "b"))
	mustWrite(t, filepath.Join(dir, "a", "f.txt"))
	mustWrite(t, filepath.Join(dir, "top.txt"))

	lines := treeLines(dir, 3)
	joined := strings.Join(lines, "\n")

	for _, want := range []string{"a", "b", "f.txt", "top.txt"} {
		if !strings.Contains(joined, want) {
			t.Errorf("tree missing %q:\n%s", want, joined)
		}
	}
	// nested entries must be indented deeper than their parent
	var aIndent, bIndent int
	for _, l := range lines {
		switch {
		case strings.HasSuffix(l, " a"):
			aIndent = indentWidth(l)
		case strings.HasSuffix(l, " b"):
			bIndent = indentWidth(l)
		}
	}
	if bIndent <= aIndent {
		t.Errorf("nested 'b' (indent %d) not deeper than 'a' (indent %d)", bIndent, aIndent)
	}
}

func TestTreeMaxDepth(t *testing.T) {
	dir := t.TempDir()
	mustMkdir(t, filepath.Join(dir, "l1", "l2", "l3", "l4"))
	joined := strings.Join(treeLines(dir, 3), "\n")
	if !strings.Contains(joined, "l3") {
		t.Error("l3 should be within 3 levels")
	}
	if strings.Contains(joined, "l4") {
		t.Error("l4 is the 4th level and must be pruned")
	}
}

func indentWidth(line string) int {
	i := strings.IndexFunc(line, func(r rune) bool {
		return r != ' ' && r != '│' && r != '├' && r != '└' && r != '─'
	})
	if i < 0 {
		return len(line)
	}
	return i
}

func mustMkdir(t *testing.T, p string) {
	t.Helper()
	if err := os.MkdirAll(p, 0o755); err != nil {
		t.Fatal(err)
	}
}

func mustWrite(t *testing.T, p string) {
	t.Helper()
	if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
}
