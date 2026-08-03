package ui

import (
	"os"
	"path/filepath"
)

// ResolveStartDir turns a `filu <path>` argument into the directory to open and,
// when the path points at a file, the entry name to land the cursor on. A
// relative path is made absolute against the CWD. A path that doesn't exist (or
// can't be stat'd) returns an error so main can report it and exit.
func ResolveStartDir(arg string) (dir, focus string, err error) {
	abs, err := filepath.Abs(arg)
	if err != nil {
		return "", "", err
	}
	info, err := os.Stat(abs)
	if err != nil {
		return "", "", err
	}
	if info.IsDir() {
		return abs, "", nil
	}
	return filepath.Dir(abs), filepath.Base(abs), nil // a file: open its parent, focus it
}
