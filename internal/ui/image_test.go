package ui

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestIsImage(t *testing.T) {
	cases := map[string]bool{
		"photo.png": true, "a.JPG": true, "clip.gif": true, "scan.tiff": true,
		"doc.txt": false, "noext": false, "archive.zip": false, "vector.svg": false,
	}
	for name, want := range cases {
		if got := isImage(name); got != want {
			t.Errorf("isImage(%q) = %v, want %v", name, got, want)
		}
	}
}

func TestImageASCII(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "swatch.png")
	img := image.NewRGBA(image.Rect(0, 0, 16, 16))
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			img.Set(x, y, color.RGBA{uint8(x * 16), uint8(y * 16), 128, 255})
		}
	}
	f, err := os.Create(p)
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(f, img); err != nil {
		t.Fatal(err)
	}
	f.Close()

	lines, ok := imageASCII(p, 12)
	if !ok {
		t.Fatal("expected the PNG to render")
	}
	if len(lines) == 0 {
		t.Fatal("no ASCII lines produced")
	}
	joined := strings.Join(lines, "\n")
	if !strings.Contains(joined, "\x1b[") {
		t.Error("coloured output should carry ANSI codes")
	}
	for i, l := range lines {
		if !strings.HasSuffix(l, ansiReset) {
			t.Errorf("line %d not reset-terminated (colour could bleed)", i)
		}
	}
}

func TestImageASCIIMissing(t *testing.T) {
	if _, ok := imageASCII("/no/such/file.png", 20); ok {
		t.Error("a missing image must return ok=false")
	}
}
