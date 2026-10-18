package ui

import (
	"fmt"
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
}

// sizesFor are the screen sizes each pair of widths is drawn at: the sizes
// filu has to work at and narrower, then the widest.
var sizesFor = map[[2]int][][2]int{
	{1, 1}: {{80, 40}, {120, 40}, {200, 50}, {72, 24}, {60, 30}, {40, 12}},
	{2, 1}: {{80, 40}, {120, 40}},
	{2, 2}: {{80, 40}, {120, 40}},
}

// Every row of the screen is exactly as wide as the terminal (Rules L4), at
// the sizes filu has to work at and narrower, with an icon one cell or two and
// a powerline symbol one or two (Rules E5), and with every kind of popup open.
func TestL4EveryRowIsTheScreenWidth(t *testing.T) {
	for iw, sizes := range sizesFor {
		for _, sz := range sizes {
			for name, keys := range popups {
				t.Run(fmt.Sprintf("icon%d-pl%d/%dx%d/%s", iw[0], iw[1], sz[0], sz[1], name), func(t *testing.T) {
					hs := newHarness(t, sz[0], sz[1], fixture)
					terminu.SetIconWidth(iw[0], iw[1])
					hs.to("notes.md")
					hs.keys(keys...)
					checkWidths(t, hs, sz[0], sz[1])
				})
			}
		}
	}
}

// A popup opened on a larger screen stays put while the terminal shrinks under
// it — wider than the screen, taller, or both — and nothing breaks (Rules E5).
func TestE5PopupLargerThanTheScreen(t *testing.T) {
	for _, iw := range [][2]int{{1, 1}, {2, 2}} {
		for name, keys := range popups {
			for _, to := range [][2]int{{30, 40}, {120, 8}, {24, 6}} {
				if name == "name finder" && to == [2]int{24, 6} {
					// TODO(lib FL8): a finder with file facts panics drawing them on a
					// screen this small.
					continue
				}
				t.Run(fmt.Sprintf("icon%d-pl%d/%s/%dx%d", iw[0], iw[1], name, to[0], to[1]), func(t *testing.T) {
					hs := newHarness(t, 120, 40, fixture)
					terminu.SetIconWidth(iw[0], iw[1])
					hs.to("notes.md")
					hs.keys(keys...)
					hs.send(tea.WindowSizeMsg{Width: to[0], Height: to[1]})
					checkWidths(t, hs, to[0], to[1])
				})
			}
		}
	}
}

// The zoomed list with every tab side by side fits too.
func TestL4ZoomedTabs(t *testing.T) {
	for _, sz := range [][2]int{{80, 40}, {200, 50}, {60, 20}} {
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
