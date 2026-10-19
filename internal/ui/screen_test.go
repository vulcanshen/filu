package ui

import (
	"fmt"
	"os"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	terminu "github.com/vulcanshen/terminu-go"
)

var fixture = map[string]string{
	"notes.md":     "# notes\n\nhello world\n\tindented\n",
	"src/main.go":  "package main\n\nfunc main() {}\n",
	"docs/":        "",
	"日本語のファイル.txt": "テキスト\n",
	".env":         "SECRET=1\n",
	"a-rather-long-file-name-that-will-not-fit-in-a-narrow-panel.go": "package x\n",
}

// popups opens each kind of popup filu has, from [1] with the cursor on a
// file: the Space menu over the global operation popup, the key reference, a
// menu over a menu, a finder, a viewer, a confirm, a text input, a toast, the
// quit confirm. Each one is a list of keys.
var popups = map[string][]string{
	"none":        nil,
	"global":      {"Space", "k", "Enter"},
	"keys":        {"?"},
	"sort column": {"S", "s"},
	"name finder": {"/", "f"},
	"info":        {"Enter"},
	"preview":     {"2", "Enter"},
	"delete":      {"D"},
	"rename":      {"r"},
	"toast":       {"y"},
	"quit":        {"q"},
	"error":       {"/", "c"},     // content search without ripgrep (openPopup)
	"terminal":    {"s", "Enter"}, // the shell, /bin/sh (openPopup)
}

// openPopup opens one of popups on hs: the error popup needs ripgrep missing,
// the terminal a shell that starts fast and is stopped after.
func openPopup(t *testing.T, hs *harness, name string) {
	t.Helper()
	switch name {
	case "error":
		old := lookPath
		lookPath = func(string) (string, error) { return "", os.ErrNotExist }
		t.Cleanup(func() { lookPath = old })
	case "terminal":
		t.Setenv("SHELL", "/bin/sh")
		t.Cleanup(func() {
			if hs.a.term != nil {
				hs.a.term.Stop()
			}
		})
	}
	hs.to("notes.md")
	hs.keys(popups[name]...)
}

// sizesFor are the screen sizes each pair of widths is drawn at: the sizes
// filu has to work at down to its least size, 60 × 24, then smaller, then the
// widest.
var sizesFor = map[[2]int][][2]int{
	{1, 1}: {{80, 40}, {120, 40}, {200, 50}, {80, 24}, {72, 24}, {60, 30}, {60, 24}, {40, 12}},
	{2, 1}: {{80, 40}, {120, 40}, {60, 24}},
	{2, 2}: {{80, 40}, {120, 40}, {60, 24}},
}

// Every row of the screen is exactly as wide as the terminal (tdp L4), at
// the sizes filu has to work at and narrower, with an icon one cell or two and
// a powerline symbol one or two (tdp E5), and with every kind of popup open.
func TestL4EveryRowIsTheScreenWidth(t *testing.T) {
	for iw, sizes := range sizesFor {
		for _, sz := range sizes {
			for name := range popups {
				t.Run(fmt.Sprintf("icon%d-pl%d/%dx%d/%s", iw[0], iw[1], sz[0], sz[1], name), func(t *testing.T) {
					hs := newHarness(t, sz[0], sz[1], fixture)
					terminu.SetIconWidth(iw[0], iw[1])
					openPopup(t, hs, name)
					checkWidths(t, hs, sz[0], sz[1])
				})
			}
		}
	}
}

// A popup opened on a larger screen stays put while the terminal shrinks under
// it — to the least size, wider than the screen and taller; then under it,
// down to 1 × 1 — and nothing breaks (tdp E5, L1).
func TestE5PopupLargerThanTheScreen(t *testing.T) {
	for _, iw := range [][2]int{{1, 1}, {2, 2}} {
		for name := range popups {
			for _, to := range [][2]int{{60, 24}, {30, 40}, {120, 8}, {80, 5}, {24, 6}, {1, 1}} {
				t.Run(fmt.Sprintf("icon%d-pl%d/%s/%dx%d", iw[0], iw[1], name, to[0], to[1]), func(t *testing.T) {
					hs := newHarness(t, 120, 40, fixture)
					terminu.SetIconWidth(iw[0], iw[1])
					openPopup(t, hs, name)
					hs.send(tea.WindowSizeMsg{Width: to[0], Height: to[1]})
					checkWidths(t, hs, to[0], to[1])
				})
			}
		}
	}
}

// Under its least size, 60 × 24, filu shows only its name and the size it
// needs; no key does anything but Ctrl-C twice, and as the terminal grows the
// screen is back as it was (tdp L1).
func TestL1UnderTheLeastSize(t *testing.T) {
	for _, sz := range [][2]int{{59, 24}, {60, 23}, {40, 12}, {80, 5}} {
		hs := newHarness(t, 120, 40, fixture)
		hs.to("notes.md")
		hs.keys("r") // a popup open as it shrinks
		hs.send(tea.WindowSizeMsg{Width: sz[0], Height: sz[1]})
		hs.has("filu")
		if sz[1] > 2 {
			hs.has("needs a terminal of at least 60×24")
		}
		hs.hasNot("RENAME")
		hs.keys("Esc", "q", "Ctrl-C")
		if hs.quit {
			t.Fatalf("%dx%d: q and one Ctrl-C quit", sz[0], sz[1])
		}
		hs.send(tea.WindowSizeMsg{Width: 120, Height: 40})
		hs.has("RENAME: notes.md")
		hs.send(tea.WindowSizeMsg{Width: sz[0], Height: sz[1]})
		hs.keys("Ctrl-C", "Ctrl-C")
		if !hs.quit {
			t.Fatalf("%dx%d: Ctrl-C twice did not quit", sz[0], sz[1])
		}
	}
}

// The zoomed list with every tab side by side fits too.
func TestL4ZoomedTabs(t *testing.T) {
	for _, sz := range [][2]int{{80, 40}, {200, 50}, {60, 24}} {
		hs := newHarness(t, sz[0], sz[1], fixture)
		hs.keys("t", "s", "t", "s", "z")
		checkWidths(t, hs, sz[0], sz[1])
	}
}

func checkWidths(t *testing.T, hs *harness, w, h int) {
	t.Helper()
	lines := hs.a.t.Frame()
	if len(lines) != h {
		t.Fatalf("%d rows on a screen %d high", len(lines), h)
	}
	for i, l := range lines {
		if got := terminu.Width(l); got != w {
			t.Fatalf("row %d is %d cells on a screen %d wide:\n%s", i, got, w, hs.screen())
		}
	}
}
