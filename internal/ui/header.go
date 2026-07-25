package ui

import (
	"fmt"
	"math"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// gradientColor interpolates the ZLC lavender→sapphire z-axis scale at t∈[0,1]
// (plain sRGB component lerp). t=0 is Lavender (#b4befe, the user-footprint
// anchor) and t=1 is Sapphire (#74c7ec, the popup/z-axis ceiling); t of
// 0.25/0.50/0.75 land within ±1 LSB of the named Lavenphire25/50/75 stops that
// popupLayerColor samples discretely (kbu computes those in linear-RGB space; the
// difference is imperceptible). The header breadcrumb uses the continuous
// form so an N-deep path spreads evenly along the scale instead of clamping at
// four tiers — the concept the original ZLC import left out (see kbu
// theme.go: "intermediate stops are linear-RGB lerp of those two").
func gradientColor(t float64) lipgloss.Color {
	t = math.Min(1, math.Max(0, t))
	const (
		r0, g0, b0 = 0xb4, 0xbe, 0xfe // Lavender  #b4befe
		r1, g1, b1 = 0x74, 0xc7, 0xec // Sapphire  #74c7ec
	)
	lerp := func(a, b int) int { return int(math.Round(float64(a) + float64(b-a)*t)) }
	return lipgloss.Color(fmt.Sprintf("#%02x%02x%02x", lerp(r0, r1), lerp(g0, g1), lerp(b0, b1)))
}

// crumbColor is the breadcrumb colour for path segment i of n: the root (i=0)
// sits at Lavenphire25 (t=0.25, just past the Lavender folder chip) and the
// current directory (i=n-1) reaches Sapphire (t=1), the middle segments evenly
// interpolated between. A single-segment path (root == current, e.g. ~ or /) is
// the Sapphire endpoint.
func crumbColor(i, n int) lipgloss.Color {
	if n <= 1 {
		return gradientColor(1)
	}
	return gradientColor(0.25 + 0.75*float64(i)/float64(n-1))
}

// pathSegments splits a directory into breadcrumb segments, folding the home dir
// to ~ and representing the filesystem root as a single "/" segment. Control
// chars are stripped up front so segment widths measure what actually renders.
func pathSegments(dir string) []string {
	sp := safeName(shortPath(dir))
	if sp == "" || sp == "/" {
		return []string{"/"}
	}
	parts := strings.Split(sp, "/")
	segs := make([]string, 0, len(parts))
	for i, p := range parts {
		if p == "" {
			if i == 0 {
				segs = append(segs, "/") // leading empty = absolute root
			}
			continue // skip empties from a trailing / double slash
		}
		segs = append(segs, p)
	}
	if len(segs) == 0 {
		segs = append(segs, "/")
	}
	return segs
}

// headerBar renders the location breadcrumb as a powerline chip chain: a
// Lavender folder + active-tab-numeral chip, then one chip per path segment
// coloured along the ZLC gradient (root→current = Lavenphire25→Sapphire).
// Unlike a panel title it never dims — the header is always the live "you are
// here". When the bar overflows the width, front segments shrink to their
// initial (~/Documents/x → ~/D/x); if that is not enough the middle collapses to
// … keeping root + as many tail segments as fit, then a final hard clip.
func (m AppModel) headerBar(w int) string {
	glyph := string(rune(0xf07c)) // nf-fa-folder-open
	folderChip := m.folderChip(glyph + " " + tabNumeral(m.tab) + " ")
	segs := breadcrumbSegments(m.active().dir, w-dispWidth(folderChip))
	return padDisp(folderChip+renderCrumb(segs), w)
}

// folderChip is the leading "you are here" chip: a round-left cap + dark-on-
// Lavender body carrying the folder glyph and the active tab's Roman numeral.
func (m AppModel) folderChip(label string) string {
	c := gradientColor(0) // Lavender anchor
	cap := lipgloss.NewStyle().Foreground(c)
	body := lipgloss.NewStyle().Foreground(lipgloss.Color(baseHex)).Background(c).Bold(true)
	return cap.Render(capLeft) + body.Render(label)
}

// renderCrumb draws the path portion of the breadcrumb: a capHard triangle
// transition into each segment chip (coloured by crumbColor), then a capHard
// tail in the current segment's colour on the terminal background. The first
// transition blends from the folder chip's Lavender.
func renderCrumb(segs []string) string {
	if len(segs) == 0 {
		return ""
	}
	prev := gradientColor(0) // folder chip colour
	base := lipgloss.Color(baseHex)
	var b strings.Builder
	n := len(segs)
	for i, seg := range segs {
		c := crumbColor(i, n)
		// §8.2: the cap separates, so no leading space after it — only a trailing
		// space closes each segment before the next cap.
		b.WriteString(lipgloss.NewStyle().Foreground(prev).Background(c).Render(capHard))
		b.WriteString(lipgloss.NewStyle().Foreground(base).Background(c).Bold(true).Render(seg + " "))
		prev = c
	}
	b.WriteString(lipgloss.NewStyle().Foreground(prev).Render(capHard)) // tail arrow
	return b.String()
}

// breadcrumbSegments shrinks a directory's path segments to fit budget display
// cells: full names first, then front-to-back initial abbreviation, then a
// middle … collapse keeping root + tail, then the current segment alone (which
// padDisp clips if even that overflows).
func breadcrumbSegments(dir string, budget int) []string {
	segs := pathSegments(dir)
	fits := func(s []string) bool { return dispWidth(renderCrumb(s)) <= budget }
	if fits(segs) {
		return segs
	}
	// stage 2: abbreviate front segments to their initial, one at a time; never
	// the current (last) segment.
	for i := 0; i < len(segs)-1; i++ {
		segs[i] = firstRune(segs[i])
		if fits(segs) {
			return segs
		}
	}
	// stage 3: collapse the middle to …, keeping root + as many tail segments as
	// fit (grow the tail until one more segment overflows).
	var best []string
	for tailStart := len(segs) - 1; tailStart >= 1; tailStart-- {
		cand := append([]string{segs[0], "…"}, segs[tailStart:]...)
		if fits(cand) {
			best = cand
		} else {
			break
		}
	}
	if best != nil {
		return best
	}
	// stage 4: not even root + … + current fits — show just the current segment.
	return []string{segs[len(segs)-1]}
}
