//go:build darwin

package ui

import "os/exec"

// osOpen hands path to the macOS launcher, which opens it with the default app.
// Run (not Start) reaps `open`, which returns as soon as it has dispatched.
func osOpen(path string) error {
	return exec.Command("open", path).Run()
}
