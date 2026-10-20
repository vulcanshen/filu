package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	terminu "github.com/vulcanshen/terminu-go"
	"github.com/vulcanshen/terminu-go/tdptest"
)

// newapp is filu as main makes it, all but Run, on the home directory: what
// tdptest starts, in a sandbox of its own each time.
func newapp() *terminu.App {
	resetGlobals()
	return NewApp(os.Getenv("HOME"), "").t
}

// seedTricky puts files named after tdptest.Tricky in the home directory, the
// one filu opens on: names with control characters, escape sequences, wide
// and joined characters, as a disk may hold them. Names a file system takes
// no part of (a /, a NUL) are left out.
func seedTricky(home string) {
	for i, name := range tdptest.Tricky {
		if name == "" || name == "." || name == ".." || strings.ContainsAny(name, "/\x00") || len(name) > 200 {
			continue
		}
		_ = os.WriteFile(filepath.Join(home, name), []byte(name+"\n"), 0o644)
		if i%3 == 0 {
			_ = os.MkdirAll(filepath.Join(home, "dir "+name), 0o755)
		}
	}
}

// filu's first launch, walked by tdptest on many terminal sizes and icon
// widths: the Space menu, the global operation popup, the App Log and the key
// reference on every panel, a terminal under the least size and back. Every
// frame is checked for tdp L4 and for a control character reaching the
// terminal, the tricky names in [1], [2] and the location row among them.
func TestFirstLaunch(t *testing.T) {
	oldS, oldD := statePathOverride, dataPathOverride
	statePathOverride, dataPathOverride = "", "" // the sandbox's, as on a first launch
	t.Cleanup(func() { statePathOverride, dataPathOverride = oldS, oldD; resetGlobals() })
	tdptest.Check(t, tdptest.App{Name: "filu", New: newapp, Seed: seedTricky})
}
