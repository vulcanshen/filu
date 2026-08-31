package ui

import (
	"errors"
	"os"
	"path/filepath"
)

// moveToTrash moves path into the system trash (macOS ~/.Trash, Linux XDG).
// Recovery is via the OS trash UI — filu has no undelete. TODO: Linux
// .trashinfo sidecar; macOS Finder "Put Back" needs the Cocoa API (cgo).
func moveToTrash(path string) error {
	td := trashDir()
	if td == "" {
		return errors.New("no trash dir")
	}
	if err := os.MkdirAll(td, 0o700); err != nil {
		return err
	}
	dst := uniquePath(filepath.Join(td, filepath.Base(path)))
	return movePath(path, dst)
}
