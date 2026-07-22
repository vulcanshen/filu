package ui

import (
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Popup open/close animation, ported from kbu (internal/ui/animation.go): a
// horizontal line grows from the centre, then the box reveals vertically; close
// reverses it. filu keeps the same feel so popups match the family.

type popupAnimState int

const (
	popupClosed popupAnimState = iota
	popupOpeningLine
	popupOpeningExpand
	popupOpen
	popupClosingCompress
	popupClosingLine
)

const (
	animFrameDuration = 20 * time.Millisecond
	animLineFrames    = 4
	animExpandFrames  = 4
)

// AnimTickMsg drives popup animations. Target identifies which popup.
type AnimTickMsg struct{ Target string }

// popupAnimator wraps a popup's rendering with open/close animations.
type popupAnimator struct {
	state  popupAnimState
	frame  int
	target string
	color  lipgloss.Color
}

func newPopupAnimator(target string, color lipgloss.Color) popupAnimator {
	return popupAnimator{state: popupClosed, target: target, color: color}
}

func (a popupAnimator) isActive() bool      { return a.state != popupClosed }
func (a popupAnimator) isInteractive() bool { return a.state == popupOpen }

// open begins the opening animation. No-op if already opening/open.
func (a *popupAnimator) open() tea.Cmd {
	if a.state == popupOpen || a.state == popupOpeningLine || a.state == popupOpeningExpand {
		return nil
	}
	a.state = popupOpeningLine
	a.frame = 0
	return a.tickCmd()
}

// close begins the closing animation. No-op if already closing/closed.
func (a *popupAnimator) close() tea.Cmd {
	if a.state == popupClosed || a.state == popupClosingCompress || a.state == popupClosingLine {
		return nil
	}
	a.state = popupClosingCompress
	a.frame = 0
	return a.tickCmd()
}

// tick advances the animation by one frame. Returns the next tick cmd or nil.
func (a *popupAnimator) tick() tea.Cmd {
	a.frame++
	switch a.state {
	case popupOpeningLine:
		if a.frame >= animLineFrames {
			a.state = popupOpeningExpand
			a.frame = 0
		}
		return a.tickCmd()
	case popupOpeningExpand:
		if a.frame >= animExpandFrames {
			a.state = popupOpen
			a.frame = 0
			return nil
		}
		return a.tickCmd()
	case popupClosingCompress:
		if a.frame >= animExpandFrames {
			a.state = popupClosingLine
			a.frame = 0
		}
		return a.tickCmd()
	case popupClosingLine:
		if a.frame >= animLineFrames {
			a.state = popupClosed
			a.frame = 0
			return nil
		}
		return a.tickCmd()
	}
	return nil
}

func (a popupAnimator) tickCmd() tea.Cmd {
	target := a.target
	return tea.Tick(animFrameDuration, func(time.Time) tea.Msg {
		return AnimTickMsg{Target: target}
	})
}

// renderFrame transforms a fully-rendered popup according to the current state:
// PopupOpen returns it unchanged, PopupClosed returns "".
func (a popupAnimator) renderFrame(full string) string {
	if a.state == popupClosed {
		return ""
	}
	if a.state == popupOpen {
		return full
	}
	width := lipgloss.Width(full)
	lines := strings.Split(full, "\n")
	height := len(lines)
	style := lipgloss.NewStyle().Foreground(a.color)

	clampW := func(p float64) int {
		w := int(float64(width) * p)
		return max(1, min(w, width))
	}
	clampH := func(p float64) int {
		h := int(float64(height) * p)
		return max(1, min(h, height))
	}

	switch a.state {
	case popupOpeningLine:
		return style.Render(strings.Repeat("─", clampW(float64(a.frame+1)/float64(animLineFrames))))
	case popupClosingLine:
		return style.Render(strings.Repeat("─", clampW(1.0-float64(a.frame+1)/float64(animLineFrames))))
	case popupOpeningExpand:
		return centerSlice(lines, clampH(float64(a.frame+1)/float64(animExpandFrames)))
	case popupClosingCompress:
		return centerSlice(lines, clampH(1.0-float64(a.frame+1)/float64(animExpandFrames)))
	}
	return full
}

// centerSlice returns n lines centred around the middle of lines.
func centerSlice(lines []string, n int) string {
	height := len(lines)
	if n >= height {
		return strings.Join(lines, "\n")
	}
	start := (height - n) / 2
	return strings.Join(lines[start:start+n], "\n")
}
