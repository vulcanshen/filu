package ui

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// defaultFinderCap bounds how many entries a finder (Search / Find / Goto) scans.
// Goto is the one that hits it — it walks all of $HOME, which can hold hundreds
// of thousands of directories. Bigger = more directories reachable, but a slower
// in-memory fuzzy filter per keystroke; machines differ, so it is user-tunable.
const defaultFinderCap = 50000

// defaultIgnoreDirs are well-known tool-generated directories nobody navigates
// into — dependency caches, build output, IDE metadata, container data. Finders
// skip them so Goto over $HOME (and Search / Find over a project) isn't flooded.
// An entry matches a directory of that name anywhere; an entry with a slash (e.g.
// "go/pkg") matches that path segment. Users edit the list in config.yaml.
var defaultIgnoreDirs = []string{
	"node_modules", ".git", "Library", "OrbStack", "go/pkg", "vendor",
	"target", "__pycache__", "Pods", "bower_components",
	".venv", ".idea", ".vscode", ".gradle", ".m2", ".cargo", ".rustup",
	".terraform", ".cache", ".Trash",
}

// finderCap / finderIgnoreDirs are the live values, overridden from config.yaml
// at startup by loadConfig.
var (
	finderCap        = defaultFinderCap
	finderIgnoreDirs = defaultIgnoreDirs
	openWithApps     []openWithApp
)

// fileConfig is the user-editable configuration (config.yaml, hand-edited) — kept
// separate from the auto-managed state.yaml so tuning knobs never mix with
// session state. IgnoreDirs is a pointer so an absent key falls back to the
// default while an explicit empty list means "exclude nothing".
type fileConfig struct {
	FinderCap  int           `yaml:"finder_cap"`
	IgnoreDirs *[]string     `yaml:"ignore_dirs"`
	OpenWith   []openWithApp `yaml:"open_with"`
}

// openWithApp is one entry in config.yaml's open_with: a display name and the
// command filu runs as `<cmd> <path>` when it is picked from the [o]pen menu.
// The command may carry args (e.g. "code -n"); the path is appended as the last
// argument.
type openWithApp struct {
	Name string `yaml:"name"`
	Cmd  string `yaml:"cmd"`
}

var configPathOverride string // tests redirect config I/O here

func configFilePath() (string, bool) {
	if configPathOverride != "" {
		return configPathOverride, true
	}
	if p := os.Getenv("FILU_CONFIG"); p != "" { // redirect config I/O (demo recordings / isolated runs)
		return p, true
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", false
	}
	return filepath.Join(dir, "filu", "config.yaml"), true
}

// loadConfig applies config.yaml over the defaults. On first run (no file) it
// drops a commented template so the knobs are discoverable and editable; a file
// that exists is never overwritten. It never fails startup — any I/O or parse
// error just leaves the defaults in place.
func loadConfig() {
	path, ok := configFilePath()
	if !ok {
		return
	}
	data, err := os.ReadFile(path)
	if err != nil {
		writeDefaultConfig(path) // first run / unreadable → leave a template behind
		return
	}
	var c fileConfig
	if yaml.Unmarshal(data, &c) != nil {
		return
	}
	if c.FinderCap > 0 {
		finderCap = c.FinderCap
	}
	if c.IgnoreDirs != nil {
		finderIgnoreDirs = *c.IgnoreDirs
	}
	openWithApps = c.OpenWith
}

// configHeader is the top-of-file comment; the keys themselves are marshalled
// from the defaults below it, so the template can never drift from the code.
const configHeader = `# filu configuration — hand-edited, and separate from the auto-managed state.yaml.
#
# finder_cap: the most entries a finder (Search / Find / Goto) will scan. Goto
#   walks all of $HOME, so this is what bounds it. Higher = more directories
#   reachable, but a slower fuzzy filter per keystroke on a big home. Lower it if
#   Goto feels laggy, raise it if you want more reach.
#
# ignore_dirs: directory names the finders skip entirely — dependency caches,
#   build output, IDE metadata, container data: nowhere you'd cd to navigate. An
#   entry matches that name anywhere in the tree; an entry with a slash (e.g.
#   go/pkg) matches that path. Add or remove freely.
#
# open_with: apps for the [O]pen-with picker (press O on a file or directory;
#   plain o just opens with the OS default). Each entry has a name and a command;
#   filu runs "<cmd> <path>" when you pick it. The picker always offers "Default"
#   (the OS default app) first, then these — handy for opening a folder in your
#   IDE. Example (edit to the tools you have):
#     open_with:
#       - name: VSCode
#         cmd: code
#       - name: IntelliJ IDEA
#         cmd: idea
#
`

func defaultConfigBytes() []byte {
	body, _ := yaml.Marshal(fileConfig{FinderCap: defaultFinderCap, IgnoreDirs: &defaultIgnoreDirs, OpenWith: []openWithApp{}})
	return append([]byte(configHeader), body...)
}

func writeDefaultConfig(path string) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return
	}
	_ = os.WriteFile(path, defaultConfigBytes(), 0o644)
}
