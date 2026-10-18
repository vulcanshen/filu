package ui

import (
	"os"
	"path/filepath"
	"testing"
)

// TestResolveStartDir: a directory arg opens as-is (no focus); a file arg opens
// its parent and focuses the file; a missing path errors.
func TestResolveStartDir(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "file.txt")
	mustWrite(t, file)

	t.Run("dir", func(t *testing.T) {
		gotDir, gotFocus, err := ResolveStartDir(dir)
		if err != nil {
			t.Fatal(err)
		}
		if gotDir != dir || gotFocus != "" {
			t.Errorf("dir arg -> (%q, %q), want (%q, \"\")", gotDir, gotFocus, dir)
		}
	})
	t.Run("file", func(t *testing.T) {
		gotDir, gotFocus, err := ResolveStartDir(file)
		if err != nil {
			t.Fatal(err)
		}
		if gotDir != dir || gotFocus != "file.txt" {
			t.Errorf("file arg -> (%q, %q), want (%q, \"file.txt\")", gotDir, gotFocus, dir)
		}
	})
	t.Run("missing", func(t *testing.T) {
		if _, _, err := ResolveStartDir(filepath.Join(dir, "nope")); err == nil {
			t.Error("a missing path should error")
		}
	})
}

// TestResolveStartDirRelative: a relative arg is made absolute against the CWD.
func TestResolveStartDirRelative(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	wd, _ := os.Getwd()
	defer os.Chdir(wd)
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	gotDir, gotFocus, err := ResolveStartDir("sub")
	if err != nil {
		t.Fatal(err)
	}
	if !filepath.IsAbs(gotDir) {
		t.Errorf("relative arg should resolve to an absolute dir, got %q", gotDir)
	}
	if gotFocus != "" {
		t.Errorf("a directory arg should have no focus, got %q", gotFocus)
	}
}
