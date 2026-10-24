package main

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// cd-on-quit: filu cannot change the directory of the shell it runs in, a
// shell function can. The function hands filu a file through
// FILU__LAST_DIR_FILE and cds to what filu wrote there when it quit.
// `filu shell install` puts a block in the shell's rc file that has filu
// print the function at every start (`filu shell init`), so it follows
// filu's updates; `filu shell uninstall` takes the block out.

// shellWrapper is the function `filu shell init` prints.
const shellWrapper = `filu() {
  local __filu_dir_file __filu_dir
  __filu_dir_file="$(mktemp -t filu-cwd.XXXXXX)" || return
  FILU__LAST_DIR_FILE="$__filu_dir_file" command filu "$@"
  __filu_dir="$(cat -- "$__filu_dir_file" 2>/dev/null)"
  rm -f -- "$__filu_dir_file"
  [ -n "$__filu_dir" ] && [ -d "$__filu_dir" ] && cd -- "$__filu_dir"
}
`

// The block filu shell install writes, between these two lines.
const (
	blockStart = "# >>> filu shell (filu shell uninstall removes this) >>>"
	blockEnd   = "# <<< filu shell <<<"
)

// blockLine is the block's one line for a shell: the function, once filu is
// on the PATH (a filu uninstalled by brew leaves the block behind).
var blockLine = map[string]string{
	"zsh":  `(( $+commands[filu] )) && eval "$(command filu shell init zsh)"`,
	"bash": `command -v filu >/dev/null 2>&1 && eval "$(command filu shell init bash)"`,
}

// shellGuide is what `filu shell` says in a terminal.
const shellGuide = `filu shell install    let quitting filu change your shell's directory: adds
                      a few lines to your shell's rc file (zsh or bash)
filu shell uninstall  takes them out again

filu shell no longer prints the function to copy or to eval. If your rc file
has eval "$(filu shell)", or a copy of a filu() function, take it out, then
run filu shell install.
`

// evalNotice is what `filu shell` prints when an old eval "$(filu shell)"
// runs it: shell code that only says, on stderr, what to do instead.
const evalNotice = `printf '%s\n' 'filu: eval "$(filu shell)" is no longer used: take it out of your rc file and run filu shell install' >&2
`

// shellEnv is what the shell commands look at: the user's shell, home,
// ZDOTDIR, the OS, and whether stdout is a terminal.
type shellEnv struct {
	shell, home, zdotdir, goos string
	tty                        bool
}

// shellCmd runs `filu shell [install|uninstall|init <shell>]` and returns the
// exit code.
func shellCmd(args []string, env shellEnv, out, errOut io.Writer) int {
	sub := ""
	if len(args) > 0 {
		sub = args[0]
	}
	switch sub {
	case "":
		if env.tty {
			fmt.Fprint(out, shellGuide)
		} else {
			io.WriteString(out, evalNotice) // its %s is printf's, in the shell
		}
		return 0
	case "init":
		if len(args) != 2 || blockLine[args[1]] == "" {
			fmt.Fprintln(errOut, "filu: filu shell init takes zsh or bash")
			return 2
		}
		fmt.Fprint(out, shellWrapper)
		return 0
	case "install":
		return shellInstall(env, out, errOut)
	case "uninstall":
		return shellUninstall(env, out, errOut)
	}
	fmt.Fprintf(errOut, "filu: no filu shell %s (install, uninstall)\n", sub)
	return 2
}

// rcFile is the rc file of the user's shell, and the shell's name.
func rcFile(env shellEnv) (path, name string, err error) {
	name = filepath.Base(env.shell)
	switch name {
	case "zsh":
		dir := env.home
		if env.zdotdir != "" {
			dir = env.zdotdir
		}
		return filepath.Join(dir, ".zshrc"), name, nil
	case "bash":
		if env.goos == "darwin" { // a terminal on macOS opens login shells, which read .bash_profile
			return filepath.Join(env.home, ".bash_profile"), name, nil
		}
		return filepath.Join(env.home, ".bashrc"), name, nil
	}
	return "", name, fmt.Errorf("filu shell install knows zsh and bash; your shell is %q", env.shell)
}

