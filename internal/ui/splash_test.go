package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

// The logo grid must be a clean rectangle — Render indexes it as rows×cols, so a
// ragged row would panic or misplace pixels.
func TestSplashLogoIsRectangular(t *testing.T) {
	want := len(logoPixels[0])
	for r, row := range logoPixels {
		if len(row) != want {
			t.Errorf("row %d width %d, want %d", r, len(row), want)
		}
	}
	if want != 25 {
		t.Errorf("logo is %d cols wide, want 25 (from icon.svg)", want)
	}
}

func TestSplashInitialState(t *testing.T) {
	m := newSplashModel()
	if m.isActive() {
		t.Error("a new splash must be inactive")
	}
	if out := m.render(80, 40); out != "" {
		t.Error("an inactive splash must render empty")
	}
}

func TestSplashShow(t *testing.T) {
	m := newSplashModel()
	cmd := m.show()
	if cmd == nil {
		t.Fatal("show() must return the first animation tick")
	}
	if !m.isActive() {
		t.Error("show() must activate the splash")
	}
	if len(m.pixelOrder) == 0 {
		t.Error("show() must populate pixelOrder")
	}
	if out := m.render(80, 40); out == "" {
		t.Error("an active splash must render non-empty")
	}
}

func TestSplashShowResetsRevealCount(t *testing.T) {
	m := newSplashModel()
	m.show()
	m, _ = m.update(splashTickMsg{})
	m.show() // re-trigger
	if m.revealedCount != 0 {
		t.Errorf("show() must reset revealedCount, got %d", m.revealedCount)
	}
}

func TestSplashTickRevealsPixels(t *testing.T) {
	m := newSplashModel()
	m.show()
	before := m.revealedCount
	m, cmd := m.update(splashTickMsg{})
	if m.revealedCount <= before {
		t.Errorf("a tick must reveal more pixels; before=%d after=%d", before, m.revealedCount)
	}
	if cmd == nil {
		t.Error("a tick mid-animation must schedule the next tick")
	}
}

func TestSplashTickDoesNotExceedTotal(t *testing.T) {
	m := newSplashModel()
	m.show()
	for i := 0; i < 2000 && m.revealedCount < len(m.pixelOrder); i++ {
		m, _ = m.update(splashTickMsg{})
	}
	if m.revealedCount != len(m.pixelOrder) {
		t.Errorf("revealedCount = %d, want %d (all pixels)", m.revealedCount, len(m.pixelOrder))
	}
}

func TestSplashBoundaryBeats(t *testing.T) {
	// Every stage boundary (bg→F, F→i, i→L, L→U) holds for exactly one tick before
	// the next stage advances. The U stage is last, so there is no beat after it.
	for k := 0; k < 4; k++ {
		m := newSplashModel()
		m.show()
		m.beatsDone = k
		m.revealedCount = m.stageEnds[k]
		before := m.revealedCount
		m, cmd := m.update(splashTickMsg{})
		if m.revealedCount != before || m.beatsDone != k+1 || cmd == nil {
			t.Errorf("stage-%d boundary must hold once (beatsDone=%d, count %d→%d, cmd nil=%v)",
				k, m.beatsDone, before, m.revealedCount, cmd == nil)
		}
		m, _ = m.update(splashTickMsg{})
		if m.revealedCount <= before {
			t.Errorf("stage %d: the second tick must advance past the boundary", k)
		}
	}
}

func TestSplashCaptionRevealsAfterAnimation(t *testing.T) {
	m := newSplashModel()
	m.show()
	m.revealedCount = len(m.pixelOrder) // fast-forward past the pixel reveal

	// A post-completion tick schedules the identity reveal without showing it yet.
	m, cmd := m.update(splashTickMsg{})
	if cmd == nil {
		t.Fatal("post-completion tick must schedule the identity reveal")
	}
	if m.identityVisible {
		t.Error("identity must wait for splashIdentityMsg")
	}

	// splashIdentityMsg reveals name + version + tagline + credit together and
	// schedules the hint.
	m, cmd = m.update(splashIdentityMsg{})
	if !m.identityVisible || !m.versionVisible || !m.taglineVisible || !m.creditVisible {
		t.Error("splashIdentityMsg must reveal the name, version, tagline and credit together")
	}
	if cmd == nil {
		t.Fatal("splashIdentityMsg must schedule the hint reveal")
	}
	if m.hintVisible {
		t.Error("hint must wait for splashHintMsg")
	}

	m, _ = m.update(splashHintMsg{})
	if !m.hintVisible {
		t.Error("splashHintMsg must reveal the hint")
	}
}

// The credit lines sit under the tagline and above the Esc hint. Asserted against
// the drawn frame, not the visibility flags, so reordering the caption is caught.
func TestSplashCreditRendersAboveHint(t *testing.T) {
	m := newSplashModel()
	m.show()
	m.revealedCount = len(m.pixelOrder) // fast-forward past the pixel reveal
	m, _ = m.update(splashIdentityMsg{})
	m, _ = m.update(splashHintMsg{})

	frame := ansi.Strip(m.render(80, 40))
	tagline := strings.Index(frame, "A single-pane terminal file manager")
	credit := strings.Index(frame, "developed by")
	email := strings.Index(frame, authorEmail)
	hint := strings.Index(frame, "Press Esc to close")
	if tagline < 0 || credit < 0 || email < 0 || hint < 0 {
		t.Fatalf("caption line missing (tagline=%d credit=%d email=%d hint=%d)", tagline, credit, email, hint)
	}
	if tagline > credit || credit > email || email > hint {
		t.Errorf("caption must read tagline→credit→email→hint, got offsets %d, %d, %d, %d",
			tagline, credit, email, hint)
	}
}

func TestSplashAnyKeyCloses(t *testing.T) {
	m := newSplashModel()
	m.show()
	m, _ = m.update(splashTickMsg{}) // reveal some pixels
	m, _ = m.update(tea.KeyMsg{Type: tea.KeyEscape})
	if m.isActive() {
		t.Error("a key must dismiss the splash")
	}
	if m.revealedCount != 0 || m.pixelOrder != nil {
		t.Error("dismissing must reset the splash state")
	}
}

// V from the main model opens the splash, and while it is up the whole screen
// becomes the logo (it owns the keyboard, so no panel key leaks through).
func TestSplashOpensWithV(t *testing.T) {
	m := minModel()
	m.width, m.height = 80, 40
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("V")})
	am := next.(AppModel)
	if !am.splash.isActive() {
		t.Fatal("V must open the splash")
	}
	if am.View() == "" {
		t.Error("an active splash must render a full screen")
	}
}

func TestSplashInactiveIgnoresMsgs(t *testing.T) {
	m := newSplashModel() // no show()
	m, cmd := m.update(splashTickMsg{})
	if m.isActive() || cmd != nil {
		t.Error("an inactive splash must ignore ticks")
	}
	m, cmd = m.update(tea.KeyMsg{Type: tea.KeyEscape})
	if m.isActive() || cmd != nil {
		t.Error("an inactive splash must ignore keys")
	}
}
