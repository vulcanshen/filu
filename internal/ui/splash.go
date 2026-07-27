package ui

import (
	"math/rand"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/vulcanshen/filu/internal/version"
)

type splashTickMsg struct{}
type splashIdentityMsg struct{} // fires the name + tagline together
type splashHintMsg struct{}

// splashModel renders the filu logo as a hidden easter egg (the `V` key), a
// sibling of kbu's splash: the mark is revealed in staged passes, then the
// caption fades in. The u-family logo spells "fiL" in gold inside a navy "U".
type splashModel struct {
	active          bool
	pixelOrder      []int // reveal order: full background sheet (top-down), U frame (bottom-up), then the letters (shuffled)
	revealedCount   int
	bgCount         int  // end of stage 1 — indices [0:bgCount] are the background sheet (every cell)
	uEnd            int  // end of stage 2 — indices [bgCount:uEnd] are U-frame pixels
	pausedBg        bool // beat consumed at the background→U boundary
	pausedU         bool // beat consumed at the U→letters boundary
	identityVisible bool // "filu" line
	versionVisible  bool // the version line
	taglineVisible  bool // the tagline line
	hintVisible     bool // the Esc hint
}

func newSplashModel() splashModel { return splashModel{} }

func (m splashModel) isActive() bool { return m.active }

// show activates the splash and returns the first animation tick. Reveal phases:
// (1) background — a dark sheet, row-major top-to-bottom sweep; (2) beat;
// (3) U frame (navy), bottom-to-top so it rises from the base, over the sheet;
// (4) beat; (5) the letters (gold), shuffled, painted over the sheet; then a
// hold reveals the name + tagline together, and a final hold the Esc hint.
func (m *splashModel) show() tea.Cmd {
	m.active = true
	m.revealedCount = 0
	m.pausedBg = false
	m.pausedU = false
	m.identityVisible = false
	m.versionVisible = false
	m.taglineVisible = false
	m.hintVisible = false

	rows, cols := len(logoPixels), len(logoPixels[0])
	// Background pass covers EVERY cell so the sheet fills solid; the U and letter
	// passes come later and paint over it (overwrite, not gaps).
	bg := make([]int, 0, rows*cols)
	for r := 0; r < rows; r++ {
		for c := 0; c < cols; c++ {
			bg = append(bg, r*cols+c)
		}
	}
	// U frame collected bottom-to-top so it rises from the base.
	var frame []int
	for r := rows - 1; r >= 0; r-- {
		for c := 0; c < cols; c++ {
			if logoPixels[r][c] == 'U' {
				frame = append(frame, r*cols+c)
			}
		}
	}
	// Gold letters (F / I / L), shuffled.
	var letters []int
	for r := 0; r < rows; r++ {
		for c := 0; c < cols; c++ {
			if b := logoPixels[r][c]; b == 'F' || b == 'I' || b == 'L' {
				letters = append(letters, r*cols+c)
			}
		}
	}
	rand.Shuffle(len(letters), func(i, j int) { letters[i], letters[j] = letters[j], letters[i] })
	m.pixelOrder = append(append(bg, frame...), letters...)
	m.bgCount = len(bg)
	m.uEnd = len(bg) + len(frame)

	return tea.Tick(10*time.Millisecond, func(time.Time) tea.Msg { return splashTickMsg{} })
}

// filu logo — generated from docs/icon.svg (25×25, trimmed to 23 rows). The mark
// spells filu: F, I and L in gold sit inside the navy U frame (side rails + base)
// that wraps them. D = background sheet, U = navy frame, F/I/L = gold letters.
var logoPixels = [23]string{
	"DDDDDDDDDDDDDDDDDDDDDDDDD",
	"DDDDDDDDDDDDDDDDDDDDDDDDD",
	"DDUUUDFFFFFFFFFFFFFDUUUDD",
	"DDDUUDFFFFFFFFFFFFFDUUDDD",
	"DDDUUDFFDDDDDDDDDDDDUUDDD",
	"DDDUUDFFDDDDDDDDDDDDUUDDD",
	"DDDUUDFFFFFFFFFFFFFDUUDDD",
	"DDDUUDFFFFFFFFFFFFFDUUDDD",
	"DDDUUDFFDDDDDDDDDDDDUUDDD",
	"DDDUUDFFDIIDLLDDDDDDUUDDD",
	"DDDUUDFFDIIDLLDDDDDDUUDDD",
	"DDDUUDFFDDDDLLDDDDDDUUDDD",
	"DDDUUDFFDIIDLLDDDDDDUUDDD",
	"DDDUUDFFDIIDLLDDDDDDUUDDD",
	"DDDUUDFFDIIDLLDDDDDDUUDDD",
	"DDDUUDFFDIIDLLDDDDDDUUDDD",
	"DDDUUDFFDIIDLLLLLLLDUUDDD",
	"DDDUUDFFDIIDLLLLLLLDUUDDD",
	"DDDUUDDDDDDDDDDDDDDDUUDDD",
	"DDUUUUUUUUUUUUUUUUUUUUUDD",
	"DUUUUUUUUUUUUUUUUUUUUUUUD",
	"DDDDDDDDDDDDDDDDDDDDDDDDD",
	"DDDDDDDDDDDDDDDDDDDDDDDDD",
}

