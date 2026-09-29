package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// drawPopupBox renders the shared rounded popup box (kbu form): title embedded in
// the top border, hint in the bottom border, pre-styled rows between two padding
// rows. rows must already be clipped to innerW by the caller.
func drawPopupBox(bc lipgloss.Color, title, hint string, rows []string, innerW int) string {
	return drawPopupBoxPad(bc, title, hint, rows, innerW, true)
}

// drawPopupBoxPad is drawPopupBox with control over the blank padding rows that
// frame the content. pad=false makes the content hug the borders (kbu's YAML
// popup form — used by the panel [2] yank viewport and the finder). hint comes
// already styled (keyLegend: key Blue, colon and description Overlay0, tdp M5).
func drawPopupBoxPad(bc lipgloss.Color, title, hint string, rows []string, innerW int, pad bool) string {
	bStyle := lipgloss.NewStyle().Foreground(bc)
	tStyle := lipgloss.NewStyle().Foreground(bc).Bold(true)

	// A title / hint wider than the box would push its border out and, when the
	// box is joined beside another, open a gap — clip both to fit. Measured with
	// dispWidth: a title glyph (the loading icon, a warning sign) takes two cells
	// on a CJK icon font, and the border must shorten to match.
	if dispWidth(title) > innerW-1 {
		title = truncate(title, innerW-1)
	}
	if dispWidth(hint) > innerW-1 {
		hint = truncate(hint, innerW-1)
	}

	var b strings.Builder
	dashesTop := max(0, innerW-1-dispWidth(title))
	b.WriteString(bStyle.Render("╭─") + tStyle.Render(title) + bStyle.Render(strings.Repeat("─", dashesTop)+"╮") + "\n")
	left, right := bStyle.Render("│"), bStyle.Render("│")
	padRow := left + strings.Repeat(" ", innerW) + right + "\n"
	if pad {
		b.WriteString(padRow)
	}
	for _, line := range rows {
		p := max(0, innerW-lipgloss.Width(line))
		b.WriteString(left + line + strings.Repeat(" ", p) + right + "\n")
	}
	if pad {
		b.WriteString(padRow)
	}
	dashesBot := max(0, innerW-dispWidth(hint)-1)
	b.WriteString(bStyle.Render("╰─") + hint + bStyle.Render(strings.Repeat("─", dashesBot)+"╯"))
	return b.String()
}

// popupMaxWidth is the widest a popup gets on a wide screen (tdp F7).
const popupMaxWidth = 120

// popupInnerWidth is the inner width every popup has (tdp F7): the box is the
// terminal less one column on each side, at most popupMaxWidth, whatever its
// content — so a popup's shape is known before it opens. The inner width is
// that less the two side borders. Unsized (tests) it is 40.
func popupInnerWidth(screenW int) int {
	if screenW <= 0 {
		return 40
	}
	return max(min(screenW-2, popupMaxWidth)-2, 20)
}
