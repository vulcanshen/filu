package ui

import (
	"fmt"
	"math"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Header path gradient endpoints. Unlike the ZLC popup scale (a low-contrast
// lavender→sapphire span), the breadcrumb runs a WIDE blue→crust gradient: the
// capHard triangle between two segments is visible only when their colours
// differ — the triangle *is* the colour boundary — so a big luminance drop per
// step keeps every separator legible, even on a deep path where the per-segment
// step is small. Root = crust, current = blue (the structural focus colour). The
// interpolation is still continuous by depth (the concept kept from the ZLC
// scale); only the endpoints widened.
const (
	crumbFromR, crumbFromG, crumbFromB = 0x11, 0x11, 0x1b // crust #11111b (root)
	crumbToR, crumbToG, crumbToB       = 0x89, 0xb4, 0xfa // blue  #89b4fa (focusColor, current)
	crumbLightText                     = "#cdd6f4"        // catppuccin text — used on the dark end
)

// crumbRGB interpolates the crust→blue gradient at t∈[0,1] (sRGB component lerp).
func crumbRGB(t float64) (int, int, int) {
	t = math.Min(1, math.Max(0, t))
	lerp := func(a, b int) int { return int(math.Round(float64(a) + float64(b-a)*t)) }
	return lerp(crumbFromR, crumbToR), lerp(crumbFromG, crumbToG), lerp(crumbFromB, crumbToB)
}

// crumbColorAt is the segment background colour at gradient position t.
func crumbColorAt(t float64) lipgloss.Color {
	r, g, b := crumbRGB(t)
	return lipgloss.Color(fmt.Sprintf("#%02x%02x%02x", r, g, b))
}

// Text-colour luminances (WCAG relative luminance), computed once at init.
var (
	crumbDarkLum  = relLuminance(0x1e, 0x1e, 0x2e) // baseHex — dark text
	crumbLightLum = relLuminance(0xcd, 0xd6, 0xf4) // crumbLightText — light text
)

// crumbTextAt keeps the segment name legible across the crust→blue span by
// picking whichever of dark/light text has the higher WCAG contrast ratio with
// the segment background — so text flips to dark exactly where dark actually
// becomes more readable (~60% along the gradient toward blue), not at an
// arbitrary perceived-luminance cutoff.
func crumbTextAt(t float64) lipgloss.Color {
	bg := relLuminance(crumbRGB(t))
	if contrastRatio(crumbLightLum, bg) > contrastRatio(crumbDarkLum, bg) {
		return lipgloss.Color(crumbLightText)
	}
	return lipgloss.Color(baseHex)
}

// relLuminance is an sRGB colour's WCAG relative luminance (0..1).
func relLuminance(r, g, b int) float64 {
	lin := func(c int) float64 {
		s := float64(c) / 255
		if s <= 0.04045 {
			return s / 12.92
		}
		return math.Pow((s+0.055)/1.055, 2.4)
	}
	return 0.2126*lin(r) + 0.7152*lin(g) + 0.0722*lin(b)
}

// contrastRatio is the WCAG contrast ratio between two relative luminances.
func contrastRatio(a, b float64) float64 {
	if a < b {
		a, b = b, a
	}
	return (a + 0.05) / (b + 0.05)
}

// crumbT maps segment i of n to its gradient position; the current directory
// (i=n-1, or the sole segment of a root-only path) sits at 1.
func crumbT(i, n int) float64 {
	if n <= 1 {
		return 1
	}
	return float64(i) / float64(n-1)
}

// crumbColor is the background colour for path segment i of n: root (i=0) = crust,
// current (i=n-1) = blue, the middle evenly interpolated.
func crumbColor(i, n int) lipgloss.Color { return crumbColorAt(crumbT(i, n)) }

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
// crust folder + active-tab-numeral chip, then one chip per path segment
// coloured along the ZLC depth gradient (root→current = crust→blue).
// Unlike a panel title it never dims — the header is always the live "you are
// here". When the bar overflows the width, front segments shrink to their
// initial (~/Documents/x → ~/D/x); if that is not enough the middle collapses to
// … keeping root + as many tail segments as fit, then a final hard clip.
func (m AppModel) headerBar(w int) string {
	glyph := string(rune(0xf07c)) // nf-fa-folder-open
	folderChip := m.folderChip(tabNumeral(m.tab) + " " + glyph + " ")
	segs := breadcrumbSegments(m.active().dir, w-dispWidth(folderChip))
	return padDisp(folderChip+renderCrumb(segs), w)
}

// folderChip is the leading "you are here" chip: a round-left cap + a body whose
// text takes the same WCAG-contrast colour as the crumbs (crust start = dark
// chip → light text), carrying the folder glyph and the active tab's Roman numeral.
func (m AppModel) folderChip(label string) string {
	c := crumbColorAt(0) // crust = the gradient start (root), so the bar opens dark
	cap := lipgloss.NewStyle().Foreground(c)
	body := lipgloss.NewStyle().Foreground(crumbTextAt(0)).Background(c).Bold(true)
	return cap.Render(capLeft) + body.Render(label)
}

// renderCrumb draws the path portion of the breadcrumb: a capHard triangle
// transition into each segment chip (coloured by crumbColor), then a capHard
// tail in the current segment's colour on the terminal background. The first
// transition blends from the folder chip's crust.
func renderCrumb(segs []string) string {
	if len(segs) == 0 {
		return ""
	}
	prev := crumbColorAt(0) // folder chip colour (crust = gradient start)
	var b strings.Builder
	n := len(segs)
	for i, seg := range segs {
		t := crumbT(i, n)
		c := crumbColorAt(t)
		// §8.2: the cap separates, so no leading space after it — only a trailing
		// space closes each segment before the next cap. Text colour flips with the
		// background luminance so the name stays legible across the crust→blue span.
		b.WriteString(lipgloss.NewStyle().Foreground(prev).Background(c).Render(capHard))
		b.WriteString(lipgloss.NewStyle().Foreground(crumbTextAt(t)).Background(c).Bold(true).Render(seg + " "))
		prev = c
	}
	b.WriteString(lipgloss.NewStyle().Foreground(prev).Render(capRight)) // rounded tail, bookends capLeft
	return b.String()
}

// fitPathSegments shrinks path segments until fits reports they satisfy the
// budget, in three progressive stages: full names first, then front-to-back
// initial abbreviation (never the current / last segment), then a middle …
// collapse keeping the root + as many tail segments as fit, then the current
// segment alone. The fits predicate lets each caller measure in its own units —
// the header measures rendered powerline width, panel [1] plain path width.
func fitPathSegments(segs []string, fits func([]string) bool) []string {
	if fits(segs) {
		return segs
	}
	for i := 0; i < len(segs)-1; i++ { // stage 2: front initials, one at a time
		segs[i] = firstRune(segs[i])
		if fits(segs) {
			return segs
		}
	}
	// stage 3: collapse the middle to …, growing the tail until one more overflows.
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
	return []string{segs[len(segs)-1]} // stage 4: current segment alone
}

// breadcrumbSegments shrinks a directory's path segments to fit budget display
// cells of rendered powerline width (see fitPathSegments); padDisp clips if even
// the current segment alone overflows.
func breadcrumbSegments(dir string, budget int) []string {
	return fitPathSegments(pathSegments(dir), func(s []string) bool {
		return dispWidth(renderCrumb(s)) <= budget
	})
}

// joinSegs rebuilds a path string from segments, keeping a leading "/" root from
// doubling (pathSegments emits "/" as its own first segment for absolute paths).
func joinSegs(segs []string) string {
	if len(segs) > 0 && segs[0] == "/" {
		return "/" + strings.Join(segs[1:], "/")
	}
	return strings.Join(segs, "/")
}

// fitPath shortens a directory into a plain (home-folded) path string of at most
// w display cells, using the same progressive scheme as the header breadcrumb
// (full → front initials → middle …). Used by panel [1] so pinned dirs and the
// header shorten the same way.
func fitPath(dir string, w int) string {
	return joinSegs(fitPathSegments(pathSegments(dir), func(s []string) bool {
		return dispWidth(joinSegs(s)) <= w
	}))
}
