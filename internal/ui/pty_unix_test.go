//go:build darwin || linux

package ui

import (
	"bytes"
	"os/exec"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

func TestBuildEditorCmd(t *testing.T) {
	t.Setenv("VISUAL", "")
	t.Setenv("EDITOR", "nvim -p")
	t.Setenv("TERM_PROGRAM", "iTerm.app")

	c := buildEditorCmd("/tmp/x.go")
	if len(c.Args) != 3 || c.Args[0] != "nvim" || c.Args[1] != "-p" || c.Args[2] != "/tmp/x.go" {
		t.Errorf("args = %v, want [nvim -p /tmp/x.go]", c.Args)
	}
	var hasTerm, hasProg bool
	for _, kv := range c.Env {
		if kv == "TERM=xterm-256color" {
			hasTerm = true
		}
		if strings.HasPrefix(kv, "TERM_PROGRAM=") {
			hasProg = true
		}
	}
	if !hasTerm {
		t.Error("env should set TERM=xterm-256color for vt10x")
	}
	if hasProg {
		t.Error("env should strip TERM_PROGRAM")
	}
}

func TestBuildShellCmd(t *testing.T) {
	t.Setenv("TERM_PROGRAM", "iTerm.app")

	t.Setenv("SHELL", "/bin/zsh")
	c := buildShellCmd()
	if len(c.Args) != 1 || c.Args[0] != "/bin/zsh" {
		t.Errorf("args = %v, want [/bin/zsh]", c.Args)
	}
	var hasTerm, hasProg bool
	for _, kv := range c.Env {
		if kv == "TERM=xterm-256color" {
			hasTerm = true
		}
		if strings.HasPrefix(kv, "TERM_PROGRAM=") {
			hasProg = true
		}
	}
	if !hasTerm {
		t.Error("env should set TERM=xterm-256color for vt10x")
	}
	if hasProg {
		t.Error("env should strip TERM_PROGRAM")
	}

	t.Setenv("SHELL", "") // no $SHELL → /bin/sh
	if c := buildShellCmd(); len(c.Args) != 1 || c.Args[0] != "/bin/sh" {
		t.Errorf("no SHELL should fall back to /bin/sh, got %v", c.Args)
	}
}

func TestPtyKeyBytes(t *testing.T) {
	cases := []struct {
		msg  tea.KeyMsg
		app  bool
		want []byte
	}{
		{tea.KeyMsg{Type: tea.KeyEnter}, false, []byte{'\r'}},
		{tea.KeyMsg{Type: tea.KeyUp}, false, []byte{'\x1b', '[', 'A'}},
		{tea.KeyMsg{Type: tea.KeyUp}, true, []byte{'\x1b', 'O', 'A'}}, // application-cursor
		{tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")}, false, []byte("a")},
		{tea.KeyMsg{Type: tea.KeyCtrlC}, false, []byte{'\x03'}},
		{tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("f"), Alt: true}, false, []byte{'\x1b', 'f'}},
	}
	for _, c := range cases {
		if got := ptyKeyBytes(c.msg, c.app); !bytes.Equal(got, c.want) {
			t.Errorf("ptyKeyBytes(%+v, app=%v) = %v, want %v", c.msg, c.app, got, c.want)
		}
	}
}

// TestPtyPopupBorderAligns guards the bordered box: every row — the titled top,
// the content rows, and the hint bottom — must be the same display width, or the
// right edge steps in and out (the long "Shell: <path>" title exposed a top-row
// off-by-one that short editor titles hid).
func TestPtyPopupBorderAligns(t *testing.T) {
	p := newPtyPopup()
	p.start(exec.Command("true"), "Shell: ~/Documents/sideproj/kbu", "/tmp", 100, 30)
	defer p.stop()
	p.anim.state = popupOpen // settle the open animation so renderFrame returns the box as-is

	lines := strings.Split(strings.TrimRight(p.renderPopup(), "\n"), "\n")
	if len(lines) < 3 {
		t.Fatalf("expected a bordered box, got %d lines", len(lines))
	}
	want := lipgloss.Width(lines[0])
	for i, ln := range lines {
		if w := lipgloss.Width(ln); w != want {
			t.Errorf("row %d width %d != %d (border misaligned):\n%q", i, w, want, ln)
		}
	}

	// Full width, and tall enough that pinned at row ptyChromeRows it reaches the
	// bottom: height == hostH - chrome (so header + status stay visible, footer is
	// covered).
	if want != 100 {
		t.Errorf("popup width = %d, want full host width 100", want)
	}
	if got := len(lines); got != 30-ptyChromeRows {
		t.Errorf("popup height = %d rows, want %d (hostH - chrome)", got, 30-ptyChromeRows)
	}
}

// TestPtyStartsInDir guards that the PTY process is rooted at the directory it
// was started with — a shell has no path argument, so without cmd.Dir it would
// open in filu's own cwd (the launch dir) instead of the active tab's directory.
func TestPtyStartsInDir(t *testing.T) {
	dir := t.TempDir()
	p := newPtyPopup()
	p.start(exec.Command("true"), "Shell", dir, 100, 30)
	defer p.stop()
	if p.cmd.Dir != dir {
		t.Errorf("cmd.Dir = %q, want the tab's dir %q", p.cmd.Dir, dir)
	}
}

// TestPtyStartStopNoPanic hammers the start→stop race: stop() can nil the shared
// handles before the readLoop goroutine reads them, and cmd.Wait on a nil *Cmd
// panics (which crashes the whole test binary from a goroutine). The nil-guard in
// readLoop must keep this quiet over many rounds.
func TestPtyStartStopNoPanic(t *testing.T) {
	for range 50 {
		p := newPtyPopup()
		p.start(exec.Command("true"), "x", t.TempDir(), 80, 24)
		p.stop()
	}
	time.Sleep(150 * time.Millisecond) // let any lingering readLoop goroutines finish
}

func TestPtyLifecycle(t *testing.T) {
	p := newPtyPopup()
	p.start(exec.Command("true"), "test", "/tmp", 80, 24) // exits immediately
	if !p.isActive() {
		t.Fatal("pty should be active after start")
	}

	deadline := time.Now().Add(3 * time.Second)
	for !p.done.Load() && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if !p.done.Load() {
		t.Fatal("the subprocess should have exited and set done")
	}

	p.update(ptyTickMsg{}) // a tick after done starts the two-phase teardown
	if !p.stopPending {
		t.Error("a tick after done should set stopPending")
	}
	p.anim.state = popupClosed               // simulate the close animation finishing
	p.handleTick(AnimTickMsg{Target: "pty"}) // runs the deferred stop
	if p.isActive() {
		t.Error("the pty should be stopped once the close animation settles")
	}
}

// ptyApp is an AppModel with a live shell-like process in the PTY popup.
func ptyApp(t *testing.T) AppModel {
	t.Helper()
	m := minModel()
	m.pty.start(exec.Command("sleep", "30"), "Shell", "/tmp", 80, 24)
	t.Cleanup(m.pty.stop)
	m.pty.anim.state = popupOpen
	return m
}

var altEsc = tea.KeyMsg{Type: tea.KeyEsc, Alt: true}

// waitDone waits for the PTY's process to end and reports whether it did.
func waitDone(p *ptyPopup, within time.Duration) bool {
	deadline := time.Now().Add(within)
	for !p.done.Load() && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	return p.done.Load()
}

// tdp K10, D5: Alt-Esc is the app key inside the PTY, and it asks first — two
// quick Esc presses in vim send the same bytes. The shell keeps running until
// the confirm is accepted.
func TestPtyAltEscAsksFirst(t *testing.T) {
	m := press(t, ptyApp(t), altEsc)
	if !m.confirm.owns() || m.confirmAction != confirmEndShell {
		t.Fatalf("Alt-Esc should open the end-shell confirm (owns %v, action %v)", m.confirm.owns(), m.confirmAction)
	}
	if m.confirm.verb != "end" || !strings.Contains(m.confirm.message, "/tmp") {
		t.Errorf("the confirm should name the shell's directory and end it: %q, verb %q", m.confirm.message, m.confirm.verb)
	}
	if m.pty.stopPending || waitDone(m.pty, 100*time.Millisecond) {
		t.Fatal("Alt-Esc alone must not end the shell")
	}
}

// tdp D5: Enter on the confirm ends the shell and closes the popup.
func TestPtyEndShellConfirmEnds(t *testing.T) {
	m := press(t, press(t, ptyApp(t), altEsc), tea.KeyMsg{Type: tea.KeyEnter})
	if !m.pty.stopPending || m.pty.anim.owns() || m.confirm.owns() {
		t.Fatalf("Enter should close the confirm and the shell popup: stopPending %v, pty owns %v, confirm owns %v", m.pty.stopPending, m.pty.anim.owns(), m.confirm.owns())
	}
	if !waitDone(m.pty, 3*time.Second) {
		t.Error("accepting should end the shell process, not leave it running")
	}
	if m = press(t, m, altEsc); m.confirm.owns() {
		t.Error("Alt-Esc while the shell is closing should not ask again")
	}
}

// tdp D5, F4: Esc on the confirm goes back to the shell, which keeps running and
// takes the keys again.
func TestPtyEndShellConfirmEscReturns(t *testing.T) {
	m := press(t, press(t, ptyApp(t), altEsc), tea.KeyMsg{Type: tea.KeyEsc})
	if m.confirm.owns() || !m.pty.isActive() || m.pty.stopPending || waitDone(m.pty, 100*time.Millisecond) {
		t.Fatalf("Esc should close only the confirm: confirm owns %v, pty active %v, stopPending %v", m.confirm.owns(), m.pty.isActive(), m.pty.stopPending)
	}
	m = press(t, m, runes("q"))
	if m.quitMenu.owns() {
		t.Error("back in the shell, q should go to it again")
	}
}

// tdp F4, K6, K9: over the confirm, the keys are the confirm's: ? is its key
// reference, q the leave flow; with no box over the shell, both went to it.
func TestPtyEndShellConfirmTakesPopupKeys(t *testing.T) {
	m := ptyApp(t)
	m.help = newHelpPopup()
	m = press(t, press(t, m, altEsc), runes("?"))
	if !m.help.owns() || m.help.title != "Confirm keys" {
		t.Errorf("? over the end-shell confirm should open its key reference, got owns %v %q", m.help.owns(), m.help.title)
	}
	m = press(t, press(t, ptyApp(t), altEsc), runes("q"))
	if !m.quitMenu.owns() {
		t.Error("q over the end-shell confirm should open the quit picker")
	}
}

// tdp F3, K10: a toast's Esc comes first only when a box is over the shell;
// otherwise Esc is the shell's.
func TestPtyToastEscOrder(t *testing.T) {
	m := ptyApp(t)
	m.toast = newToast()
	m.toast.show("x")
	m.toast.anim.state = popupOpen
	if m = press(t, m, tea.KeyMsg{Type: tea.KeyEsc}); !m.toast.owns() {
		t.Error("with nothing over the shell, Esc should go to it, not close the toast")
	}
	m = press(t, press(t, m, altEsc), tea.KeyMsg{Type: tea.KeyEsc})
	if m.toast.owns() || !m.confirm.owns() {
		t.Errorf("over the confirm, the first Esc should close the toast only: toast %v, confirm %v", m.toast.owns(), m.confirm.owns())
	}
}

// tdp T1: the shell ending by itself takes the confirm (and its key
// reference) with it — there is nothing left to end.
func TestPtyEndShellConfirmGoesWithTheShell(t *testing.T) {
	m := ptyApp(t)
	m.help = newHelpPopup()
	m = press(t, press(t, m, altEsc), runes("?"))
	model, _ := m.Update(ptyExitMsg{dir: "/tmp"})
	m = model.(AppModel)
	if m.confirm.owns() || m.help.owns() {
		t.Errorf("the confirm and its key reference should close with the shell: confirm %v, help %v", m.confirm.owns(), m.help.owns())
	}
}

// ptyScreen is a sized, truecolor app with the shell open over the panels.
func ptyScreen(t *testing.T) AppModel {
	t.Helper()
	m := f8Model(t)
	m.pty.start(exec.Command("sleep", "30"), "Shell", "/tmp", m.width, m.height)
	t.Cleanup(m.pty.stop)
	m.pty.anim.state = popupOpen
	return m
}

// tdp F4, F8, D2: the confirm is drawn over the shell in the next layer's
// colour, and the shell beneath it dims, its frame keeping its layer colour.
func TestPtyEndShellConfirmDrawsOverTheShell(t *testing.T) {
	m := ptyScreen(t)
	lines := strings.Split(m.View(), "\n")
	// The shell's frame on its own: Lavenphire25 (#A4C0FA, layer 1) at full strength.
	if got, want := cellFG(lines[0])[0], [3]int{164, 192, 250}; !near(got, want) {
		t.Fatalf("the shell frame is %v, want layer 1 at full strength %v", got, want)
	}

	m = press(t, m, altEsc)
	out := m.View()
	if !strings.Contains(ansi.Strip(out), "End the shell in /tmp?") {
		t.Fatalf("the confirm should be drawn over the shell:\n%s", ansi.Strip(out))
	}
	lines = strings.Split(out, "\n")
	// Lavenphire25 faded 55% into the base #1e1e2e (written out, not dimRGB).
	if got, want := cellFG(lines[0])[0], [3]int{90, 103, 138}; !near(got, want) {
		t.Errorf("the shell frame under the confirm is %v, want its layer colour dimmed %v", got, want)
	}
	box := m.confirm.renderFull()
	y0, x0 := centred(m.height, boxRows(box)), centred(m.width, boxWidth(box))
	// Lavenphire50 (#94C3F5): the confirm is layer 2, the shell layer 1.
	if got, want := cellFG(lines[y0])[x0], [3]int{148, 195, 245}; !near(got, want) {
		t.Errorf("the confirm border is %v, want layer 2 %v", got, want)
	}
}

// tdp T1, F8: accepting the Shell confirm clears the stack at once — the shell
// sits under the stack, so a menu still collapsing would draw over it and dim it.
func TestPtyShellStartDropsTheStack(t *testing.T) {
	t.Setenv("SHELL", "/bin/cat") // stays up on the PTY without loading a profile
	m := f4Model(t)
	m.width, m.height = 100, 30
	m = press(t, press(t, m, runes(" ")), runes("s"))
	if m.confirmAction != confirmShell || !m.spaceMenu.owns() {
		t.Fatalf("s in the Space menu should open the Shell confirm over it (action %v)", m.confirmAction)
	}
	m = press(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	t.Cleanup(m.pty.stop)
	if !m.pty.isActive() {
		t.Fatal("Enter should start the shell")
	}
	if m.spaceMenu.isActive() || m.confirm.isActive() {
		t.Errorf("the Space menu and the confirm should be gone at once: menu %v, confirm %v", m.spaceMenu.isActive(), m.confirm.isActive())
	}
}

// tdp K10: every other key, plain Esc and Ctrl-C included, belongs to the shell.
func TestPtyKeysBelongToShell(t *testing.T) {
	for _, k := range []tea.KeyMsg{{Type: tea.KeyEsc}, {Type: tea.KeyCtrlC}, {Type: tea.KeyRunes, Runes: []rune("q")}} {
		m := ptyApp(t)
		model, _ := m.Update(k)
		got := model.(AppModel)
		if got.pty.stopPending || got.quitMenu.owns() {
			t.Errorf("%q inside the PTY should go to the shell, not close it or open the quit picker", k.String())
		}
	}
}

// tdp K10 / M1: the exit key is on show in the PTY frame the whole time.
func TestPtyFrameShowsExitKey(t *testing.T) {
	m := ptyApp(t)
	if out := m.pty.renderPopup(); !strings.Contains(out, "Alt+Esc") {
		t.Errorf("the PTY frame should name the exit key:\n%s", out)
	}
}
