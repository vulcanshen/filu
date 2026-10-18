package ui

import (
	"bytes"
	"os"
	"testing"

	terminu "github.com/vulcanshen/terminu-go"
)

// The splash draws the same mark as docs/icon.svg, the original (tdp E6).
func TestIconMatchesDocs(t *testing.T) {
	docs, err := os.ReadFile("../../docs/icon.svg")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(docs, icon) {
		t.Fatal("internal/ui/icon.svg is not docs/icon.svg: copy it again")
	}
	if _, err := terminu.NewSplash(icon, "filu", "v0.0.0", splashIntro); err != nil {
		t.Fatal(err)
	}
}

func TestFirstScreen(t *testing.T) {
	hs := newHarness(t, 120, 40, map[string]string{"notes.md": "# notes\nhello\n", "src/": "", "src/main.go": "package main\n"})
	t.Log("\n" + hs.screen())
	hs.has("[1]", "[2] PREVIEW", "[3] MARKS", "notes.md", "src")
}
