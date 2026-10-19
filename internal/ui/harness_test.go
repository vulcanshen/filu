package ui

import (
	"bytes"
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	terminu "github.com/vulcanshen/terminu-go"
)

// harness runs filu without a terminal: keys in, the screen out as text.
type harness struct {
	t    *testing.T
	a    *App
	root string       // the directory filu opened in
	home string       // $HOME for the test: the settings, the session and the trash are under it
	quit bool         // filu asked to quit
	clip bytes.Buffer // what was copied to the clipboard (OSC 52)
	// late are the messages of filu's background channels that a read the
	// harness stopped waiting for got after all, to be handled still.
	late chan tea.Msg
}

// newHarness is filu on a w × h screen over a fresh directory holding files
// (name → content; a name ending in / is a directory), with its settings and
// session in a temporary directory of their own.
func newHarness(t *testing.T, w, h int, files map[string]string) *harness {
	t.Helper()
	terminu.SetIconWidth(1, 1)
	home := t.TempDir()
	oldC, oldS, oldD := configPathOverride, statePathOverride, dataPathOverride
	configPathOverride = filepath.Join(home, "config", "config.yaml")
	statePathOverride = filepath.Join(home, "state", "state.yaml")
	dataPathOverride = filepath.Join(home, "data", "data.yaml")
	t.Cleanup(func() { configPathOverride, statePathOverride, dataPathOverride = oldC, oldS, oldD })
	t.Setenv("FILU__LAST_DIR_FILE", "")
	t.Setenv("HOME", home) // the trash, ~ in paths and Goto stay in here
	t.Setenv("XDG_CONFIG_HOME", "")
	oldClip := clipboardOut
	hs0 := &harness{late: make(chan tea.Msg, 256)}
	clipboardOut = &hs0.clip
	t.Cleanup(func() { clipboardOut = oldClip })
	resetGlobals()
	root := filepath.Join(home, "work")
	makeDir(t, root)
	for name, body := range files {
		p := filepath.Join(root, name)
		if strings.HasSuffix(name, "/") {
			makeDir(t, p)
			continue
		}
		makeDir(t, filepath.Dir(p))
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	a := NewApp(root, "")
	t.Cleanup(func() {
		if a.watcher != nil {
			a.watcher.Close()
		}
	})
	hs := hs0
	hs.t, hs.a, hs.root, hs.home = t, a, root, home
	hs.send(tea.WindowSizeMsg{Width: w, Height: h})
	return hs
}

// resetGlobals puts back what a test before may have changed.
func resetGlobals() {
	sortByDir = map[string][]sortRule{}
	finderCap, finderIgnoreDirs, openWithApps = defaultFinderCap, defaultIgnoreDirs, nil
	inputTabWidth, hideSecretsInPreview = defaultInputTabWidth, false
}

func makeDir(t *testing.T, p string) {
	t.Helper()
	if err := os.MkdirAll(p, 0o755); err != nil {
		t.Fatal(err)
	}
}

// send gives msg to filu and runs what it sets off, animations included,
// until nothing is left to run.
func (hs *harness) send(msg tea.Msg) {
	_, cmd := hs.a.t.Update(msg)
	hs.run(cmd)
}

// longTimer is how long a command may take before the harness stops waiting
// for it: an animation frame is 32ms; a toast's time, the channels filu reads
// in the background, are not waited for.
const longTimer = 150 * time.Millisecond

func (hs *harness) run(cmd tea.Cmd) {
	deadline := time.Now().Add(5 * time.Second)
	queue := []tea.Cmd{cmd}
	for len(queue) > 0 && time.Now().Before(deadline) {
		// What filu's goroutines have sent: finder results, task progress.
		select {
		case m := <-hs.a.finds:
			_, next := hs.a.t.Update(m)
			queue = append(queue, next)
		case m := <-hs.a.taskCh:
			_, next := hs.a.t.Update(m)
			queue = append(queue, next)
		case m := <-hs.late:
			_, next := hs.a.t.Update(m)
			queue = append(queue, next)
		default:
		}
		c := queue[0]
		queue = queue[1:]
		if c == nil {
			continue
		}
		done := make(chan tea.Msg, 1)
		go func() { done <- c() }()
		var msg tea.Msg
		select {
		case msg = <-done:
		case <-time.After(longTimer):
			// A read of a background channel goes on reading: what it gets is
			// handled later, not lost. A timer's message is let go.
			go func() {
				switch m := (<-done).(type) {
				case finderMsg, landMsg, watchMsg:
					hs.late <- m
				}
			}()
			continue
		}
		switch msg := msg.(type) {
		case nil:
		case tea.BatchMsg:
			queue = append(queue, msg...)
		case tea.QuitMsg:
			hs.quit = true
			return
		default:
			_, next := hs.a.t.Update(msg)
			queue = append(queue, next)
		}
	}
}

// keys presses each key: a name as terminu.KeyName writes it ("Enter", "Esc",
// "Space", "Tab", "←"), or text to type.
func (hs *harness) keys(keys ...string) {
	for _, k := range keys {
		hs.send(keyMsg(k))
	}
}

func keyMsg(k string) tea.KeyMsg {
	named := map[string]tea.KeyType{
		"Enter": tea.KeyEnter, "Esc": tea.KeyEsc, "Tab": tea.KeyTab, "Shift-Tab": tea.KeyShiftTab,
		"Space": tea.KeySpace, "Backspace": tea.KeyBackspace, "Delete": tea.KeyDelete,
		"←": tea.KeyLeft, "→": tea.KeyRight, "↑": tea.KeyUp, "↓": tea.KeyDown,
		"Home": tea.KeyHome, "End": tea.KeyEnd, "Ctrl-C": tea.KeyCtrlC,
	}
	if t, ok := named[k]; ok {
		if t == tea.KeySpace {
			return tea.KeyMsg{Type: t, Runes: []rune{' '}}
		}
		return tea.KeyMsg{Type: t}
	}
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(k)}
}

