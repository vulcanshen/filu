package ui

import (
	"fmt"
	"strings"
	"testing"
)

// hexRGB parses "#rrggbb" into its three channel ints.
func hexRGB(t *testing.T, s string) (int, int, int) {
	t.Helper()
	var r, g, b int
	if _, err := fmt.Sscanf(s, "#%02x%02x%02x", &r, &g, &b); err != nil {
		t.Fatalf("bad hex %q: %v", s, err)
	}
	return r, g, b
}

func TestGradientColorStops(t *testing.T) {
	// Endpoints are exact; the interior stops match kbu's published Lavenphire
	// scale to within ±1 LSB (naive sRGB vs linear-RGB lerp).
	cases := []struct {
		t         float64
		want      string
		tolerance int
	}{
		{0.0, "#b4befe", 0},  // Lavender anchor
		{0.25, "#a4c0fa", 1}, // Lavenphire25
		{0.50, "#94c3f5", 1}, // Lavenphire50
		{0.75, "#84c5f0", 1}, // Lavenphire75
		{1.0, "#74c7ec", 0},  // Sapphire ceiling
	}
	for _, c := range cases {
		got := string(gradientColor(c.t))
		gr, gg, gb := hexRGB(t, got)
		wr, wg, wb := hexRGB(t, c.want)
		off := abs(gr-wr) + abs(gg-wg) + abs(gb-wb)
		if off > c.tolerance {
			t.Errorf("gradientColor(%.2f) = %s, want ~%s (off %d > tol %d)", c.t, got, c.want, off, c.tolerance)
		}
	}
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

// TestGradientColorMonotone: blue must fall monotonically from Lavender→Sapphire
// (254→236) as t rises, confirming the interpolation never reverses the z-axis.
func TestGradientColorMonotone(t *testing.T) {
	prev := 999
	for i := 0; i <= 10; i++ {
		_, _, b := hexRGB(t, string(gradientColor(float64(i)/10)))
		if b > prev {
			t.Fatalf("blue rose at t=%.1f (%d > %d): z-axis reversed", float64(i)/10, b, prev)
		}
		prev = b
	}
}

func TestCrumbColorEndpoints(t *testing.T) {
	// Root of a multi-segment path = Lavenphire25 (t=0.25); current = Sapphire.
	if got := string(crumbColor(0, 4)); got != string(gradientColor(0.25)) {
		t.Errorf("crumbColor(0,4) = %s, want Lavenphire25 %s", got, gradientColor(0.25))
	}
	if got := string(crumbColor(3, 4)); got != string(gradientColor(1)) {
		t.Errorf("crumbColor(3,4) = %s, want Sapphire %s", got, gradientColor(1))
	}
	// Single-segment path (root == current) is the Sapphire endpoint.
	if got := string(crumbColor(0, 1)); got != string(gradientColor(1)) {
		t.Errorf("crumbColor(0,1) = %s, want Sapphire %s", got, gradientColor(1))
	}
}

func TestPathSegments(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{"/", []string{"/"}},
		{"/etc/nginx", []string{"/", "etc", "nginx"}},
		{"/etc/nginx/", []string{"/", "etc", "nginx"}}, // trailing slash dropped
		{"~", []string{"~"}},
		{"~/Documents/filu", []string{"~", "Documents", "filu"}},
	}
	for _, c := range cases {
		got := pathSegments(c.in)
		if strings.Join(got, "|") != strings.Join(c.want, "|") {
			t.Errorf("pathSegments(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

// crumbWidth is the on-screen width of the rendered path portion (same measure
// breadcrumbSegments uses to decide fit).
func crumbWidth(segs []string) int { return dispWidth(renderCrumb(segs)) }

func TestBreadcrumbSegmentsFits(t *testing.T) {
	// An absolute path (not under $HOME) is returned by shortPath unchanged, so
	// this is independent of the test machine's home dir.
	const dir = "/aaa/bbb/ccc/ddd"
	full := pathSegments(dir) // ["/","aaa","bbb","ccc","ddd"]
	fullW := crumbWidth(full)

	// Budget ≥ full width → untouched.
	if got := breadcrumbSegments(dir, fullW); strings.Join(got, "|") != strings.Join(full, "|") {
		t.Errorf("full-budget = %v, want %v", got, full)
	}

	// Every budget from wide down to tiny must either fit the budget or reduce to
	// the single current segment (the only case padDisp is allowed to clip).
	for budget := fullW; budget >= 1; budget-- {
		got := breadcrumbSegments(dir, budget)
		if len(got) == 0 {
			t.Fatalf("budget %d: empty result", budget)
		}
		if crumbWidth(got) > budget && len(got) != 1 {
			t.Errorf("budget %d: %v width %d overflows and is not a lone segment", budget, got, crumbWidth(got))
		}
		// The current directory ("ddd") must always survive as the last segment.
		if got[len(got)-1] != "ddd" {
			t.Errorf("budget %d: %v dropped the current segment", budget, got)
		}
	}
}

func TestBreadcrumbSegmentsAbbreviatesFrontFirst(t *testing.T) {
	const dir = "/aaa/bbb/ccc/ddd"
	full := pathSegments(dir)
	// One cell under full forces exactly the first non-root segment to abbreviate
	// (root "/" is already one rune) while the rest stay full.
	got := breadcrumbSegments(dir, crumbWidth(full)-1)
	want := []string{"/", "a", "bbb", "ccc", "ddd"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Errorf("got %v, want front-abbreviated %v", got, want)
	}
}

func TestBreadcrumbSegmentsMiddleEllipsis(t *testing.T) {
	const dir = "/aaa/bbb/ccc/ddd"
	// One cell under the fully-abbreviated width cannot fit even the all-initials
	// chain, forcing the middle … collapse; root and current must remain.
	fullyAbbrev := []string{"/", "a", "b", "c", "ddd"}
	got := breadcrumbSegments(dir, crumbWidth(fullyAbbrev)-1)
	if got[0] != "/" || got[len(got)-1] != "ddd" {
		t.Errorf("got %v, want root + current preserved", got)
	}
	joined := strings.Join(got, "|")
	if !strings.Contains(joined, "…") {
		t.Errorf("got %v, want a … collapse", got)
	}
}
