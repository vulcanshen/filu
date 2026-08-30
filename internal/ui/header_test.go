package ui

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

// TestCrumbRowFitsPanel: the breadcrumb row (panel [1]'s first content line) is
// always a single line of at most w display cells, even for a deep path in a
// narrow panel — it shrinks, never wraps.
func TestCrumbRowFitsPanel(t *testing.T) {
	deep := "/very/long/path/that/goes/on/and/on/with/many/segments/deep-directory-name"
	for _, w := range []int{20, 40, 78} {
		row := crumbRow(deep, w)
		if strings.Contains(row, "\n") {
			t.Fatalf("crumbRow(w=%d) must be a single line", w)
		}
		if got := dispWidth(row); got != w {
			t.Errorf("crumbRow(w=%d) width = %d, want exactly %d (padded/clipped)", w, got, w)
		}
	}
}

// TestListBodyStartsWithCrumb: panel [1]'s inner content opens with the active
// tab's breadcrumb row; the file list follows below it.
func TestListBodyStartsWithCrumb(t *testing.T) {
	m := minModel()
	m.width, m.height = 100, 30
	dir := t.TempDir()
	m.tabs = []listModel{newList(dir)}
	m.tab = 0

	body := m.listBody(0, 60, 10, true)
	lines := strings.Split(body, "\n")
	if len(lines) < 2 {
		t.Fatalf("list body should have a crumb row + list, got %d line(s)", len(lines))
	}
	if !strings.Contains(ansi.Strip(lines[0]), filepath.Base(dir)) {
		t.Errorf("first line should be the breadcrumb (ends at %q), got %q", filepath.Base(dir), lines[0])
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

// TestRenderCrumbAbsoluteRoot: with plain-text "/" separators, an absolute
// path's root segment must render as the leading slash, never a "//" double-up.
func TestRenderCrumbAbsoluteRoot(t *testing.T) {
	if got := ansi.Strip(renderCrumb([]string{"/", "usr", "local"})); got != "/usr/local" {
		t.Errorf("absolute path rendered %q, want /usr/local", got)
	}
	if got := ansi.Strip(renderCrumb([]string{"/"})); got != "/" {
		t.Errorf("bare root rendered %q, want /", got)
	}
	if got := ansi.Strip(renderCrumb([]string{"~", "proj"})); got != "~/proj" {
		t.Errorf("home path rendered %q, want ~/proj", got)
	}
}

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
