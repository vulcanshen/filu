//go:build darwin || linux

package ui

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func TestIsTextFile(t *testing.T) {
	dir := t.TempDir()
	txt := filepath.Join(dir, "a.txt")
	bin := filepath.Join(dir, "b.bin")
	if err := os.WriteFile(txt, []byte("hello\nworld\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(bin, []byte{0x00, 0x01, 0xff, 0xfe, 0x03}, 0o644); err != nil {
		t.Fatal(err)
	}
	if !isTextFile(txt) {
		t.Error("a UTF-8 file should be editable text")
	}
	if isTextFile(bin) {
		t.Error("a binary file should not be treated as text")
	}
}

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
