package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func borderColor(focused bool) lipgloss.Color {
	if focused {
		return focusColor
	}
	return borderDim
}

// singleChip is a panel's border title as one powerline chip (§8.1):
// round-left cap + dark-on-border-colour body + round-right cap.
func singleChip(title string, focused bool) string {
	bc := borderColor(focused)
	cap := lipgloss.NewStyle().Foreground(bc)
	chip := lipgloss.NewStyle().Foreground(lipgloss.Color(baseHex)).Background(bc).Bold(true)
	return cap.Render(capLeft) + chip.Render(" "+title+" ") + cap.Render(capRight)
}

// tabBar renders a starship powerline chip chain (§8.2): a bright [N] chip, then
// one chip per tab — active bright (border colour), inactive recessed on crust.
func tabBar(num string, labels []string, active int, focused bool) string {
	bc := borderColor(focused)
	crust := lipgloss.Color(crustHex)
	chev := lipgloss.Color(baseHex) // inactive↔inactive divider, dim when unfocused
	if focused {
		chev = lipgloss.Color("#313244") // surface0 when focused
	}
	capOnBase := lipgloss.NewStyle().Foreground(bc)
	bright := lipgloss.NewStyle().Foreground(lipgloss.Color(baseHex)).Background(bc).Bold(true)
	recessed := lipgloss.NewStyle().Foreground(bc).Background(crust)

	var b strings.Builder
	b.WriteString(capOnBase.Render(capLeft))
	b.WriteString(bright.Render(num)) // [N] chip, always bright
	prevBright := true

	for i, lab := range labels {
		cur := i == active
		switch {
		case prevBright && cur:
			// merge with the previous bright chip — no boundary cap
		case prevBright && !cur:
			b.WriteString(lipgloss.NewStyle().Foreground(bc).Background(crust).Render(capHard))
		case !prevBright && cur:
			b.WriteString(lipgloss.NewStyle().Foreground(crust).Background(bc).Render(capHard))
		default:
			b.WriteString(lipgloss.NewStyle().Foreground(chev).Background(crust).Render(capThin))
		}
		seg := " " + lab + " "
		if cur {
			b.WriteString(bright.Render(seg))
		} else {
			b.WriteString(recessed.Render(seg))
		}
		prevBright = cur
	}

	if prevBright {
		b.WriteString(capOnBase.Render(capRight))
	} else {
		b.WriteString(lipgloss.NewStyle().Foreground(crust).Render(capRight))
	}
	return b.String()
}
