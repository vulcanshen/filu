package ui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// loadingFrames is the family loading icon (tdp D3, from webu): a circle filled
// one slice at a time, nf-md-circle_slice_1 to _8, one cell wide like the
// glyphs it stands beside.
var loadingFrames = [8]string{
	string(rune(0xf0a9e)), string(rune(0xf0a9f)), string(rune(0xf0aa0)), string(rune(0xf0aa1)),
	string(rune(0xf0aa2)), string(rune(0xf0aa3)), string(rune(0xf0aa4)), string(rune(0xf0aa5)),
}

// loadingStep is how long each frame shows: a turn every 720ms (tdp D3).
const loadingStep = 90 * time.Millisecond

// loadingNow is the clock the icon reads; tests pin it.
var loadingNow = time.Now

// loadingIcon is the frame for now. The clock picks it, not a counter, so every
// loading icon on screen turns together and a late tick cannot skip or repeat
// a frame (tdp D3).
func loadingIcon() string {
	return loadingFrames[(loadingNow().UnixNano()/int64(loadingStep))%int64(len(loadingFrames))]
}

// loadingTickMsg redraws the loading icons.
type loadingTickMsg struct{}

func loadingTick() tea.Cmd {
	return tea.Tick(loadingStep, func(time.Time) tea.Msg { return loadingTickMsg{} })
}

// anyLoading reports something on screen showing the loading icon: a running
// task in [3] Tasks, or a finder whose results are still coming in.
func (m *AppModel) anyLoading() bool {
	return m.anyRunning() || (m.search.isActive() && m.search.isLoading())
}

// keepLoading starts the icon tick when something is loading and none is in
// flight; the tick then re-arms itself only while something still loads (tdp D3).
func (m *AppModel) keepLoading() tea.Cmd {
	if m.loadingTicking || !m.anyLoading() {
		return nil
	}
	m.loadingTicking = true
	return loadingTick()
}
