package ui

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"

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
// "go/pkg") matches that path segment. Users edit the list in ignore_dirs.yaml.
var defaultIgnoreDirs = []string{
	"node_modules", ".git", "Library", "OrbStack", "go/pkg", "vendor",
	"target", "__pycache__", "Pods", "bower_components",
	".venv", ".idea", ".vscode", ".gradle", ".m2", ".cargo", ".rustup",
	".terraform", ".cache", ".Trash",
}

// finderCap / finderIgnoreDirs are the live values, overridden from config.yaml
// at startup by loadConfig, and from the lists' files by loadLists.
var (
	finderCap        = defaultFinderCap
	finderIgnoreDirs = defaultIgnoreDirs
	openWithApps     []openWithApp
	// The family's shared settings (tdp E3), handed to terminu.
	inputTabWidth        = defaultInputTabWidth
	hideSecretsInPreview bool
)

// defaultInputTabWidth is input_tab_width's default (tdp E3).
const defaultInputTabWidth = 4

// fileConfig is the user-editable configuration (config.yaml, hand-edited) — kept
// separate from the auto-managed state.yaml so tuning knobs never mix with
// session state. IgnoreDirs and OpenWith are the lists versions before tdp
// v0.9.1 kept here: read only while they are still here, when moving them to
// their own files failed or found those there already (moveLists). IgnoreDirs
// is a pointer so an absent key falls back to the default while an explicit
// empty list means "exclude nothing".
type fileConfig struct {
	FinderCap            int           `yaml:"finder_cap"`
	IgnoreDirs           *[]string     `yaml:"ignore_dirs,omitempty"`
	OpenWith             []openWithApp `yaml:"open_with,omitempty"`
	InputTabWidth        int           `yaml:"input_tab_width"`
	HideSecretsInPreview bool          `yaml:"hide_secrets_in_preview"`
}

// openWithApp is one entry in open_with.yaml: a display name and the
// command filu runs as `<cmd> <path>` when it is picked from the [o]pen menu.
// The command may carry args (e.g. "code -n"); the path is appended as the last
// argument.
type openWithApp struct {
	Name string `yaml:"name"`
	Cmd  string `yaml:"cmd"`
}

var configPathOverride string // tests redirect config I/O here

// filuConfigDir resolves the directory that holds config.yaml and state.yaml:
// $XDG_CONFIG_HOME/filu, else ~/.config/filu, the same on every platform (tdp
// E3). After a move from the old place that failed, the old place.
func filuConfigDir() (string, bool) {
	if x := os.Getenv("XDG_CONFIG_HOME"); x != "" {
		return filepath.Join(x, "filu"), true
	}
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return "", false
	}
	if stuckInOld {
		if old := legacyConfigDir(home); old != "" {
			return old, true
		}
	}
	return filepath.Join(home, ".config", "filu"), true
}

// stuckInOld says moving the old settings directory failed: filu keeps using
// it until the user moves it (tdp E3).
var stuckInOld bool

// settingsMove is what moveOldSettings did.
type settingsMove struct {
	from, to string
	moved    []string // the names moved
	left     []string // the names the new place had already: left in the old one
	err      error    // why it could not move them: filu goes on using the old place
}

