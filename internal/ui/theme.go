package ui

import (
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// eza catppuccin-mocha (blue accent) file-type colours.
// https://github.com/catppuccin/eza — themes/mocha/catppuccin-mocha-blue.yml
var (
	colDir    = lipgloss.Color("#89b4fa") // directory (accent)
	colLink   = lipgloss.Color("#89b4fa") // symlink (accent)
	colNormal = lipgloss.Color("#cdd6f4") // text / document
	ezaImage  = lipgloss.Color("#f9e2af") // yellow
	ezaVideo  = lipgloss.Color("#f38ba8") // red
	ezaMusic  = lipgloss.Color("#a6e3a1") // green
	ezaLoss   = lipgloss.Color("#94e2d5") // teal (lossless audio)
	ezaComp   = lipgloss.Color("#f5c2e7") // pink (compressed)
	ezaTemp   = lipgloss.Color("#eba0ac") // maroon
	ezaBuilt  = lipgloss.Color("#74c7ec") // sapphire (compiled)
	ezaCrypto = lipgloss.Color("#7f849c") // overlay1
	ezaSource = lipgloss.Color("#89b4fa") // blue (source code)
)

var extColor = map[string]lipgloss.Color{}

func init() {
	reg := func(c lipgloss.Color, exts ...string) {
		for _, e := range exts {
			extColor[e] = c
		}
	}
	reg(ezaImage, "png", "jpg", "jpeg", "gif", "bmp", "svg", "webp", "ico", "tiff", "heic", "heif", "avif")
	reg(ezaVideo, "mp4", "mkv", "mov", "avi", "webm", "flv", "wmv", "m4v", "mpg", "mpeg")
	reg(ezaMusic, "mp3", "ogg", "m4a", "aac", "wma", "opus")
	reg(ezaLoss, "flac", "wav", "alac", "aiff")
	reg(ezaComp, "zip", "tar", "gz", "tgz", "bz2", "tbz", "xz", "7z", "rar", "zst", "lz", "lzma")
	reg(ezaTemp, "tmp", "swp", "bak", "old", "orig")
	reg(ezaBuilt, "o", "so", "a", "class", "pyc", "pyo", "obj", "dll", "lib")
	reg(ezaCrypto, "gpg", "asc", "pgp", "sig")
	reg(ezaSource, "go", "py", "js", "ts", "jsx", "tsx", "rs", "c", "cc", "cpp", "h", "hpp",
		"java", "rb", "php", "swift", "kt", "scala", "sh", "bash", "zsh", "lua", "vim", "el",
		"sql", "html", "css", "scss", "json", "yaml", "yml", "toml")
}

// fileColor returns the eza-mocha colour for an entry.
func fileColor(it fileItem) lipgloss.Color {
	switch {
	case it.isDir:
		return colDir
	case it.isLink:
		return colLink
	}
	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(it.name), "."))
	if c, ok := extColor[ext]; ok {
		return c
	}
	return colNormal
}
