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

// filu's first launch, walked by tdptest on many terminal sizes and icon
// widths: the Space menu, the global operation popup, the App Log and the key
// reference on every panel, a terminal under the least size and back. Every
// frame is checked for tdp L4, for a control character reaching the terminal
// and for a colour code of filu's own drawn as text.
func TestFirstLaunch(t *testing.T) {
	oldS, oldD := statePathOverride, dataPathOverride
	statePathOverride, dataPathOverride = "", "" // the sandbox's, as on a first launch
	t.Cleanup(func() { statePathOverride, dataPathOverride = oldS, oldD; resetGlobals() })
	tdptest.Check(t, tdptest.App{Name: "filu", New: newapp})
}

// The same first launch on a home directory holding the tricky names: they
// are in [1], [2] and the location row on every size and icon width. A name
// may hold a colour code, drawn by its name as it should be (Tricky).
func TestFirstLaunchOnTrickyNames(t *testing.T) {
	oldS, oldD := statePathOverride, dataPathOverride
	statePathOverride, dataPathOverride = "", ""
	t.Cleanup(func() { statePathOverride, dataPathOverride = oldS, oldD; resetGlobals() })
	tdptest.Check(t, tdptest.App{Name: "filu", New: newapp, Tricky: true, Seed: func(home string) {
		for name, body := range trickyFiles() {
			p := filepath.Join(home, name)
			if strings.HasSuffix(name, "/") {
				_ = os.MkdirAll(p, 0o755)
				continue
			}
			_ = os.WriteFile(p, []byte(body), 0o644)
		}
	}})
}

// trickyFiles are files and directories named after tdptest.Tricky: names
// with control characters, escape sequences, wide and joined characters, as a
// disk may hold them. Names a file system takes no part of (a /, a NUL) are
// left out.
func trickyFiles() map[string]string {
	files := map[string]string{}
	for i, name := range tdptest.Tricky {
		if name == "" || name == "." || name == ".." || strings.ContainsAny(name, "/\x00") || len(name) > 200 {
			continue
		}
		files[name] = name + "\n"
		if i%3 == 0 {
			files["dir "+name+"/"] = ""
		}
	}
	return files
}

// Every tricky name in [1] under the cursor in turn, previewed in [2], its
// facts opened, marked into [3], searched for: tdptest checks each frame for
// tdp L4 and for a control character reaching the terminal
// (tdp components.5).
func TestTrickyNames(t *testing.T) {
	tdptest.EachIconWidth(t, func(t *testing.T) {
		hs := newHarness(t, 120, 40, trickyFiles())
		d := tdptest.Start(t, hs.a.t, 120, 40)
		d.AllowLineBreaks = true // the names hold line breaks on purpose, drawn by name in red
		for range len(hs.a.cur().items) {
			d.Press("j")
			if it, _, ok := hs.a.item(); ok && !it.isDir {
				d.Press("Enter", "Esc", "m")
			}
		}
		d.Press("3", "1", "/", "f")
		d.Type("x1b")
		d.Press("Esc")
		d.Resize(60, 24)
		d.Resize(80, 40)
	})
}