// moveOldSettings moves what is in the settings directory older versions used
// on macOS (Application Support) to tdp E3's place, one item at a time: what
// the new place has already stays where it is, for the user to put together,
// and the old directory goes once it is empty. When an item cannot be moved
// the ones moved go back, and filu keeps using the old place. A directory
// named by FILU__CONFIG or XDG_CONFIG_HOME is the user's choice, and is left
// alone.
func moveOldSettings() settingsMove {
	if configPathOverride != "" || os.Getenv("FILU__CONFIG") != "" || os.Getenv("XDG_CONFIG_HOME") != "" {
		return settingsMove{}
	}
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return settingsMove{}
	}
	m := settingsMove{from: legacyConfigDir(home), to: filepath.Join(home, ".config", "filu")}
	if m.from == "" || !isDir(m.from) {
		return settingsMove{}
	}
	entries, err := os.ReadDir(m.from)
	if err == nil {
		err = os.MkdirAll(m.to, 0o755)
	}
	var done []string
	for _, e := range entries {
		if err != nil {
			break
		}
		old, cur := filepath.Join(m.from, e.Name()), filepath.Join(m.to, e.Name())
		if _, serr := os.Lstat(cur); serr == nil {
			m.left = append(m.left, e.Name())
			continue
		}
		if err = os.Rename(old, cur); err == nil {
			done = append(done, e.Name())
		}
	}
	if err != nil {
		for _, name := range done { // back, so that filu reads one place
			_ = os.Rename(filepath.Join(m.to, name), filepath.Join(m.from, name))
		}
		stuckInOld = true
		return settingsMove{from: m.from, to: m.to, err: err}
	}
	m.moved = done
	if len(m.left) == 0 {
		_ = os.Remove(m.from) // empty now
	}
	return m
}

// legacyConfigDir is where filu kept its settings before tdp E3, when that was
// not ~/.config/filu: macOS's Application Support.
func legacyConfigDir(home string) string {
	if runtime.GOOS == "darwin" {
		return filepath.Join(home, "Library", "Application Support", "filu")
	}
	return ""
}

func isDir(p string) bool {
	fi, err := os.Stat(p)
	return err == nil && fi.IsDir()
}

func configFilePath() (string, bool) {
	if configPathOverride != "" {
		return configPathOverride, true
	}
	if d := os.Getenv("FILU__CONFIG"); d != "" { // a config directory (demo recordings / isolated runs, tdp E2, E3)
		return filepath.Join(d, "config.yaml"), true
	}
	dir, ok := filuConfigDir()
	if !ok {
		return "", false
	}
	return filepath.Join(dir, "config.yaml"), true
}

// loadConfig applies config.yaml over the defaults. On first run (no file) it
// drops a commented template so the knobs are discoverable and editable, made
// being where; a file that exists is never overwritten. An unreadable or
// malformed file leaves the defaults in place, and its error is returned for
// the app to ask about it (tdp E3: move it aside, or quit).
func loadConfig() (made string, err error) {
	path, ok := configFilePath()
	if !ok {
		return "", nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			if writeDefaultConfig(path) == nil { // first run → leave a template behind
				return path, nil
			}
			return "", nil
		}
		return "", err
	}
	var c fileConfig
	if err := yaml.Unmarshal(data, &c); err != nil {
		return "", err
	}
	if c.FinderCap > 0 {
		finderCap = c.FinderCap
	}
	if c.IgnoreDirs != nil {
		finderIgnoreDirs = *c.IgnoreDirs
	}
	openWithApps = c.OpenWith
	if c.InputTabWidth > 0 {
		inputTabWidth = c.InputTabWidth
	}
	hideSecretsInPreview = c.HideSecretsInPreview
	return "", nil
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
# input_tab_width: how many cells a tab takes in the preview and the viewer.
#
# hide_secrets_in_preview: true keeps private keys, and files only their owner
#   may read, out of the finders' previews.
#
# The last two are the same setting in every terminu family app.
#
# The apps of Open with and the directories the finders skip are lists you
# write: they are in open_with.yaml and ignore_dirs.yaml, next to filu's
# data.yaml (~/.filu).
#
`

func defaultConfigBytes() []byte {
	body, _ := yaml.Marshal(fileConfig{FinderCap: defaultFinderCap, InputTabWidth: defaultInputTabWidth})
	return append([]byte(configHeader), body...)
}

func writeDefaultConfig(path string) error {
	return writeFileIn(path, defaultConfigBytes())
}

// writeFileIn writes b to path, making its directory.
func writeFileIn(path string, b []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}
