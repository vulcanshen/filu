package ui

import (
	"path/filepath"
	"strings"

	"github.com/TheZoraiz/ascii-image-converter/aic_package"
)

// imageExt is the raster formats aic_package (via x/image) can decode.
var imageExt = map[string]bool{
	"png": true, "jpg": true, "jpeg": true, "gif": true,
	"bmp": true, "webp": true, "tiff": true, "tif": true,
}

func isImage(name string) bool {
	return imageExt[strings.ToLower(strings.TrimPrefix(filepath.Ext(name), "."))]
}

// imageASCII renders an image as coloured ASCII art fitted to w chars wide
// (aspect preserved; a tall image spills into scrollable lines). ok is false
// when the file can't be decoded, so the caller falls back to a hex preview.
// Wrapped in recover() as belt-and-braces against a decoder panic.
func imageASCII(path string, w int) (lines []string, ok bool) {
	defer func() {
		if recover() != nil {
			lines, ok = nil, false
		}
	}()
	if w < 1 {
		return nil, false
	}
	flags := aic_package.DefaultFlags()
	flags.Colored = true
	flags.Width = w // fit to panel width; height follows the aspect ratio
	art, err := aic_package.Convert(path, flags)
	if err != nil {
		return nil, false
	}
	out := strings.Split(strings.TrimRight(art, "\n"), "\n")
	for i := range out {
		out[i] += ansiReset // stop colour bleeding into the panel padding/border
	}
	return out, true
}
