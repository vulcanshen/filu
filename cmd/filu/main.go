// Command filu is a ZLC terminal file manager (kbu u-family).
package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/vulcanshen/filu/internal/ui"
)

func main() {
	ui.DetectIconWidth() // measure Nerd Font icon cell width (CJK fonts draw them 2-wide)
	p := tea.NewProgram(ui.New(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "filu:", err)
		os.Exit(1)
	}
}
