package ui

import (
	"strings"
	"unicode"

	"github.com/charmbracelet/lipgloss"
)

// singleLine is what a single-line value keeps of text that comes in as runes —
// typed, pasted (one KeyRunes holds a whole bracketed paste) or prefilled. A
// line break or a tab stays, \r\n as one \n, so the value is what the user gave
// and Backspace takes a break whole; every other control character (C0, DEL,
// C1) is dropped, so an ESC never reaches the terminal.
func singleLine(s string) string {
	return strings.Map(func(r rune) rune {
		if r != '\n' && r != '\r' && r != '\t' && unicode.IsControl(r) {
			return -1
		}
		return r
	}, strings.ReplaceAll(s, "\r\n", "\n"))
}

// hasBreak reports whether a single-line value holds a line break or a tab —
// what a value that gets used can't be submitted with.
func hasBreak(s string) bool { return strings.ContainsAny(s, "\n\r\t") }

// valueTail draws pre + v + post in w cells, keeping the tail (the cursor) like
// truncPathLeft. Each line break in v is drawn as \n and each tab as \t: two
// cells, never cut — a cut landing inside one drops it and a space keeps its
// cell. red draws them in Red, apart from a \ and n typed by hand; a row drawn
// in one colour leaves it unset and colours the whole row.
func valueTail(pre, v, post string, w int, red bool) string {
	type unit struct {
		s   string
		esc bool
	}
	var units []unit
	for _, r := range pre {
		units = append(units, unit{s: string(r)})
	}
	for _, r := range v {
		switch r {
		case '\n', '\r':
			units = append(units, unit{`\n`, true})
		case '\t':
			units = append(units, unit{`\t`, true})
		default:
			units = append(units, unit{s: string(r)})
		}
	}
	for _, r := range post {
		units = append(units, unit{s: string(r)})
	}

	var b strings.Builder
	total := 0
	for _, u := range units {
		total += dispWidth(u.s)
	}
	if total > w {
		if w <= 0 {
			return ""
		}
		start, used := len(units), 0
		for start > 0 && used+dispWidth(units[start-1].s) <= w-1 {
			start--
			used += dispWidth(units[start].s)
		}
		b.WriteString("…" + strings.Repeat(" ", w-1-used))
		units = units[start:]
	}
	mark := lipgloss.NewStyle().Foreground(lipgloss.Color("#f38ba8"))
	for _, u := range units {
		if u.esc && red {
			b.WriteString(mark.Render(u.s))
		} else {
			b.WriteString(u.s)
		}
	}
	return b.String()
}
