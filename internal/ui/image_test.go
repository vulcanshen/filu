package ui

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func TestIsImage(t *testing.T) {
	cases := map[string]bool{
		"photo.png": true, "a.JPG": true, "clip.gif": true, "scan.tiff": true,
		"favicon.ico": true,
		"doc.txt":     false, "noext": false, "archive.zip": false,
		"vector.svg": false, // SVG is text — previews as XML, not a data URI
	}
	for name, want := range cases {
		if got := isImage(name); got != want {
			t.Errorf("isImage(%q) = %v, want %v", name, got, want)
		}
	}
}

func TestImageDataURI(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "pixel.png")
	want := []byte("hello-image-bytes-\x00\x01\x02")
	if err := os.WriteFile(p, want, 0o644); err != nil {
		t.Fatal(err)
	}

	lines, ok := imageDataURI(p, "pixel.png", 20)
	if !ok {
		t.Fatal("expected ok=true")
	}
	joined := ansi.Strip(strings.Join(lines, "")) // "" join drops the wrap newlines
	if !strings.Contains(joined, "data:image/png") {
		t.Errorf("missing mime prefix: %q", joined)
	}
	_, b64, found := strings.Cut(joined, ";base64,")
	if !found {
		t.Fatalf("no data URI produced: %q", joined)
	}
	got, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		t.Fatalf("body is not valid base64: %v", err)
	}
	if string(got) != string(want) {
		t.Errorf("roundtrip mismatch: %q != %q", got, want)
	}
}

func TestImageDataURITooLarge(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "big.png")
	if err := os.WriteFile(p, make([]byte, dataURIMaxBytes+1), 0o644); err != nil {
		t.Fatal(err)
	}
	lines, ok := imageDataURI(p, "big.png", 40)
	if !ok {
		t.Fatal("expected ok=true (with a note)")
	}
	if !strings.Contains(ansi.Strip(strings.Join(lines, "\n")), "too large") {
		t.Error("an oversized image should show a note, not a blob")
	}
}

func TestImageDataURIMissing(t *testing.T) {
	if _, ok := imageDataURI("/no/such/file.png", "file.png", 40); ok {
		t.Error("a missing image must return ok=false")
	}
}
