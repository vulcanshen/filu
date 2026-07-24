package ui

import (
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// fileColor returns the colour filu paints an entry, matching the user's
// terminal (eza / ls) via the baked LS_COLORS tables in lscolors_data.go — no
// LS_COLORS is needed at run time. It mirrors eza's resolution order: a
// directory, then a symlink, then the executable bit (which in eza beats any
// name or extension match), then the longest filename-suffix match (exact names
// like go.mod, multi-part extensions, ~ temps), then the plain lower-cased
// extension, then the normal colour.
func fileColor(it fileItem) lipgloss.Color {
	switch {
	case it.isDir:
		return lsDir
	case it.isLink:
		return lsLink
	case it.isExec:
		return lsExec
	}
	for _, s := range lsSuffix { // pre-sorted longest-first: first hit is most specific
		if strings.HasSuffix(it.name, s.suf) {
			return s.col
		}
	}
	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(it.name), "."))
	if c, ok := lsExt[ext]; ok {
		return c
	}
	return lsNormal
}
