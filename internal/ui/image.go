package ui

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

const dataURIMaxBytes = 1 << 20 // 1 MiB: cap so the wrapped blob stays sane

// imageMime maps an extension to its data-URI media type. base64 encoding needs
// no decoder, so vector/exotic formats are fine here too.
var imageMime = map[string]string{
	"png": "image/png", "jpg": "image/jpeg", "jpeg": "image/jpeg",
	"gif": "image/gif", "bmp": "image/bmp", "webp": "image/webp",
	"tiff": "image/tiff", "tif": "image/tiff", "svg": "image/svg+xml",
	"ico": "image/x-icon", "avif": "image/avif", "heic": "image/heic",
}

func isImage(name string) bool {
	_, ok := imageMime[strings.ToLower(strings.TrimPrefix(filepath.Ext(name), "."))]
	return ok
}

// imageDataURI renders an image as a base64 data: URI wrapped to w columns so
// the whole string scrolls into view. ok is false when the file can't be read;
// a file over dataURIMaxBytes shows a note instead of an enormous blob.
func imageDataURI(path, name string, w int) (lines []string, ok bool) {
	mime := imageMime[strings.ToLower(strings.TrimPrefix(filepath.Ext(name), "."))]
	if mime == "" {
		return nil, false
	}
	info, err := os.Stat(path)
	if err != nil {
		return nil, false
	}
	dim := lipgloss.NewStyle().Foreground(dimColor)
	head := dim.Render(fmt.Sprintf("data:%s · %s", mime, humanSize(info.Size())))
	if info.Size() > dataURIMaxBytes {
		note := dim.Render(fmt.Sprintf("(%s — too large to inline as a data URI)", humanSize(info.Size())))
		return []string{head, "", note}, true
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}
	uri := "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(data)
	return append([]string{head, ""}, wrapCols(uri, max(w, 1))...), true
}

// wrapCols hard-wraps ASCII s into lines of at most w columns (base64 is ASCII,
// so byte slicing is column-accurate).
func wrapCols(s string, w int) []string {
	var lines []string
	for len(s) > w {
		lines = append(lines, s[:w])
		s = s[w:]
	}
	return append(lines, s)
}
