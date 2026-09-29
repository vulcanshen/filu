package main

import (
	"strings"
	"testing"
)

// tdp D6 (v0.1.21): the cd-on-quit shell function hands filu the file under
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