// drain runs what filu's background channels have for it: finder batches and
// task progress.
func (hs *harness) drain() {
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		select {
		case m := <-hs.a.finds:
			hs.send(m)
		case m := <-hs.a.taskCh:
			hs.send(m)
		case m := <-hs.late:
			hs.send(m)
		case <-time.After(300 * time.Millisecond):
			return
		}
	}
}

// frame is the screen as plain text.
func (hs *harness) frame() []string {
	var out []string
	for _, l := range hs.a.t.Frame() {
		out = append(out, terminu.StripStyles(l))
	}
	return out
}

// screen is the frame in one string, to search and to show on a failure.
func (hs *harness) screen() string { return strings.Join(hs.frame(), "\n") }

// has fails the test unless the screen shows every one of want.
func (hs *harness) has(want ...string) {
	hs.t.Helper()
	s := hs.screen()
	for _, w := range want {
		if !strings.Contains(s, w) {
			hs.t.Fatalf("the screen has no %q:\n%s", w, s)
		}
	}
}

// hasNot fails the test when the screen shows any of these.
func (hs *harness) hasNot(not ...string) {
	hs.t.Helper()
	s := hs.screen()
	for _, w := range not {
		if strings.Contains(s, w) {
			hs.t.Fatalf("the screen has %q:\n%s", w, s)
		}
	}
}

// copied is the text last copied to the clipboard.
func (hs *harness) copied() string {
	s := hs.clip.String()
	i := strings.LastIndex(s, ";")
	if i < 0 {
		return ""
	}
	s = strings.TrimRight(s[i+1:], "\a\x1b\\")
	b, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		hs.t.Fatalf("not an OSC 52 sequence: %q", hs.clip.String())
	}
	return string(b)
}

// restart ends filu as quitting does and starts it again on the same
// directory, settings and session.
func (hs *harness) restart() {
	hs.a.shutdown()
	w, h := hs.a.t.Size()
	hs.a = NewApp(hs.root, "")
	hs.send(tea.WindowSizeMsg{Width: w, Height: h})
}

// path is a file of the test's directory.
func (hs *harness) path(name string) string { return filepath.Join(hs.root, name) }

// exists reports a file of the test's directory.
func (hs *harness) exists(name string) bool {
	_, err := os.Lstat(hs.path(name))
	return err == nil
}

// cursor is the entry under [1]'s cursor.
func (hs *harness) cursor() string {
	it, _, _ := hs.a.item()
	return it.name
}

// to puts [1]'s cursor on name.
func (hs *harness) to(name string) {
	hs.t.Helper()
	if !hs.a.cur().focus(name) {
		hs.t.Fatalf("%s is not in the list", name)
	}
	hs.send(nil)
}
