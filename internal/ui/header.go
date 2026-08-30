package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

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

// crumbRow renders a tab's location breadcrumb as plain lavender text with dim
// slash separators — no chip backgrounds, so it can't fight the border tab
// pills right above it. It is panel [1]'s first content row — the live "you
// are here". No tab numeral: the panel title's tab bar already marks which tab
// this is. Always a single line: when the path overflows w, front segments
// shrink to their initial (~/Documents/x → ~/D/x); if that is not enough the
// middle collapses to … keeping root + as many tail segments as fit, then a
// final hard clip.
func crumbRow(dir string, w int) string {
	segs := breadcrumbSegments(dir, w-1) // one cell of left margin off the border
	return truncate(padDisp(" "+renderCrumb(segs), w), w)
}

// renderCrumb draws the path portion of the breadcrumb: each segment in
// lavender, joined by a dim slash — no spaces, so the row reads as the literal
// path string. An absolute path's "/" root segment becomes the leading slash
// itself (not a "//" double-up), matching joinSegs.
func renderCrumb(segs []string) string {
	if len(segs) == 0 {
		return ""
	}
	sep := lipgloss.NewStyle().Foreground(dimColor).Render("/")
	text := lipgloss.NewStyle().Foreground(userColor)
	parts := make([]string, len(segs))
	for i, seg := range segs {
		parts[i] = text.Render(seg)
	}
	if segs[0] == "/" {
		return sep + strings.Join(parts[1:], sep)
	}
	return strings.Join(parts, sep)
}

// fitPathSegments shrinks path segments until fits reports they satisfy the
// budget, in three progressive stages: full names first, then front-to-back
// initial abbreviation (never the current / last segment), then a middle …
// collapse keeping the root + as many tail segments as fit, then the current
// segment alone. The fits predicate lets each caller measure in its own units —
// the breadcrumb measures its rendered width, the Goto picker plain path width.
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
// cells of rendered width (see fitPathSegments); padDisp clips if even the
// current segment alone overflows.
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
// (full → front initials → middle …). Used by the Goto picker's Pinned list so
// pinned dirs and the header shorten the same way.
func fitPath(dir string, w int) string {
	return joinSegs(fitPathSegments(pathSegments(dir), func(s []string) bool {
		return dispWidth(joinSegs(s)) <= w
	}))
}
