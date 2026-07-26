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
