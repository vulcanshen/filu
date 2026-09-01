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
type splashIdentityMsg struct{} // fires the name + version + tagline + credit together
type splashHintMsg struct{}

// splashModel renders the filu logo as a hidden easter egg (the `V` key), a
// sibling of kbu's splash: the mark is revealed one letter at a time — F, i, L,
// then the U frame — and the caption fades in. The reveal spells filu, whatever
// order the letters sit in: the gold letters are placed L then i inside the navy
// "U", but they still light up F → i → L.
type splashModel struct {
	active          bool
	pixelOrder      []int    // reveal order across all stages: bg sheet, then F, i, L, then U
	orderColor      []string // colour for pixelOrder[i] (parallel), set by the stage that paints it
	stageEnds       []int    // cumulative pixel count at each stage's end; last == len(pixelOrder)
	stageStep       []int    // pixels revealed per tick within each stage
	beatsDone       int      // inter-stage holds already taken (stages always reveal in order)
	revealedCount   int
	identityVisible bool // "filu" line
	versionVisible  bool // the version line
	taglineVisible  bool // the tagline line
	creditVisible   bool // the "developed by" credit lines
	hintVisible     bool // the Esc hint
}

func newSplashModel() splashModel { return splashModel{} }

func (m splashModel) isActive() bool { return m.active }

// show activates the splash and returns the first animation tick. Reveal stages,
// each held apart by a beat: (1) background — a dark sheet, row-major top-to-bottom
// sweep; (2) F; (3) i; (4) L — the gold letters, one at a time, each shuffled in;
// (5) the U frame (navy), bottom-to-top so it rises from the base around them.
// Then a hold reveals the name + version + tagline + credit, and a final hold the
// Esc hint.
func (m *splashModel) show() tea.Cmd {
	m.active = true
	m.revealedCount = 0
	m.beatsDone = 0
	m.identityVisible = false
	m.versionVisible = false
	m.taglineVisible = false
	m.creditVisible = false
	m.hintVisible = false

	rows, cols := len(logoPixels), len(logoPixels[0])
	// Background pass covers EVERY cell so the sheet fills solid; the letter and
	// frame passes come later and paint over it (overwrite, not gaps).
	var bg []int
	for r := 0; r < rows; r++ {
		for c := 0; c < cols; c++ {
			bg = append(bg, r*cols+c)
		}
	}
	// letter returns one gold letter's pixels, shuffled so it scatters in.
	letter := func(b byte) []int {
		var px []int
		for r := 0; r < rows; r++ {
			for c := 0; c < cols; c++ {
				if logoPixels[r][c] == b {
					px = append(px, r*cols+c)
				}
			}
		}
		rand.Shuffle(len(px), func(i, j int) { px[i], px[j] = px[j], px[i] })
		return px
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

	// Assemble the stages in reveal order — bg, F, i, L, U — recording each stage's
	// colour (parallel to pixelOrder), cumulative end, and per-tick reveal step.
	m.pixelOrder, m.orderColor, m.stageEnds, m.stageStep = nil, nil, nil, nil
	addStage := func(px []int, color string, step int) {
		m.pixelOrder = append(m.pixelOrder, px...)
		for range px {
			m.orderColor = append(m.orderColor, color)
		}
		m.stageEnds = append(m.stageEnds, len(m.pixelOrder))
		m.stageStep = append(m.stageStep, step)
	}
	addStage(bg, logoBg, cols) // one full row per tick
	addStage(letter('F'), logoGold, 2)
	addStage(letter('I'), logoGold, 2)
	addStage(letter('L'), logoGold, 2)
	addStage(frame, logoNavy, 3) // rises bottom-to-top

	return tea.Tick(10*time.Millisecond, func(time.Time) tea.Msg { return splashTickMsg{} })
}

// filu logo — generated from docs/icon.svg (25×25, trimmed to 23 rows). The mark
// carries F, i and L in gold inside the navy U frame (side rails + base) that
// wraps them, with L placed before i. D = background sheet, U = navy frame,
// F/I/L = gold letters.
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
	"DDDUUDFFDLLDDDDDDIIDUUDDD",
	"DDDUUDFFDLLDDDDDDIIDUUDDD",
	"DDDUUDFFDLLDDDDDDDDDUUDDD",
	"DDDUUDFFDLLDDDDDDIIDUUDDD",
	"DDDUUDFFDLLDDDDDDIIDUUDDD",
	"DDDUUDFFDLLDDDDDDIIDUUDDD",
	"DDDUUDFFDLLDDDDDDIIDUUDDD",
	"DDDUUDFFDLLLLLLLDIIDUUDDD",
	"DDDUUDFFDLLLLLLLDIIDUUDDD",
	"DDDUUDDDDDDDDDDDDDDDUUDDD",
	"DDUUUUUUUUUUUUUUUUUUUUUDD",
	"DUUUUUUUUUUUUUUUUUUUUUUUD",
	"DDDDDDDDDDDDDDDDDDDDDDDDD",
	"DDDDDDDDDDDDDDDDDDDDDDDDD",
}

