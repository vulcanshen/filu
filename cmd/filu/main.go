// Command filu is a ZLC terminal file manager (kbu u-family).
package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/vulcanshen/filu/internal/ui"
	"github.com/vulcanshen/filu/internal/version"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "version" {
		fmt.Println("filu " + version.Display()) // e.g. "filu v0.1.0" (a release) or "filu dev"
		return
	}
	if len(os.Args) > 1 && os.Args[1] == "shell" {
		fmt.Print(shellWrapper) // `eval "$(filu shell)"` in your rc enables cd-on-quit
		return
	}
	if len(os.Args) > 1 && os.Args[1] == "iconwidth" {
		ui.DetectIconWidth() // probe + report the detected Nerd Font icon cell width
		fmt.Println(ui.IconCells())
		return
	}
	ui.DetectIconWidth() // measure Nerd Font icon cell width (CJK fonts draw them 2-wide)
	p := tea.NewProgram(ui.New(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "filu:", err)
		os.Exit(1)
	}
}

// shellWrapper is a shell function that cd's to the directory filu wrote on quit
// (cd-on-quit). It passes filu a temp file via FILU_LAST_DIR_FILE, then cds to
// whatever filu recorded there. Enable with: eval "$(filu shell)".
const shellWrapper = `filu() {
  local __filu_dir_file __filu_dir
  __filu_dir_file="$(mktemp -t filu-cwd.XXXXXX)" || return
  FILU_LAST_DIR_FILE="$__filu_dir_file" command filu "$@"
  __filu_dir="$(cat -- "$__filu_dir_file" 2>/dev/null)"
  rm -f -- "$__filu_dir_file"
  [ -n "$__filu_dir" ] && [ -d "$__filu_dir" ] && cd -- "$__filu_dir"
}
`
