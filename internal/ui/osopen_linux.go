//go:build linux

package ui

import "os/exec"

// osOpen hands path to xdg-open, which opens it with the default app.
func osOpen(path string) error {
	return exec.Command("xdg-open", path).Run()
}
