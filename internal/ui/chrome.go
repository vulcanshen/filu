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
	// flush both sides — no padding space, so the title sits between the caps.
	return cap.Render(capLeft) + chip.Render(title) + cap.Render(capRight)
}

// carouselChip is filu's narrow-panel tab strategy: when a tabbed panel is too
// thin to show a full starship tab bar, collapse it to [N] + the active label as
// one bright chip, a hard cap, then just the NEXT tab's initial (recessed) — a
// carousel hint that more tabs exist without spending the width to name them.
// Panels prefer the full tabBar and only reach for this on overflow (see
// listTitle / carryTitle). Kept as the reference implementation of the pattern.
func carouselChip(num string, labels []string, active int, focused bool) string {
	bc := borderColor(focused)
	crust := lipgloss.Color(crustHex)
	base := lipgloss.Color(baseHex)
	n := len(labels)

	capOn := lipgloss.NewStyle().Foreground(bc)                              // round-left on panel bg
	capDark := lipgloss.NewStyle().Foreground(crust)                         // round-right closing a crust segment
	bright := lipgloss.NewStyle().Foreground(base).Background(bc).Bold(true) // [N] + active tab
	recessed := lipgloss.NewStyle().Foreground(bc).Background(crust)         // neighbour initials
	brToRe := lipgloss.NewStyle().Foreground(bc).Background(crust)           // hard cap bright -> recessed

	return capOn.Render(capLeft) +
		bright.Render(num+" "+labels[active]) +
		brToRe.Render(capHard) +
		recessed.Render(firstRune(labels[(active+1)%n])) + // next wraps
		capDark.Render(capRight)
}

func firstRune(s string) string {
	for _, r := range s {
		return string(r)
	}
	return ""
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
		lead := "" // after a cap, the cap itself separates — no leading space (§8.2)
		switch {
		case prevBright && cur:
			lead = " " // merge: no cap, one space separates from the previous chip
		case prevBright && !cur:
			b.WriteString(lipgloss.NewStyle().Foreground(bc).Background(crust).Render(capHard))
		case !prevBright && cur:
			b.WriteString(lipgloss.NewStyle().Foreground(crust).Background(bc).Render(capHard))
		default:
			b.WriteString(lipgloss.NewStyle().Foreground(chev).Background(crust).Render(capThin))
		}
		seg := lead + lab + " "
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
