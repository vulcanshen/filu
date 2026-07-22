package ui

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func TestArchiveTreeZip(t *testing.T) {
	dir := t.TempDir()
	zp := filepath.Join(dir, "sample.zip")
	f, err := os.Create(zp)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(f)
	for _, name := range []string{"src/main.go", "README.md", "src/util/helper.go"} {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		_, _ = w.Write([]byte("x"))
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	f.Close()

	assertArchiveHas(t, zp, "sample.zip", "src", "main.go", "README.md", "util", "helper.go")
}

func TestArchiveTreeTarGz(t *testing.T) {
	dir := t.TempDir()
	tp := filepath.Join(dir, "sample.tar.gz")
	f, err := os.Create(tp)
	if err != nil {
		t.Fatal(err)
	}
	gz := gzip.NewWriter(f)
	tw := tar.NewWriter(gz)
	for _, name := range []string{"app/config.yaml", "app/bin/run"} {
		body := []byte("data")
		if err := tw.WriteHeader(&tar.Header{Name: name, Mode: 0o644, Size: int64(len(body))}); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write(body); err != nil {
			t.Fatal(err)
		}
	}
	tw.Close()
	gz.Close()
	f.Close()

	assertArchiveHas(t, tp, "sample.tar.gz", "app", "config.yaml", "bin", "run")
}

func TestArchiveTreeNotArchive(t *testing.T) {
	if _, ok := archiveTree("/nope/whatever.txt", "whatever.txt"); ok {
		t.Error("a .txt should not be treated as an archive")
	}
}

func assertArchiveHas(t *testing.T, path, name string, wants ...string) {
	t.Helper()
	lines, ok := archiveTree(path, name)
	if !ok {
		t.Fatalf("%s: expected a recognised archive", name)
	}
	joined := ansi.Strip(strings.Join(lines, "\n"))
	for _, want := range wants {
		if !strings.Contains(joined, want) {
			t.Errorf("%s tree missing %q:\n%s", name, want, joined)
		}
	}
}
