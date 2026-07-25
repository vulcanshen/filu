package ui

import (
	"strings"
	"testing"
)

func TestCrumbColorEndpoints(t *testing.T) {
	// Root of a path = blue; current = crust; a single-segment path is the crust
	// (current) endpoint.
	if got := string(crumbColor(0, 4)); got != "#89b4fa" {
		t.Errorf("crumbColor(0,4) = %s, want blue #89b4fa (root)", got)
	}
	if got := string(crumbColor(3, 4)); got != "#11111b" {
		t.Errorf("crumbColor(3,4) = %s, want crust #11111b (current)", got)
	}
	if got := string(crumbColor(0, 1)); got != "#11111b" {
		t.Errorf("crumbColor(0,1) = %s, want crust (single segment = current)", got)
	}
}

// TestCrumbGradientMonotone: perceived luminance must fall monotonically from
// the blue root to the crust current, so the z-axis never reverses across depth.
func TestCrumbGradientMonotone(t *testing.T) {
	lum := func(t float64) int { r, g, b := crumbRGB(t); return (299*r + 587*g + 114*b) / 1000 }
	prev := 1 << 30
	for i := 0; i <= 10; i++ {
		if l := lum(float64(i) / 10); l > prev {
			t.Fatalf("luminance rose at t=%.1f (%d > %d)", float64(i)/10, l, prev)
		} else {
			prev = l
		}
	}
}

// TestCrumbTextFlips: dark text on the bright blue end (and still dark a third of
// the way in — the flip must not come too early), light text on the dark crust
// end, so the segment name never renders dark-on-dark.
func TestCrumbTextFlips(t *testing.T) {
	if got := string(crumbTextAt(0)); got != baseHex {
		t.Errorf("crumbTextAt(0) = %s, want dark %s on the blue end", got, baseHex)
	}
	if got := string(crumbTextAt(0.3)); got != baseHex {
		t.Errorf("crumbTextAt(0.3) = %s, want dark %s (flip must not be too early)", got, baseHex)
	}
	if got := string(crumbTextAt(1)); got != crumbLightText {
		t.Errorf("crumbTextAt(1) = %s, want light %s on the crust end", got, crumbLightText)
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
