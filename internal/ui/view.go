package ui

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// kbu colour mindset: blue = structure/focus, overlay0 = unfocused, lavender = user footprint.
var (
	focusColor = lipgloss.Color("#89b4fa") // catppuccin blue
	dimColor   = lipgloss.Color("#6c7086") // catppuccin overlay0
	pathColor  = lipgloss.Color("#b4befe") // catppuccin lavender
)

func (m AppModel) View() string {
	if m.width == 0 || m.height == 0 {
		return "" // wait for the first WindowSizeMsg
	}

	w, h := m.width, m.height
	midH := h - 2 // minus header + footer rows
	if midH < 3 || w < 24 {
		return "terminal too small"
	}

	leftW, rightW := 22, 34
	midW := w - leftW - rightW
	if midW < 20 { // narrow: collapse side columns (real zoom comes later)
		leftW, rightW, midW = 0, 0, w
	}

	listBody := m.active().view(midW-2, midH-2, m.focus == panelList)
	list := m.panelBox(panelList, m.listTitle(), midW, midH, listBody)

	middle := list
	if leftW > 0 && rightW > 0 {
		pinH := midH * 2 / 3
		left := lipgloss.JoinVertical(
			lipgloss.Left,
			m.panelBox(panelPin, "filu", leftW, pinH, m.places.view(leftW-2, pinH-2, m.focus == panelPin)),
			m.panelBox(panelCarry, "[4] carry", leftW, midH-pinH, m.carry.view(leftW-2, (midH-pinH)-2, m.focus == panelCarry)),
		)
		right := m.panelBox(panelDetail, m.detailTitle(), rightW, midH, m.detailBody(rightW-2, midH-2))
		middle = lipgloss.JoinHorizontal(lipgloss.Top, left, list, right)
	}

	return lipgloss.JoinVertical(lipgloss.Left, m.headerBar(w), middle, m.footerBar(w))
}

// listTitle renders panel [2]'s tab bar; the active tab is bracketed.
func (m AppModel) listTitle() string {
	parts := make([]string, len(m.tabs))
	for i, t := range m.tabs {
		b := pathBase(t.dir)
		if i == m.tab {
			b = "[" + b + "]"
		}
		parts[i] = b
	}
	return "[2] " + strings.Join(parts, " ")
}

// detailTitle marks panel [3]'s active tab with brackets.
func (m AppModel) detailTitle() string {
	if m.detail == tabInfo {
		return "[3] Preview [Info]"
	}
	return "[3] [Preview] Info"
}

// detailBody renders panel [3]'s active tab.
func (m AppModel) detailBody(w, rows int) string {
	if m.detail == tabInfo {
		return renderLines(infoLines(m.active().cursorItem(), m.active().dir), w, rows)
	}
	return m.preview.view(w, rows)
}

// panelBox draws a bordered panel with the title embedded in the top border
// (kbu style). Focused = double border + blue, else rounded + dim.
func (m AppModel) panelBox(id panelID, title string, w, h int, body string) string {
	focused := m.focus == id
	color := dimColor
	tl, tr, bl, br, hz, vt := "╭", "╮", "╰", "╯", "─", "│"
	if focused {
		color = focusColor
		tl, tr, bl, br, hz, vt = "╔", "╗", "╚", "╝", "═", "║"
	}
	bs := lipgloss.NewStyle().Foreground(color)
	ts := lipgloss.NewStyle().Foreground(color).Bold(focused)
	inner := max(w-2, 1)

	titleText := " " + title + " "
	if lipgloss.Width(titleText) > inner-1 {
		titleText = truncate(titleText, inner-1)
	}
	fill := max(inner-1-lipgloss.Width(titleText), 0)
	top := bs.Render(tl+hz) + ts.Render(titleText) + bs.Render(strings.Repeat(hz, fill)+tr)

	bodyLines := strings.Split(body, "\n")
	var b strings.Builder
	b.WriteString(top + "\n")
	for r := range h - 2 {
		line := ""
		if r < len(bodyLines) {
			line = bodyLines[r]
		}
		if d := inner - lipgloss.Width(line); d > 0 {
			line += strings.Repeat(" ", d)
		}
		b.WriteString(bs.Render(vt) + line + bs.Render(vt) + "\n")
	}
	b.WriteString(bs.Render(bl + strings.Repeat(hz, inner) + br))
	return b.String()
}

func (m AppModel) headerBar(w int) string {
	return lipgloss.NewStyle().Width(w).Foreground(pathColor).
		Render(truncate(" "+breadcrumb(m.active().dir), w))
}

func (m AppModel) footerBar(w int) string {
	color, content := dimColor, " Space 動作   ? 選單   Tab/1-4 切面板   q 離開"
	if m.input.kind != inputNone {
		color = focusColor
		content = " " + m.input.prompt + ": " + m.input.buffer + "█"
	}
	return lipgloss.NewStyle().Width(w).Foreground(color).Render(truncate(content, w))
}

func pathBase(p string) string {
	if b := filepath.Base(p); b != "." && b != string(filepath.Separator) {
		return b
	}
	return "/"
}

// breadcrumb turns a path into "~ › a › b › c" (home folded to ~).
func breadcrumb(p string) string {
	if home, err := os.UserHomeDir(); err == nil && strings.HasPrefix(p, home) {
		p = "~" + strings.TrimPrefix(p, home)
	}
	var segs []string
	for s := range strings.SplitSeq(p, string(filepath.Separator)) {
		if s != "" {
			segs = append(segs, s)
		}
	}
	return strings.Join(segs, " › ")
}

// truncate tail-clips to width w (middle-elision comes later).
func truncate(s string, w int) string {
	if w <= 0 {
		return ""
	}
	if lipgloss.Width(s) <= w {
		return s
	}
	r := []rune(s)
	if len(r) > w-1 {
		r = r[:w-1]
	}
	return string(r) + "…"
}
