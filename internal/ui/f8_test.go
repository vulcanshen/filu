package ui

import (
	"strconv"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/muesli/termenv"
)

var textRGB = [3]int{0xcd, 0xd6, 0xf4} // the default foreground, as dimANSI takes it

// cellFG is the foreground colour of every cell of a styled line, as a
// truecolor terminal would draw it (unset = the default foreground).
func cellFG(line string) [][3]int {
	var out [][3]int
	fg := textRGB
	for i := 0; i < len(line); {
		if strings.HasPrefix(line[i:], "\x1b[") {
			end := i + 2
			for end < len(line) && (line[end] < 0x40 || line[end] > 0x7e) {
				end++
			}
			if end < len(line) && line[end] == 'm' {
				ps := strings.Split(line[i+2:end], ";")
				for k := 0; k < len(ps); k++ {
					switch ps[k] {
					case "", "0", "39":
						fg = textRGB
					case "38":
						if k+4 < len(ps) && ps[k+1] == "2" {
							for c := 0; c < 3; c++ {
								fg[c], _ = strconv.Atoi(ps[k+2+c])
							}
							k += 4
						}
					case "48":
						if k+1 < len(ps) && ps[k+1] == "2" {
							k += 4
						} else if k+1 < len(ps) && ps[k+1] == "5" {
							k += 2
						}
					}
				}
			}
			i = end + 1
			continue
		}
		_, size := decodeRune(line[i:])
		for w := max(ansi.StringWidth(line[i:i+size]), 0); w > 0; w-- {
			out = append(out, fg)
		}
		i += size
	}
	return out
}

func decodeRune(s string) (rune, int) {
	for i, r := range s {
		_ = i
		return r, len(string(r))
	}
	return 0, 1
}

// truecolor draws styles as 24-bit SGR for the test, as a real terminal gets.
func truecolor(t *testing.T) {
	t.Helper()
	old := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.TrueColor)
	t.Cleanup(func() { lipgloss.SetColorProfile(old) })
}

func hexRGB(c lipgloss.Color) [3]int {
	s := strings.TrimPrefix(string(c), "#")
	var out [3]int
	for k := 0; k < 3; k++ {
		v, _ := strconv.ParseUint(s[2*k:2*k+2], 16, 8)
		out[k] = int(v)
	}
	return out
}

// near compares colours allowing the rounding termenv does from hex to RGB.
func near(a, b [3]int) bool {
	for k := range a {
		if d := a[k] - b[k]; d > 2 || d < -2 {
			return false
		}
	}
	return true
}

// f8Model is a sized app with a file list to draw under the popups.
func f8Model(t *testing.T) AppModel {
	t.Helper()
	truecolor(t)
	m := f4Model(t)
	m.globalMenu, m.help, m.quitHelp = newGlobalMenu(), newHelpPopup(), newQuitHelp()
	model, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	return model.(AppModel)
}

// centred is where the overlay puts a box of size n in a span of N: half the
// span less half the box, each halved on its own (bubbletea-overlay Center).
func centred(N, n int) int { return N/2 - n/2 }

// outsideBox reports a cell outside the centred box of size bw×bh on a W×H screen.
func outsideBox(x, y, W, H, bw, bh int) bool {
	x0, y0 := centred(W, bw), centred(H, bh)
	return x < x0 || x >= x0+bw || y < y0 || y >= y0+bh
}

// tdp F8: with a popup open, everything outside it is the same screen dimmed —
// each cell's colour faded, none left at full strength.
func TestF8EverythingBelowTheTopIsDimmed(t *testing.T) {
	m := f8Model(t)
	plain := strings.Split(m.View(), "\n")

	m = press(t, m, runes(" "))
	box := m.spaceMenu.renderFull()
	bw, bh := boxWidth(box), boxRows(box)
	dimmed := strings.Split(m.View(), "\n")

	checked := 0
	for y := range plain {
		before, after := cellFG(plain[y]), cellFG(dimmed[y])
		for x := 0; x < len(before) && x < len(after); x++ {
			if !outsideBox(x, y, m.width, m.height, bw, bh) {
				continue
			}
			if want := dimRGB(before[x]); after[x] != want {
				t.Fatalf("cell (%d,%d) is %v under the Space menu, want the dimmed %v (was %v)", x, y, after[x], want, before[x])
			}
			checked++
		}
	}
	if checked < m.width*(m.height-bh) {
		t.Errorf("only %d cells compared", checked)
	}
}

