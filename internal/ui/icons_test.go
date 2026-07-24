package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

// TestFileIcon checks eza's icon_for_file precedence: a directory matches
// dirIcon by exact name (else the folder glyph); a file matches nameIcon by
// exact name, then extIcon by lower-cased extension; an unknown extension falls
// back to the plain file glyph and an extensionless unknown to FILE_UNKNOW.
func TestFileIcon(t *testing.T) {
	cases := []struct {
		it   fileItem
		want string
	}{
		{fileItem{name: ".git", isDir: true}, string(dirIcon[".git"])},
		{fileItem{name: "whatever", isDir: true}, iconDir}, // dir with no special name
		{fileItem{name: "main.go"}, string(extIcon["go"])},
		{fileItem{name: "App.TSX"}, string(extIcon["tsx"])},          // ext lower-cased
		{fileItem{name: "README.md"}, string(nameIcon["README.md"])}, // name beats .md ext
		{fileItem{name: "go.mod"}, string(nameIcon["go.mod"])},
		{fileItem{name: "Dockerfile"}, string(nameIcon["Dockerfile"])}, // exact name, no ext
		{fileItem{name: "mystery.zzz"}, iconFile},                      // unknown extension
		{fileItem{name: "noext"}, iconFileUnknown},                     // no extension at all
	}
	for _, c := range cases {
		if got := fileIcon(c.it); got != c.want {
			t.Errorf("fileIcon(%q) = %q, want %q", c.it.name, got, c.want)
		}
	}
}

// suffixColor mirrors fileColor's lsSuffix step, for the expected value below.
func suffixColor(name string) lipgloss.Color {
	for _, s := range lsSuffix {
		if strings.HasSuffix(name, s.suf) {
			return s.col
		}
	}
	return ""
}

// TestFileColor checks that fileColor resolves against the baked LS_COLORS
// tables in eza's order: directory / symlink / executable filekinds, then the
// executable bit beating a name or extension match, then a longest suffix, then
// the extension, then normal.
func TestFileColor(t *testing.T) {
	cases := []struct {
		it   fileItem
		want lipgloss.Color
	}{
		{fileItem{name: "src", isDir: true}, lsDir},
		{fileItem{name: "link", isLink: true}, lsLink},
		{fileItem{name: "run", isExec: true}, lsExec},
		{fileItem{name: "deploy.sh", isExec: true}, lsExec},  // exec beats *.sh
		{fileItem{name: "main.go"}, lsExt["go"]},             // extension
		{fileItem{name: "go.mod"}, suffixColor("go.mod")},    // exact-name suffix
		{fileItem{name: "mystery.nosuchext12345"}, lsNormal}, // unknown -> normal
	}
	for _, c := range cases {
		if got := fileColor(c.it); got != c.want {
			t.Errorf("fileColor(%q) = %q, want %q", c.it.name, got, c.want)
		}
	}
}
