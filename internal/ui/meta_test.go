package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func TestMetaLines(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "hello.txt"), []byte("hi"), 0o644); err != nil {
		t.Fatal(err)
	}
	joined := ansi.Strip(strings.Join(metaLines(fileItem{name: "hello.txt"}, dir), "\n"))
	for _, want := range []string{"Name", "hello.txt", "Type", "file", "Size", "2 bytes", "Owner", "Inode", "Modified"} {
		if !strings.Contains(joined, want) {
			t.Errorf("meta missing %q:\n%s", want, joined)
		}
	}
}

func TestMetaLinesDir(t *testing.T) {
	dir := t.TempDir()
	for _, n := range []string{"a", "b", "c"} {
		if err := os.WriteFile(filepath.Join(dir, n), nil, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	joined := ansi.Strip(strings.Join(metaLines(fileItem{name: filepath.Base(dir), isDir: true}, filepath.Dir(dir)), "\n"))
	if !strings.Contains(joined, "Type") || !strings.Contains(joined, "dir") {
		t.Errorf("dir meta should say Type dir:\n%s", joined)
	}
	if !strings.Contains(joined, "Items") || !strings.Contains(joined, "3") {
		t.Errorf("dir meta should count 3 items:\n%s", joined)
	}
}

func TestMetaLinesEmpty(t *testing.T) {
	if got := metaLines(fileItem{}, "/tmp"); len(got) != 1 || !strings.Contains(got[0], "no selection") {
		t.Errorf("empty selection: %v", got)
	}
}