// tdp F8: the popup on top stays bright; a popup under it dims too, its border
// kept in a dimmed version of its own layer colour (D2).
func TestF8LowerPopupBorderKeepsItsLayerColour(t *testing.T) {
	m := f8Model(t)
	m = press(t, m, runes(" "))
	spaceBox := m.spaceMenu.renderFull()
	m.spaceMenu.cursor = m.spaceMenu.lastSelectable() // Global operation
	m = press(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if !m.globalMenu.owns() {
		t.Fatal("the last row should open the global operation popup")
	}
	globalBox := m.globalMenu.renderFull()
	lines := strings.Split(m.View(), "\n")

	// The Space menu is taller, so its top border shows above the global popup.
	spaceTop := centred(m.height, boxRows(spaceBox))
	globalTop := centred(m.height, boxRows(globalBox))
	if spaceTop >= globalTop {
		t.Fatalf("the Space menu (row %d) should show above the global popup (row %d)", spaceTop, globalTop)
	}
	x0 := centred(m.width, boxWidth(spaceBox))
	// Lavenphire25 (#A4C0FA, layer 1) faded 55% into the base #1e1e2e: written
	// out, not computed with dimRGB, so a dim that loses the layer colour fails.
	if got, want := cellFG(lines[spaceTop])[x0], [3]int{90, 103, 138}; !near(got, want) {
		t.Errorf("the Space menu border beneath is %v, want its layer colour dimmed %v", got, want)
	}
	if got, want := cellFG(lines[globalTop])[x0], hexRGB(popupLayerColor(2)); !near(got, want) {
		t.Errorf("the global popup on top is %v, want its layer colour at full strength %v", got, want)
	}
}

// tdp F8: a toast holds no keys and is not a layer, so it dims nothing.
func TestF8ToastDoesNotDim(t *testing.T) {
	m := f8Model(t)
	plain := m.View()
	m.toast.show("x")
	m.toast.anim.state = popupOpen
	lines, before := strings.Split(m.View(), "\n"), strings.Split(plain, "\n")
	for y := 0; y < m.height/2; y++ { // the top half: well clear of the toast at the bottom
		if lines[y] != before[y] {
			t.Fatalf("row %d changed with only a toast up:\n%q\n%q", y, before[y], lines[y])
		}
	}
}

func TestDimSGR(t *testing.T) {
	c := dimRGB([3]int{200, 100, 50})
	for _, tc := range []struct{ in, want string }{
		{"38;2;200;100;50", sgrRGB(true, c)},
		{"1;38;2;200;100;50", "1;" + sgrRGB(true, c)},
		{"48;2;200;100;50", sgrRGB(false, c)},
		{"0", "0;" + sgrRGB(true, dimText)},
		{"", "0;" + sgrRGB(true, dimText)},
		{"39", sgrRGB(true, dimText)},
		{"0;38;2;200;100;50", "0;" + sgrRGB(true, c)},
		{"38;5;196", sgrRGB(true, dimRGB(xterm256(196)))},
		{"31", sgrRGB(true, dimRGB(ansi16[1]))},
		{"7", "7"},
	} {
		if got := dimSGR(tc.in); got != tc.want {
			t.Errorf("dimSGR(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// tdp D2: dimming never lightens. Each channel keeps the smaller of the original
// and the faded value, so a colour darker than the base (#1e1e2e) stays as it
// is, and one partly darker keeps those channels. Expected values written out
// from c × 0.45 + base × 0.55.
func TestD2DimNeverLightens(t *testing.T) {
	for _, tc := range []struct{ in, want [3]int }{
		{[3]int{0, 0, 0}, [3]int{0, 0, 0}},               // black: fading would lift it to (17,17,25)
		{[3]int{0x11, 0x11, 0x1b}, [3]int{17, 17, 27}},   // crust, darker than the base: unchanged
		{[3]int{200, 10, 50}, [3]int{107, 10, 48}},       // mixed: the dark green channel stays
		{[3]int{0xa4, 0xc0, 0xfa}, [3]int{90, 103, 138}}, // Lavenphire25: faded as before
	} {
		if got := dimRGB(tc.in); got != tc.want {
			t.Errorf("dimRGB(%v) = %v, want %v", tc.in, got, tc.want)
		}
	}
}
