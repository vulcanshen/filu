package ui

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// tdp F5: a shell that cannot start says why instead of silently not opening.
func TestF5ShellStartFailureIsReported(t *testing.T) {
	p := newPtyPopup()
	cmd := p.start(exec.Command(filepath.Join(t.TempDir(), "no-such-shell")), "Shell", t.TempDir(), 80, 24)
	msg, ok := cmd().(opFailedMsg)
	if !ok || !strings.HasPrefix(msg.text, "Cannot start shell: ") {
		t.Errorf("a failed start should report itself, got %#v", msg)
	}
	if p.isActive() {
		t.Error("a shell that never started must not leave the popup active")
	}
}

// tdp F5: a malformed config.yaml is reported (and the defaults stay); a missing
// one is the normal first run and is not an error.
func TestF5BadConfigIsReported(t *testing.T) {
	old := configPathOverride
	defer func() { configPathOverride = old }()

	bad := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(bad, []byte("finder_cap: [not a number\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	configPathOverride = bad
	if err := loadConfig(); err == nil {
		t.Error("a malformed config.yaml should come back as an error")
	}
	m := New(t.TempDir(), "")
	if !strings.HasPrefix(m.startupErr, "Cannot read config.yaml: ") || !strings.HasSuffix(m.startupErr, "(using the defaults)") {
		t.Errorf("New should keep the config error to show, got %q", m.startupErr)
	}

	configPathOverride = filepath.Join(t.TempDir(), "config.yaml") // not there yet
	if err := loadConfig(); err != nil {
		t.Errorf("a missing config is the first run, not an error: %v", err)
	}
}

// tdp F5: content search without ripgrep says so instead of showing "(no
// matches)" for every query; the chooser stays for a filename search.
func TestF5ContentSearchWithoutRipgrep(t *testing.T) {
	old := lookPath
	lookPath = func(string) (string, error) { return "", errors.New("not found") }
	defer func() { lookPath = old }()

	m := f4Model(t)
	m.openSearchMenu()
	m.searchMenu.anim.state = popupOpen
	m = press(t, m, runes("c"))
	if m.search.owns() {
		t.Error("without rg the content finder should not open")
	}
	if !m.toast.owns() || !strings.Contains(m.toast.message, "ripgrep (rg) is not installed") {
		t.Errorf("without rg the user should be told why, toast %q", m.toast.message)
	}
	if !m.searchMenu.owns() {
		t.Error("the chooser should stay, so a filename search is one key away")
	}
}

// tdp F5: a session that cannot be saved says so, from each place that saves.
func TestF5SaveFailureIsReported(t *testing.T) {
	old := statePathOverride
	statePathOverride = "/dev/null/filu/state.yaml" // under a file: the directory can't be made
	defer func() { statePathOverride = old }()

	for _, tc := range []struct {
		name string
		run  func(m *AppModel)
	}{
		{"persist", func(m *AppModel) { m.persist() }},
		{"sort reset", func(m *AppModel) { m.advanceSortFlow("r") }},
		{"task delete", func(m *AppModel) {
			m.focus, m.marksTab = panelMarks, 1
			m.tasks = []landTask{{id: 1, status: taskDone}}
			m.handleMarksKey("D")
		}},
		{"land finished", func(m *AppModel) {
			m.tasks = []landTask{{id: 7, status: taskRunning}}
			m.handleLandMsg(landMsg{taskID: 7, finished: true})
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := f4Model(t)
			tc.run(&m)
			if !m.toast.owns() || !strings.HasPrefix(m.toast.message, "Cannot save the session: ") {
				t.Errorf("a failed save should show a toast, got owns %v %q", m.toast.owns(), m.toast.message)
			}
		})
	}
}

// Init hands the startup problem to Update as a failure message, once.
func TestF5InitShowsStartupError(t *testing.T) {
	plain, _ := minModel().Init()().(tea.BatchMsg)
	m := minModel()
	m.startupErr = "Cannot read config.yaml: bad (using the defaults)"
	batch, _ := m.Init()().(tea.BatchMsg)
	// Only the extra command is run: the others are readers that block on their
	// channels, so running one would hang the test instead of failing it.
	if len(batch) != len(plain)+1 {
		t.Fatalf("Init should add one command for the startup error: %d vs %d", len(batch), len(plain))
	}
	msg, ok := batch[len(batch)-1]().(opFailedMsg)
	if !ok || msg.text != m.startupErr {
		t.Errorf("Init should report the startup error, got %#v", msg)
	}
}
