package ui

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// oldConfig is a config.yaml as versions before tdp v0.9.1 wrote it, edited:
// the lists in it, a comment of the user's on one.
const oldConfig = `# filu configuration — hand-edited.
#
# finder_cap: the most entries a finder will scan.
#
# ignore_dirs: directory names the finders skip entirely — dependency caches,
#   build output. Add or remove freely.
#
# open_with: apps for the [O]pen-with picker. Example:
#     open_with:
#       - name: VSCode
#         cmd: code
#
# input_tab_width: how many cells a tab takes in the preview and the viewer.
#
finder_cap: 7
# the dirs I never go to
ignore_dirs:
    - foo # mine
open_with:
    - name: Code
      cmd: code
input_tab_width: 2
`

// listsSetUp leaves config.yaml as body and the lists' files as files (name →
// content, none for a list not named), then starts filu again.
func listsSetUp(t *testing.T, hs *harness, body string, files map[string]string) {
	t.Helper()
	listsFiles(t, body, files)
	listsStart(hs)
}

func listsFiles(t *testing.T, body string, files map[string]string) {
	t.Helper()
	dir := filepath.Dir(dataPathOverride)
	for _, l := range userLists {
		if err := os.Remove(filepath.Join(dir, l.key+".yaml")); err != nil {
			t.Fatal(err)
		}
	}
	for name, b := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(b), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(configPathOverride, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func listsStart(hs *harness) {
	resetGlobals()
	hs.a = NewApp(hs.root, "")
	hs.send(tea.WindowSizeMsg{Width: 120, Height: 40})
	hs.run(hs.a.start())
}

func readText(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// The lists the user writes for filu are data (tdp E3): the first start
// writes them, commented, next to data.yaml, and config.yaml holds only
// filu's own settings; what the files say is what filu uses.
func TestE3ListsAreData(t *testing.T) {
	hs := newHarness(t, 120, 40, map[string]string{"a.txt": ""})
	dir := filepath.Dir(dataPathOverride)
	for _, l := range userLists {
		if b := readText(t, filepath.Join(dir, l.key+".yaml")); !strings.HasPrefix(b, l.header) {
			t.Fatalf("%s.yaml is not commented:\n%s", l.key, b)
		}
	}
	if c := readText(t, configPathOverride); strings.Contains(c, "\nopen_with:") || strings.Contains(c, "\nignore_dirs:") {
		t.Fatalf("config.yaml holds a list:\n%s", c)
	}
	if !slices.Equal(finderIgnoreDirs, defaultIgnoreDirs) || len(openWithApps) != 0 {
		t.Fatalf("a new file does not hold the default: %v %v", finderIgnoreDirs, openWithApps)
	}
	listsSetUp(t, hs, "finder_cap: 7\n", map[string]string{"open_with.yaml": "- name: Code\n  cmd: code\n", "ignore_dirs.yaml": "[]\n"})
	if len(finderIgnoreDirs) != 0 {
		t.Fatalf("ignore_dirs.yaml is not read: %v", finderIgnoreDirs)
	}
	hs.to("a.txt")
	hs.keys("O")
	hs.has("OPEN WITH", "Code", "runs code")
}

// A config.yaml holding the lists, as older versions wrote it: the first
// start moves each to its file next to data.yaml with the comments on it,
// takes what the template said about them out of config.yaml, leaves the
// rest as it was, and says from where to where (tdp E3, when the place
// changes). A second start moves nothing.
func TestE3ListsMove(t *testing.T) {
	hs := newHarness(t, 120, 40, nil)
	listsSetUp(t, hs, oldConfig, nil)
	hs.has("Moved open_with and ignore_dirs from ~/config/config.yaml to ~/data")
	c := readText(t, configPathOverride)
	for _, want := range []string{"# filu configuration", "# finder_cap: the most", "# input_tab_width: how many", "finder_cap: 7", "input_tab_width: 2"} {
		if !strings.Contains(c, want) {
			t.Fatalf("config.yaml lost %q:\n%s", want, c)
		}
	}
	for _, gone := range []string{"ignore_dirs", "open_with", "never go to", "VSCode"} {
		if strings.Contains(c, gone) {
			t.Fatalf("config.yaml still holds %q:\n%s", gone, c)
		}
	}
	dir := filepath.Dir(dataPathOverride)
	if b := readText(t, filepath.Join(dir, "ignore_dirs.yaml")); !strings.HasPrefix(b, ignoreDirsHeader) ||
		!strings.Contains(b, "# the dirs I never go to\n- foo # mine\n") {
		t.Fatalf("ignore_dirs.yaml:\n%s", b)
	}
	if b := readText(t, filepath.Join(dir, "open_with.yaml")); !strings.HasPrefix(b, openWithHeader) || !strings.Contains(b, "name: Code") {
		t.Fatalf("open_with.yaml:\n%s", b)
	}
	if !slices.Equal(finderIgnoreDirs, []string{"foo"}) || len(openWithApps) != 1 || openWithApps[0].Name != "Code" || finderCap != 7 {
		t.Fatalf("filu uses %v %v %d", finderIgnoreDirs, openWithApps, finderCap)
	}
	hs.keys("Esc", "!")
	hs.has("Moved open_with from ~/config/config.yaml to ~/data/open_with.yaml", "Moved ignore_dirs from")
	if m := moveLists(); len(m.moved) != 0 || len(m.left) != 0 || m.err != nil || readText(t, configPathOverride) != c {
		t.Fatalf("a second start moved %+v", m)
	}
}

// A list first in config.yaml goes without the comment on top of the file:
// that one stays on top.
func TestE3ListFirstInConfig(t *testing.T) {
	hs := newHarness(t, 120, 40, nil)
	listsSetUp(t, hs, "# my settings\nopen_with: []\nfinder_cap: 3\n", nil)
	if c := readText(t, configPathOverride); c != "# my settings\nfinder_cap: 3\n" {
		t.Fatalf("config.yaml:\n%s", c)
	}
	if b := readText(t, filepath.Join(filepath.Dir(dataPathOverride), "open_with.yaml")); b != openWithHeader+"[]\n" {
		t.Fatalf("open_with.yaml:\n%s", b)
	}
}

// A list the data directory has already stays in config.yaml, and its file
// is not touched: filu uses the file and shows both paths, to put the two
// together (tdp E3). The other list moves.
func TestE3ListInTwoPlaces(t *testing.T) {
	hs := newHarness(t, 120, 40, nil)
	listsSetUp(t, hs, oldConfig, map[string]string{"open_with.yaml": "- name: Vim\n  cmd: vim\n"})
	hs.has("FOUND IN TWO PLACES", "open_with in", "/config/config.yaml and in", "/data/open_with.yaml")
	hs.hasNot(`\n`)
	hs.has("┃ filu uses the ones in") // what to do starts a row of its own (tdp components-31)
	dir := filepath.Dir(dataPathOverride)
	if b := readText(t, filepath.Join(dir, "open_with.yaml")); b != "- name: Vim\n  cmd: vim\n" {
		t.Fatalf("open_with.yaml was written over:\n%s", b)
	}
	if c := readText(t, configPathOverride); !strings.Contains(c, "open_with:") || !strings.Contains(c, "name: Code") || strings.Contains(c, "ignore_dirs:") {
		t.Fatalf("config.yaml:\n%s", c)
	}
	if len(openWithApps) != 1 || openWithApps[0].Name != "Vim" || !slices.Equal(finderIgnoreDirs, []string{"foo"}) {
		t.Fatalf("filu uses %v %v", openWithApps, finderIgnoreDirs)
	}
}

// When config.yaml cannot be written without the lists, the files written go
// and filu reads the lists from config.yaml, showing both paths (tdp E3).
func TestE3ListsCannotMove(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root writes a read-only file too")
	}
	hs := newHarness(t, 120, 40, nil)
	listsFiles(t, oldConfig, nil)
	if err := os.Chmod(configPathOverride, 0o444); err != nil { // read, but not written
		t.Fatal(err)
	}
	listsStart(hs)
	hs.has("MOVING THE LISTS FAILED", "Cannot move open_with and ignore_dirs from", "goes on reading them from")
	if readText(t, configPathOverride) != oldConfig {
		t.Fatal("config.yaml was changed")
	}
	dir := filepath.Dir(dataPathOverride)
	for _, l := range userLists {
		if _, err := os.Stat(filepath.Join(dir, l.key+".yaml")); err == nil {
			t.Fatalf("%s.yaml is there: filu reads two places", l.key)
		}
	}
	if len(openWithApps) != 1 || openWithApps[0].Name != "Code" || !slices.Equal(finderIgnoreDirs, []string{"foo"}) {
		t.Fatalf("filu uses %v %v", openWithApps, finderIgnoreDirs)
	}
}
