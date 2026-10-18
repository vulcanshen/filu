package main

import (
	"strings"
	"testing"
)

// tdp E2: the cd-on-quit shell function hands filu the file under
// the family name FILU__LAST_DIR_FILE, the one filu reads; the old
// single-underscore name is gone.
func TestShellWrapperUsesTheFamilyName(t *testing.T) {
	if !strings.Contains(shellWrapper, `FILU__LAST_DIR_FILE="$__filu_dir_file" command filu`) {
		t.Errorf("the shell function should set FILU__LAST_DIR_FILE:\n%s", shellWrapper)
	}
	if strings.Contains(shellWrapper, "FILU_LAST_DIR_FILE") {
		t.Errorf("the shell function still sets the old name:\n%s", shellWrapper)
	}
}

// filu help names every command, version and help among them with their
// aliases (tdp E1), and the variables filu reads (tdp E2).
func TestUsage(t *testing.T) {
	for _, want := range []string{"filu version", "--version, -v", "filu help", "--help, -h", "filu shell", "filu iconwidth",
		"FILU__CONFIG", "FILU__STATE", "FILU__ICON_WIDTH", "FILU__POWERLINE_WIDTH", "TERMINU__ICON_WIDTH", "TERMINU__POWERLINE_WIDTH"} {
		if !strings.Contains(usage, want) {
			t.Errorf("filu help does not say %q", want)
		}
	}
}
