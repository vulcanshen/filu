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
// popup form — used by the panel [2] yank viewport and the finder).
func drawPopupBoxPad(bc lipgloss.Color, title, hint string, rows []string, innerW int, pad bool) string {
	bStyle := lipgloss.NewStyle().Foreground(bc)
	tStyle := lipgloss.NewStyle().Foreground(bc).Bold(true)

	// A title / hint wider than the box would push its border out and, when the
	// box is joined beside another, open a gap — clip both to fit.
	if lipgloss.Width(title) > innerW-1 {
		title = truncate(title, innerW-1)
	}
	if lipgloss.Width(hint) > innerW-1 {
		hint = truncate(hint, innerW-1)
	}

	var b strings.Builder
	dashesTop := max(0, innerW-1-lipgloss.Width(title))
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
	dashesBot := max(0, innerW-lipgloss.Width(hint)-1)
	b.WriteString(bStyle.Render("╰─") + tStyle.Render(hint) + bStyle.Render(strings.Repeat("─", dashesBot)+"╯"))
	return b.String()
}

// maxInnerWidth caps a popup's inner width at 85% of the screen (min 40).
func maxInnerWidth(screenW int) int {
	if screenW <= 0 {
		return 40
	}
	return max(screenW*85/100, 40)
}