const (
	logoBg   = "#313244" // background sheet (catppuccin surface0)
	logoNavy = "#205090" // U frame — side rails + base (icon.svg)
	logoGold = "#f2b753" // F / I / L letters (icon.svg)

	// pixelGlyph is one logo pixel: the nf-fa-square Nerd Font glyph, coloured per
	// cell (same glyph everywhere, matching kbu's splash).
	pixelGlyph = ""
)

func (m splashModel) render(width, height int) string {
	if !m.active {
		return ""
	}

	// Colour each cell by the reveal pass that last touched it: the background
	// pass paints every cell; the U and letter passes come later in pixelOrder, so
	// they overwrite where they land (no gaps left).
	cols := len(logoPixels[0])
	cellColor := make([]string, len(logoPixels)*cols)
	for i := 0; i < m.revealedCount; i++ {
		idx := m.pixelOrder[i]
		switch {
		case i < m.bgCount:
			cellColor[idx] = logoBg
		case i < m.uEnd:
			cellColor[idx] = logoNavy
		default:
			cellColor[idx] = logoGold
		}
	}

	// A pixel is the nf-fa-square glyph in the cell's colour plus a space (two cells
	// per pixel), matching kbu's splash; unrevealed cells are two blanks.
	var logoLines []string
	for r := 0; r < len(logoPixels); r++ {
		var line strings.Builder
		for c := 0; c < cols; c++ {
			if color := cellColor[r*cols+c]; color != "" {
				line.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color(color)).Render(pixelGlyph + " "))
			} else {
				line.WriteString("  ")
			}
		}
		logoLines = append(logoLines, line.String())
	}
	logo := strings.Join(logoLines, "\n")

	// Caption space is always reserved so the logo doesn't shift when text appears.
	logoW := cols * 2
	name := lipgloss.NewStyle().Foreground(focusColor).Bold(true)
	line := lipgloss.NewStyle().Foreground(focusColor)
	dim := lipgloss.NewStyle().Foreground(dimColor)
	identityText, versionText, taglineText, hintText := " ", " ", " ", " "
	if m.identityVisible {
		identityText = name.Render("filu")
	}
	if m.versionVisible {
		versionText = line.Render(version.Display())
	}
	if m.taglineVisible {
		taglineText = line.Render("A single-pane terminal file manager")
	}
	if m.hintVisible {
		hintText = dim.Render("Press Esc to close")
	}
	caption := "\n\n" +
		lipgloss.PlaceHorizontal(logoW, lipgloss.Center, identityText) +
		"\n" +
		lipgloss.PlaceHorizontal(logoW, lipgloss.Center, versionText) +
		"\n" +
		lipgloss.PlaceHorizontal(logoW, lipgloss.Center, taglineText) +
		"\n\n" +
		lipgloss.PlaceHorizontal(logoW, lipgloss.Center, hintText)

	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, logo+caption)
}

// update handles key events and animation ticks while the splash is active.
func (m splashModel) update(msg tea.Msg) (splashModel, tea.Cmd) {
	if !m.active {
		return m, nil
	}
	switch msg.(type) {
	case tea.KeyMsg:
		// Any key dismisses the easter egg.
		m = splashModel{}
	case splashTickMsg:
		// Beat at the background→U boundary — a brief hold before the frame rises.
		if m.bgCount > 0 && m.revealedCount == m.bgCount && !m.pausedBg {
			m.pausedBg = true
			return m, tea.Tick(250*time.Millisecond, func(time.Time) tea.Msg { return splashTickMsg{} })
		}
		// Beat at the U→letters boundary — a brief hold before the letters shuffle in.
		if m.uEnd > m.bgCount && m.revealedCount == m.uEnd && !m.pausedU {
			m.pausedU = true
			return m, tea.Tick(250*time.Millisecond, func(time.Time) tea.Msg { return splashTickMsg{} })
		}
		if m.revealedCount < len(m.pixelOrder) {
			// Stage 1 (background sheet): one full row per tick — fast top-to-bottom fill.
			// Stage 2 (U frame): 3 px/tick, bottom-to-top. Stage 3 (letters): 2 px/tick.
			step := 2
			switch {
			case m.revealedCount < m.bgCount:
				step = len(logoPixels[0])
			case m.revealedCount < m.uEnd:
				step = 3
			}
			newCount := m.revealedCount + step
			// Clamp to each boundary so the beats fire cleanly.
			if m.revealedCount < m.bgCount && newCount > m.bgCount {
				newCount = m.bgCount
			} else if m.revealedCount < m.uEnd && newCount > m.uEnd {
				newCount = m.uEnd
			}
			if newCount > len(m.pixelOrder) {
				newCount = len(m.pixelOrder)
			}
			m.revealedCount = newCount
			return m, tea.Tick(10*time.Millisecond, func(time.Time) tea.Msg { return splashTickMsg{} })
		}
		// Pixels done — reveal the name + tagline after a brief hold.
		if !m.identityVisible {
			return m, tea.Tick(400*time.Millisecond, func(time.Time) tea.Msg { return splashIdentityMsg{} })
		}
	case splashIdentityMsg:
		m.identityVisible = true
		m.versionVisible = true
		m.taglineVisible = true
		return m, tea.Tick(500*time.Millisecond, func(time.Time) tea.Msg { return splashHintMsg{} })
	case splashHintMsg:
		m.hintVisible = true
	}
	return m, nil
}
