package ui

import (
	"strings"

	"github.com/charmbracelet/x/ansi"
)

// IconCells reports the detected Nerd Font icon cell width (1 or 2) — exposed
// for the `filu iconwidth` debug command.
func IconCells() int { return iconCells }

// iconCells is how many terminal cells a Nerd Font file-type icon actually
// occupies. On a normal Nerd Font it is 1; on a CJK "full-width icon" font
// (e.g. Maple Mono NF CN) the icons are drawn 2 cells wide to align to the CJK
// grid, while lipgloss/x-ansi still measure them as 1 — that mismatch is what
// breaks the panel borders. DetectIconWidth (CPR probe, startup) sets this; the
// default of 1 means "no adjustment", so nothing changes on a normal font.
var iconCells = 1

// isWideIcon reports whether r is a Nerd Font file-type glyph that a CJK icon
// font renders double-width. The powerline caps (U+E0A0–E0D7, the tab-bar
// triangles/rounds) live in the PUA too but render single-width, so they are
// excluded — only file-type icons get the +1 treatment.
func isWideIcon(r rune) bool {
	if r >= 0x2160 && r <= 0x2164 {
		return true // Ⅰ..Ⅴ tab numerals: ambiguous width, drawn wide on CJK fonts
	}
	if r >= 0xe0a0 && r <= 0xe0d7 {
		return false // powerline caps — single-width even on CJK icon fonts
	}
	// BMP Private Use Area + supplementary PUA-A (Material Design icons).
	return (r >= 0xe000 && r <= 0xf8ff) || (r >= 0xf0000 && r <= 0xffffd)
}

// iconCount counts wide-icon runes in s (ANSI stripped). Fast-pathed to 0 when
// icons are single-width, so dispWidth == ansi.StringWidth on a normal font.
func iconCount(s string) int {
	if iconCells == 1 {
		return 0
	}
	n := 0
	for _, r := range ansi.Strip(s) {
		if isWideIcon(r) {
			n++
		}
	}
	return n
}

// dispWidth is the on-screen width of s: the measured width plus the extra cell
// each file-type icon eats on a CJK icon font.
func dispWidth(s string) int {
	return ansi.StringWidth(s) + iconCount(s)*(iconCells-1)
}

// dispClip trims s to display width w (ANSI- and wide-icon-aware), no ellipsis.
// Icons sit at the line start, so trailing cells removed are single-width text —
// one measured cell freed == one display cell freed; the loop is a safety net
// for the rare case a trim crosses an icon.
func dispClip(s string, w int) string {
	if w <= 0 {
		return ""
	}
	if dispWidth(s) <= w {
		return s
	}
	target := ansi.StringWidth(s) - (dispWidth(s) - w)
	for target > 0 {
		out := ansi.Truncate(s, target, "")
		if dispWidth(out) <= w {
			return out
		}
		target--
	}
	return ""
}

// padDisp clips then space-pads s to exactly display width w.
func padDisp(s string, w int) string {
	s = dispClip(s, w)
	if d := w - dispWidth(s); d > 0 {
		s += strings.Repeat(" ", d)
	}
	return s
}

// truncate clips s to display width w, appending "…" when it had to cut. Icons
// are all near the line start, so keeping them costs iconCount*(iconCells-1)
// extra display cells the "…"-budget must reserve; the loop tightens the rare
// off-by-one from an icon landing at the cut.
func truncate(s string, w int) string {
	if w <= 0 {
		return ""
	}
	if dispWidth(s) <= w {
		return s
	}
	target := max(w-iconCount(s)*(iconCells-1), 1)
	for {
		out := ansi.Truncate(s, target, "…")
		if dispWidth(out) <= w || target <= 1 {
			return out
		}
		target--
	}
}

// truncPathLeft clips s to display width w from the LEFT, keeping the tail
// (filename) visible and prepending "…" — the right choice for paths, where the
// end matters more than the root. Paths carry no wide icons, so the measured and
// display widths coincide.
func truncPathLeft(s string, w int) string {
	if w <= 0 {
		return ""
	}
	if dispWidth(s) <= w {
		return s
	}
	return ansi.TruncateLeft(s, dispWidth(s)-(w-1), "…")
}

// joinH lays multi-line blocks side by side. Each block's lines are padded to
// that block's own display width, so a wide icon in one column never shoves the
// next column left. Replaces lipgloss.JoinHorizontal, whose width maths is
// icon-blind.
func joinH(blocks ...string) string {
	rows := make([][]string, len(blocks))
	widths := make([]int, len(blocks))
	maxRows := 0
	for i, b := range blocks {
		rows[i] = strings.Split(b, "\n")
		for _, ln := range rows[i] {
			if wd := dispWidth(ln); wd > widths[i] {
				widths[i] = wd
			}
		}
		if len(rows[i]) > maxRows {
			maxRows = len(rows[i])
		}
	}
	var out strings.Builder
	for r := 0; r < maxRows; r++ {
		for i := range rows {
			if r < len(rows[i]) {
				out.WriteString(padDisp(rows[i][r], widths[i]))
			} else {
				out.WriteString(strings.Repeat(" ", widths[i]))
			}
		}
		if r < maxRows-1 {
			out.WriteByte('\n')
		}
	}
	return out.String()
}

// joinV stacks blocks, left-aligned, every line padded to the widest display
// width. Replaces lipgloss.JoinVertical (icon-blind).
func joinV(blocks ...string) string {
	var lines []string
	maxW := 0
	for _, b := range blocks {
		for _, ln := range strings.Split(b, "\n") {
			if wd := dispWidth(ln); wd > maxW {
				maxW = wd
			}
			lines = append(lines, ln)
		}
	}
	for i := range lines {
		lines[i] = padDisp(lines[i], maxW)
	}
	return strings.Join(lines, "\n")
}