// shellInstall writes the block into the shell's rc file, or brings it up to
// date, and points at old setups it sees there.
func shellInstall(env shellEnv, out, errOut io.Writer) int {
	rc, name, err := rcFile(env)
	if err != nil {
		fmt.Fprintln(errOut, "filu:", err)
		return 1
	}
	text, mode, err := readRC(rc)
	if err != nil {
		fmt.Fprintln(errOut, "filu:", err)
		return 1
	}
	block := blockStart + "\n" + blockLine[name] + "\n" + blockEnd + "\n"
	rest, had := cutBlock(text)
	switch {
	case had && strings.Contains(text, block):
		fmt.Fprintf(out, "cd-on-quit is already in %s.\n", rc)
	default:
		if rest != "" && !strings.HasSuffix(rest, "\n") {
			rest += "\n"
		}
		if rest != "" {
			rest += "\n"
		}
		if err := os.WriteFile(rc, []byte(rest+block), mode); err != nil {
			fmt.Fprintln(errOut, "filu:", err)
			return 1
		}
		verb := "Added"
		if had {
			verb = "Updated"
		}
		fmt.Fprintf(out, "%s cd-on-quit in %s. Open a new shell (or run: exec %s) for it to take effect.\n", verb, rc, name)
	}
	if old := oldSetups(rest); len(old) > 0 {
		fmt.Fprintf(out, "\n%s still has the old way of doing it; take these lines out, the later one wins:\n", rc)
		for _, l := range old {
			fmt.Fprintln(out, "  "+l)
		}
	}
	fmt.Fprintln(out, "\nA filu() function your shell loads from another file wins too when it comes later: take that out as well.")
	return 0
}

// shellUninstall takes the block out of every rc file filu shell install may
// have written.
func shellUninstall(env shellEnv, out, errOut io.Writer) int {
	dir := env.home
	if env.zdotdir != "" {
		dir = env.zdotdir
	}
	removed := false
	for _, rc := range []string{filepath.Join(dir, ".zshrc"), filepath.Join(env.home, ".bashrc"), filepath.Join(env.home, ".bash_profile")} {
		text, mode, err := readRC(rc)
		if err != nil {
			fmt.Fprintln(errOut, "filu:", err)
			return 1
		}
		rest, had := cutBlock(text)
		if !had {
			continue
		}
		if rest = strings.TrimRight(rest, "\n"); rest != "" {
			rest += "\n"
		}
		if err := os.WriteFile(rc, []byte(rest), mode); err != nil {
			fmt.Fprintln(errOut, "filu:", err)
			return 1
		}
		removed = true
		fmt.Fprintf(out, "Took cd-on-quit out of %s. Open a new shell for it to take effect.\n", rc)
	}
	if !removed {
		fmt.Fprintln(out, "No cd-on-quit from filu shell install found.")
	}
	return 0
}

// readRC reads an rc file and its mode; one that is not there is empty.
func readRC(path string) (string, fs.FileMode, error) {
	b, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return "", 0o644, nil
	}
	if err != nil {
		return "", 0, err
	}
	fi, err := os.Stat(path)
	if err != nil {
		return "", 0, err
	}
	return string(b), fi.Mode().Perm(), nil
}

// cutBlock is text without filu's block, and whether it had one; the blank
// line written before the block goes with it.
func cutBlock(text string) (string, bool) {
	i := strings.Index(text, blockStart)
	if i < 0 {
		return text, false
	}
	j := strings.Index(text[i:], blockEnd)
	if j < 0 {
		return text, false
	}
	end := i + j + len(blockEnd)
	if end < len(text) && text[end] == '\n' {
		end++
	}
	return strings.TrimSuffix(text[:i], "\n") + text[end:], true
}

// oldSetups are the lines of an rc file, outside filu's block, that set
// cd-on-quit up the old way: an eval of filu shell, or a copied function.
func oldSetups(text string) []string {
	var lines []string
	for n, l := range strings.Split(text, "\n") {
		t := strings.TrimSpace(l)
		if strings.HasPrefix(t, "#") {
			continue
		}
		if strings.Contains(t, "filu shell") || strings.Contains(t, "LAST_DIR_FILE") ||
			strings.HasPrefix(t, "filu()") || strings.HasPrefix(t, "function filu") {
			lines = append(lines, fmt.Sprintf("line %d: %s", n+1, t))
		}
	}
	return lines
}