const (
	logoBg   = "#313244" // background sheet (catppuccin surface0)
	logoNavy = "#205090" // U frame — side rails + base (icon.svg)
	logoGold = "#f2b753" // F / i / L letters (icon.svg)

	// authorEmail is credited under the tagline, above the Esc hint.
	authorEmail = "vulcan.shen.2304@gmail.com"

	// pixelGlyph is one logo pixel: the nf-fa-square Nerd Font glyph, coloured per
	// cell (same glyph everywhere, matching kbu's splash).
	pixelGlyph = ""
)

func (m splashModel) render(width, height int) string {
	if !m.active {
		return ""
	}

	// Colour each revealed cell by the stage that painted it (orderColor is parallel
	// to pixelOrder). The background pass covers every cell; the letter and frame
	// passes come later in pixelOrder, so they overwrite where they land (no gaps).
	cols := len(logoPixels[0])
	cellColor := make([]string, len(logoPixels)*cols)
	for i := 0; i < m.revealedCount; i++ {
		cellColor[m.pixelOrder[i]] = m.orderColor[i]
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
	identityText, versionText, taglineText := " ", " ", " "
	creditText, emailText, hintText := " ", " ", " "
	if m.identityVisible {
		identityText = name.Render("filu")
	}
	if m.versionVisible {
		versionText = line.Render(version.Display())
	}
	if m.taglineVisible {
		taglineText = line.Render("A single-pane terminal file manager")
	}
	if m.creditVisible {
		creditText = dim.Render("developed by")
		emailText = dim.Render(authorEmail)
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
		lipgloss.PlaceHorizontal(logoW, lipgloss.Center, creditText) +
		"\n" +
		lipgloss.PlaceHorizontal(logoW, lipgloss.Center, emailText) +
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
		// Hold once at each stage boundary (bg→F→i→L→U) before the next begins.
		// Stages reveal in order, so beatsDone is also the next boundary to check.
		if m.beatsDone < len(m.stageEnds)-1 && m.revealedCount == m.stageEnds[m.beatsDone] {
			m.beatsDone++
			return m, tea.Tick(250*time.Millisecond, func(time.Time) tea.Msg { return splashTickMsg{} })
		}
		if m.revealedCount < len(m.pixelOrder) {
			// Advance by the current stage's step, clamped to that stage's end so the
			// boundary beats fire cleanly (bg fills a full row per tick, the letters
			// 2 px/tick, the frame rises 3 px/tick).
			stage := 0
			for stage < len(m.stageEnds)-1 && m.revealedCount >= m.stageEnds[stage] {
				stage++
			}
			newCount := m.revealedCount + m.stageStep[stage]
			if newCount > m.stageEnds[stage] {
				newCount = m.stageEnds[stage]
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
		m.creditVisible = true
		return m, tea.Tick(500*time.Millisecond, func(time.Time) tea.Msg { return splashHintMsg{} })
	case splashHintMsg:
		m.hintVisible = true
	}
	return m, nil
}
