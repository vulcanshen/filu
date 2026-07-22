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
	// no leading space — [N] sits flush against the cap, matching the tab bar.
	return cap.Render(capLeft) + chip.Render(title+" ") + cap.Render(capRight)
}

// carouselChip is a compact single-tab title: [N] plus the active label in full,
// flanked by the neighbouring tabs' first letters separated by a › chevron
// (first tab has no left neighbour, last tab no right). For narrow columns.
func carouselChip(num string, labels []string, active int, focused bool) string {
	bc := borderColor(focused)
	cap := lipgloss.NewStyle().Foreground(bc)
	chip := lipgloss.NewStyle().Foreground(lipgloss.Color(baseHex)).Background(bc).Bold(true)
	n := len(labels)
	prefix := " " + firstRune(labels[(active-1+n)%n]) + arrowR // prev wraps (first tab shows last)
	suffix := ""
	if active < n-1 { // no right neighbour on the last tab
		suffix = arrowR + firstRune(labels[active+1])
	}
	return cap.Render(capLeft) + chip.Render(num+prefix+labels[active]+suffix) + cap.Render(capRight)
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
