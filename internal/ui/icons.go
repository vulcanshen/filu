package ui

import (
	"path/filepath"
	"strings"
)

// eza-style Nerd Font file-type icons. The lookup tables live in ezadata.go
// (generated from eza's sources); this file holds the few special glyphs and
// the resolution logic. Built from rune values so no PUA glyph sits in source.
// iconFileUnknown is eza's FILE_UNKNOW, used for an extensionless file with no
// name match. The folder / plain-file defaults are iconDir / iconFile (list.go).
var iconFileUnknown = string(rune(0xf086f))

// fileIcon returns the eza glyph for an entry, mirroring eza's icon_for_file
// precedence: a directory matches DIRECTORY_ICONS by exact name (else the folder
// glyph); a file matches FILENAME_ICONS by exact name, then EXTENSION_ICONS by
// lower-cased extension; an entry with no extension and no name match falls back
// to the "unknown file" glyph.
func fileIcon(it fileItem) string {
	if it.isDir {
		if g, ok := dirIcon[it.name]; ok {
			return string(g)
		}
		return iconDir
	}
	if g, ok := nameIcon[it.name]; ok {
		return string(g)
	}
	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(it.name), "."))
	if ext == "" {
		return iconFileUnknown
	}
	if g, ok := extIcon[ext]; ok {
		return string(g)
	}
	return iconFile
}
