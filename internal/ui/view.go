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

	listBody := m.list.view(midW-2, midH-3, m.focus == panelList)
	list := m.panelBox(panelList, "[2] "+pathBase(m.list.dir), midW, midH, listBody)

	middle := list
	if leftW > 0 && rightW > 0 {
		pinH := midH * 2 / 3
		left := lipgloss.JoinVertical(
			lipgloss.Left,
			m.panelBox(panelPin, "[1] pin", leftW, pinH, "Places\nPinned"),
			m.panelBox(panelCarry, "[4] carry", leftW, midH-pinH, "empty"),
		)
		right := m.panelBox(panelDetail, "[3] Preview|Info", rightW, midH, "(preview)")
		middle = lipgloss.JoinHorizontal(lipgloss.Top, left, list, right)
	}

	return lipgloss.JoinVertical(lipgloss.Left, m.headerBar(w), middle, m.footerBar(w))
}

// panelBox renders one bordered panel; focused = double border + blue, else rounded + dim.
func (m AppModel) panelBox(id panelID, title string, w, h int, body string) string {
	focused := m.focus == id
	border, color := lipgloss.RoundedBorder(), dimColor
	if focused {
		border, color = lipgloss.DoubleBorder(), focusColor
	}
	titleLine := lipgloss.NewStyle().Foreground(color).Bold(focused).Render(title)
	content := titleLine + "\n" + body
	return lipgloss.NewStyle().
		Border(border).
		BorderForeground(color).
		Width(w - 2).
		Height(h - 2).
		Render(content)
}

func (m AppModel) headerBar(w int) string {
	return lipgloss.NewStyle().Width(w).Foreground(pathColor).
		Render(truncate(" "+breadcrumb(m.list.dir), w))
}

func (m AppModel) footerBar(w int) string {
	return lipgloss.NewStyle().Width(w).Foreground(dimColor).
		Render(truncate(" Space 動作   ? 選單   Tab/1-4 切面板   q 離開", w))
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
