package ui

import (
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	terminu "github.com/vulcanshen/terminu-go"
)

// The colours of filu's own content (catppuccin-mocha, as the family's). What
// is terminu's — frames, cursors, hints, popups — terminu colours.
var (
	userColor = lipgloss.Color("#b4befe") // lavender: where you are, the user's footprint
	dimColor  = lipgloss.Color("#6c7086") // overlay0: quiet text
)

// maxTabs caps [1]'s directory tabs: it opens with one, t adds one, w closes
// one. Five keep the tab chain and the zoomed columns bounded.
const maxTabs = 5

// tabMarks name the tabs by their place: one glyph each, always as wide, so the
// tab chain does not change width with the directories (tdp L2). The first is
// the launch glyph: the first tab always opens where filu started.
var tabMarks = []string{
	iconCWD,      // nf-md-rocket_launch
	"\U000F011B", // nf-md-cat
	"\U000F0A43", // nf-md-dog
	"\U000F03E9", // nf-md-paw
	"\U000F0AB0", // nf-md-egg_easter
}

// tabMark is the idx-th (0-based) tab's glyph.
func tabMark(idx int) string {
	if idx >= 0 && idx < len(tabMarks) {
		return tabMarks[idx]
	}
	return ""
}

// shortPath folds the home directory to ~.
func shortPath(p string) string {
	if home, err := os.UserHomeDir(); err == nil && home != "" && strings.HasPrefix(p, home) {
		return "~" + strings.TrimPrefix(p, home)
	}
	return p
}

// eza's long-format colours (catppuccin-mocha): read yellow, write red,
// execute green; the owner yellow, its group in subtext.
var (
	ezaYellow = terminu.Hex("#f9e2af")
	ezaRed    = terminu.Hex("#f38ba8")
	ezaGreen  = terminu.Hex("#a6e3a1")
	ezaPeach  = terminu.Hex("#fab387")
	ezaGroup  = terminu.Hex("#bac2de")
)

// colorPerm paints a mode string (drwxr-xr-x) as eza does: the type Blue,
// read yellow, write red, execute green, a bare - quiet.
func colorPerm(perm string) []terminu.Span {
	var out []terminu.Span
	for i, r := range perm {
		c := terminu.Overlay0
		switch {
		case i == 0:
			c = terminu.Blue
		case r == 'r':
			c = ezaYellow
		case r == 'w':
			c = ezaRed
		case r == 'x' || r == 's' || r == 't' || r == 'S' || r == 'T':
			c = ezaGreen
		}
		out = append(out, terminu.Span{Text: string(r), Style: terminu.Fg(c)})
	}
	return out
}

// colorSize paints a file's size warmer as it grows, as eza's colour scale: green
// under 1 MiB, yellow under 1 GiB, peach under 1 TiB, red beyond. A directory
// has no size (filu never adds up a tree): a quiet dash.
func colorSize(it fileItem) []terminu.Span {
	if it.isDir {
		return []terminu.Span{{Text: "-", Style: terminu.Fg(terminu.Overlay0)}}
	}
	c := ezaGreen
	switch {
	case it.size >= 1<<40:
		c = ezaRed
	case it.size >= 1<<30:
		c = ezaPeach
	case it.size >= 1<<20:
		c = ezaYellow
	}
	return []terminu.Span{{Text: compactSize(it.size), Style: terminu.Fg(c)}}
}

// colorOwner paints "owner:group" as eza does.
func colorOwner(s string) []terminu.Span {
	if o, g, ok := strings.Cut(s, ":"); ok {
		return []terminu.Span{{Text: o, Style: terminu.Fg(ezaYellow)}, {Text: ":", Style: terminu.Fg(terminu.Overlay0)}, {Text: g, Style: terminu.Fg(ezaGroup)}}
	}
	return []terminu.Span{{Text: s, Style: terminu.Fg(ezaYellow)}}
}

// spansText is spans as plain text: a cell's words, for the filter and the
// sort (Row.Cells).
func spansText(spans []terminu.Span) string {
	var b strings.Builder
	for _, s := range spans {
		b.WriteString(s.Text)
	}
	return b.String()
}
