package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// shellRun runs `filu shell args…` for env, with what it printed.
func shellRun(env shellEnv, args ...string) (code int, out, errOut string) {
	var o, e bytes.Buffer
	code = shellCmd(args, env, &o, &e)
	return code, o.String(), e.String()
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// filu shell init prints the function, for the line filu shell install
// writes; only for the shells install knows.
func TestShellInit(t *testing.T) {
	for _, sh := range []string{"zsh", "bash"} {
		if code, out, _ := shellRun(shellEnv{}, "init", sh); code != 0 || out != shellWrapper {
			t.Errorf("init %s: %d %q", sh, code, out)
		}
	}
	for _, args := range [][]string{{"init"}, {"init", "fish"}} {
		if code, out, errOut := shellRun(shellEnv{}, args...); code != 2 || out != "" || !strings.Contains(errOut, "zsh or bash") {
			t.Errorf("%v: %d %q %q", args, code, out, errOut)
		}
	}
}

// filu shell no longer prints the function. In a terminal it says to use
// filu shell install; run by an old eval "$(filu shell)" it prints shell code
// that only says so on stderr, so the shell starts without errors.
func TestShellBare(t *testing.T) {
	_, out, _ := shellRun(shellEnv{tty: true})
	if !strings.Contains(out, "filu shell install") || strings.Contains(out, "FILU__LAST_DIR_FILE") {
		t.Fatalf("in a terminal: %q", out)
	}
	_, code, _ := shellRun(shellEnv{tty: false})
	cmd := exec.Command("sh", "-c", `eval "$1"`, "sh", code)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil || stdout.Len() != 0 ||
		!strings.Contains(stderr.String(), `eval "$(filu shell)" is no longer used`) || !strings.Contains(stderr.String(), "filu shell install") {
		t.Fatalf("eval of %q: %v, stdout %q, stderr %q", code, err, stdout.String(), stderr.String())
	}
}

// filu shell install writes the block into the rc file of the user's shell,
// once: again it says so, an older block is brought up to date. Lines that set
// it up the old way are pointed at, not touched.
func TestShellInstall(t *testing.T) {
	home := t.TempDir()
	for _, c := range []struct {
		env  shellEnv
		rc   string
		line string
	}{
		{shellEnv{shell: "/bin/zsh", home: home}, ".zshrc", blockLine["zsh"]},
		{shellEnv{shell: "/opt/homebrew/bin/zsh", home: home, zdotdir: filepath.Join(home, "zdot")}, "zdot/.zshrc", blockLine["zsh"]},
		{shellEnv{shell: "/bin/bash", home: home, goos: "linux"}, ".bashrc", blockLine["bash"]},
		{shellEnv{shell: "/bin/bash", home: home, goos: "darwin"}, ".bash_profile", blockLine["bash"]},
	} {
		rc := filepath.Join(home, c.rc)
		if err := os.MkdirAll(filepath.Dir(rc), 0o755); err != nil {
			t.Fatal(err)
		}
		mine := "export EDITOR=vim\neval \"$(filu shell)\"\n"
		if err := os.WriteFile(rc, []byte(mine), 0o600); err != nil {
			t.Fatal(err)
		}
		block := blockStart + "\n" + c.line + "\n" + blockEnd + "\n"
		code, out, _ := shellRun(c.env, "install")
		if got := readFile(t, rc); code != 0 || got != mine+"\n"+block {
			t.Fatalf("%s: %d\n%s", c.rc, code, got)
		}
		if !strings.Contains(out, "Added cd-on-quit in "+rc) || !strings.Contains(out, `line 2: eval "$(filu shell)"`) {
			t.Fatalf("%s: %q", c.rc, out)
		}
		if fi, _ := os.Stat(rc); fi.Mode().Perm() != 0o600 {
			t.Fatalf("%s: the mode became %v", c.rc, fi.Mode())
		}
		if _, out, _ = shellRun(c.env, "install"); !strings.Contains(out, "already in") || readFile(t, rc) != mine+"\n"+block {
			t.Fatalf("%s again: %q\n%s", c.rc, out, readFile(t, rc))
		}
		old := mine + "\n" + blockStart + "\neval \"$(filu shell)\"\n" + blockEnd + "\n"
		if err := os.WriteFile(rc, []byte(old), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, out, _ = shellRun(c.env, "install"); !strings.Contains(out, "Updated") || readFile(t, rc) != mine+"\n"+block {
			t.Fatalf("%s update: %q\n%s", c.rc, out, readFile(t, rc))
		}
	}
	if code, _, errOut := shellRun(shellEnv{shell: "/usr/bin/fish", home: home}, "install"); code != 1 || !strings.Contains(errOut, "zsh and bash") {
		t.Fatalf("fish: %d %q", code, errOut)
	}
}

// A new rc file is made; one that is a link (a dotfiles repo) stays a link,
// its file written.
func TestShellInstallNewAndLinkedRC(t *testing.T) {
	home := t.TempDir()
	env := shellEnv{shell: "/bin/zsh", home: home}
	if code, _, _ := shellRun(env, "install"); code != 0 || readFile(t, filepath.Join(home, ".zshrc")) != blockStart+"\n"+blockLine["zsh"]+"\n"+blockEnd+"\n" {
		t.Fatal("no new .zshrc")
	}

	home = t.TempDir()
	env.home = home
	real := filepath.Join(home, "dotfiles", "zshrc")
	if err := os.MkdirAll(filepath.Dir(real), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(real, []byte("alias l=ls\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(real, filepath.Join(home, ".zshrc")); err != nil {
		t.Fatal(err)
	}
	shellRun(env, "install")
	if fi, err := os.Lstat(filepath.Join(home, ".zshrc")); err != nil || fi.Mode()&os.ModeSymlink == 0 {
		t.Fatal(".zshrc is no longer a link")
	}
	if !strings.Contains(readFile(t, real), blockLine["zsh"]) {
		t.Fatalf("the linked file: %q", readFile(t, real))
	}
}

// filu shell uninstall takes the block out of every rc file it may be in,
// leaving the rest as it was.
func TestShellUninstall(t *testing.T) {
	home := t.TempDir()
	mine := "export EDITOR=vim\n"
	for _, c := range []struct {
		env shellEnv
		rc  string
	}{
		{shellEnv{shell: "/bin/zsh", home: home}, ".zshrc"},
		{shellEnv{shell: "/bin/bash", home: home, goos: "linux"}, ".bashrc"},
		{shellEnv{shell: "/bin/bash", home: home, goos: "darwin"}, ".bash_profile"},
	} {
		if err := os.WriteFile(filepath.Join(home, c.rc), []byte(mine), 0o644); err != nil {
			t.Fatal(err)
		}
		shellRun(c.env, "install")
	}
	if err := os.WriteFile(filepath.Join(home, ".bashrc"), []byte(readFile(t, filepath.Join(home, ".bashrc"))+"alias l=ls\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, out, _ := shellRun(shellEnv{shell: "/bin/zsh", home: home}, "uninstall")
	for _, rc := range []string{".zshrc", ".bashrc", ".bash_profile"} {
		want := mine
		if rc == ".bashrc" {
			want += "alias l=ls\n"
		}
		if got := readFile(t, filepath.Join(home, rc)); got != want {
			t.Errorf("%s: %q", rc, got)
		}
		if !strings.Contains(out, "Took cd-on-quit out of "+filepath.Join(home, rc)) {
			t.Errorf("%s: %q", rc, out)
		}
	}
	if _, out, _ = shellRun(shellEnv{shell: "/bin/zsh", home: home}, "uninstall"); !strings.Contains(out, "No cd-on-quit") {
		t.Fatalf("again: %q", out)
	}

	// An rc file of nothing but the block is left empty.
	home = t.TempDir()
	env := shellEnv{shell: "/bin/zsh", home: home}
	shellRun(env, "install")
	shellRun(env, "uninstall")
	if got := readFile(t, filepath.Join(home, ".zshrc")); got != "" {
		t.Fatalf("left %q", got)
	}
}
