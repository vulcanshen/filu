//go:build darwin || linux

package ui

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/creack/pty"
	"github.com/hinshun/vt10x"
)

// ptyTickMsg is the 50ms heartbeat that polls the subprocess done flag and
// refreshes the embedded terminal grid.
type ptyTickMsg struct{}

// ptyExitMsg fires when the editor subprocess exits; dir is reloaded so an edit
// that changed the file's size/mtime shows immediately.
type ptyExitMsg struct{ dir string }

// ptyPopup runs $EDITOR on a file inside an embedded PTY popup, so the editor
// renders within filu instead of taking over the whole screen (the approach kbu
// settled on). It is a pointer field on AppModel — the read goroutine and the
// value-copied model must share the same terminal/mutex/file handles.
type ptyPopup struct {
	active      bool
	stopPending bool // subprocess exited; hard cleanup deferred until the close animation settles
	title       string
	dir         string
	hostW       int
	hostH       int

	ptmx *os.File
	term vt10x.Terminal
	cmd  *exec.Cmd
	done *atomic.Bool
	mu   *sync.Mutex

	anim popupAnimator
}

func newPtyPopup() *ptyPopup {
	return &ptyPopup{anim: newPopupAnimator("pty", popupLayerColor(1))}
}

// isActive / isRendered are nil-safe so a zero-value AppModel (e.g. in tests
// that skip newPtyPopup) can still run View/Update without the editor.
func (p *ptyPopup) isActive() bool   { return p != nil && p.active }
func (p *ptyPopup) isRendered() bool { return p != nil && (p.active || p.anim.isActive()) }

// start launches cmd in a PTY sized to a popup inside hostW×hostH, rooted at dir
// — a shell takes no path argument, so cmd.Dir is what puts it in the active
// tab's directory; dir is also the directory reloaded when the process exits.
func (p *ptyPopup) start(cmd *exec.Cmd, title, dir string, hostW, hostH int) tea.Cmd {
	p.active = true
	p.stopPending = false
	p.title = title
	p.dir = dir
	p.cmd = cmd
	p.hostW, p.hostH = hostW, hostH
	p.done = &atomic.Bool{}
	p.mu = &sync.Mutex{}

	cols, rows := p.dims()
	p.term = vt10x.New(vt10x.WithSize(cols, rows))
	cmd.Dir = dir // root the process in the tab's directory (the shell opens here)
	ptmx, err := pty.StartWithSize(cmd, &pty.Winsize{Cols: uint16(cols), Rows: uint16(rows)})
	if err != nil {
		p.active = false
		return func() tea.Msg { return ptyExitMsg{dir: dir} }
	}
	p.ptmx = ptmx
	go p.readLoop()
	return tea.Batch(p.tick(), p.anim.open())
}

func (p *ptyPopup) tick() tea.Cmd {
	return tea.Tick(50*time.Millisecond, func(time.Time) tea.Msg { return ptyTickMsg{} })
}

// readLoop copies the PTY output into the terminal emulator until the pipe
// closes, then reaps the subprocess. Local pointer copies stay valid even if
// stop() clears the fields concurrently.
func (p *ptyPopup) readLoop() {
	ptmx, cmd, done := p.ptmx, p.cmd, p.done
	buf := make([]byte, 4096)
	for {
		n, err := ptmx.Read(buf)
		if n > 0 {
			p.mu.Lock()
			if p.term != nil {
				_, _ = p.term.Write(buf[:n])
			}
			p.mu.Unlock()
		}
		if err != nil {
			break
		}
	}
	_ = cmd.Wait()
	done.Store(true)
}

// stop force-terminates the subprocess and releases the terminal. Idempotent.
func (p *ptyPopup) stop() {
	if !p.active {
		return
	}
	cmd, ptmx, done := p.cmd, p.ptmx, p.done
	if cmd != nil && cmd.Process != nil && done != nil && !done.Load() {
		_ = cmd.Process.Kill()
	}
	if ptmx != nil {
		_ = ptmx.Close()
	}
	p.active = false
	p.mu.Lock()
	p.term = nil
	p.mu.Unlock()
	p.cmd = nil
	p.ptmx = nil
}

// handleTick advances the open/close animation and runs the deferred hard
// cleanup once the close animation settles.
func (p *ptyPopup) handleTick(msg AnimTickMsg) tea.Cmd {
	if p == nil || msg.Target != p.anim.target {
		return nil
	}
	cmd := p.anim.tick()
	if p.stopPending && p.anim.state == popupClosed {
		p.stop()
		p.stopPending = false
	}
	return cmd
}

// update polls for exit (ptyTickMsg) and forwards keystrokes to the editor.
func (p *ptyPopup) update(msg tea.Msg) tea.Cmd {
	if p == nil || !p.active {
		return nil
	}
	switch msg := msg.(type) {
	case ptyTickMsg:
		if p.done != nil && p.done.Load() {
			if !p.stopPending {
				p.stopPending = true // two-phase: play the close animation, defer the hard stop
				dir := p.dir
				return tea.Batch(p.anim.close(), func() tea.Msg { return ptyExitMsg{dir: dir} })
			}
			return nil
		}
		return p.tick()
	case tea.KeyMsg:
		if p.ptmx == nil {
			return nil
		}
		appCursor := p.term != nil && p.term.Mode()&vt10x.ModeAppCursor != 0
		if raw := ptyKeyBytes(msg, appCursor); len(raw) > 0 {
			_, _ = p.ptmx.Write(raw)
		}
		return nil
	}
	return nil
}

// setSize resizes the PTY (SIGWINCH) so the editor redraws to the new size.
func (p *ptyPopup) setSize(hostW, hostH int) {
	if p == nil {
		return
	}
	p.hostW, p.hostH = hostW, hostH
	if !p.active || p.ptmx == nil {
		return
	}
	cols, rows := p.dims()
	_ = pty.Setsize(p.ptmx, &pty.Winsize{Cols: uint16(cols), Rows: uint16(rows)})
	p.mu.Lock()
	if p.term != nil {
		p.term.Resize(cols, rows)
	}
	p.mu.Unlock()
}

// ptyChromeRows is how many rows of filu stay visible above the shell popup: the
// header bar + the directory status line. The popup pins just below them so the
// "where am I" context shows without the shell having to print it (see view.go's
// overlay offset, which must match).
const ptyChromeRows = 2

// dims is the PTY content size inside the popup. The popup is full width and runs
// from just below filu's header+status down to the very bottom (covering the
// panels + footer), so the content is the full width minus the two side borders,
// and the height below the chrome minus the top/bottom border.
func (p *ptyPopup) dims() (cols, rows int) {
	cols = p.hostW - 2                 // full width, minus the two │ borders
	rows = p.hostH - ptyChromeRows - 2 // below header+status, minus top/bottom border
	if cols < 20 {
		cols = 20
	}
	if rows < 5 {
		rows = 5
	}
	return cols, rows
}

// renderPopup composes the bordered editor grid for overlay compositing.
func (p *ptyPopup) renderPopup() string {
	if (!p.active && !p.anim.isActive()) || p.term == nil {
		return ""
	}
	cols, rows := p.term.Size()

	var lines []string
	p.mu.Lock()
	cursorX, cursorY := -1, -1
	if p.term.CursorVisible() {
		c := p.term.Cursor()
		cursorX, cursorY = c.X, c.Y
	}
	for y := range rows {
		var line strings.Builder
		for x := range cols {
			line.WriteString(renderGlyph(p.term.Cell(x, y), x == cursorX && y == cursorY))
		}
		lines = append(lines, line.String())
	}
	p.mu.Unlock()

	bs := lipgloss.NewStyle().Foreground(popupLayerColor(1))
	ts := bs.Bold(true)
	title := " " + p.title + " "
	if lipgloss.Width(title) > cols-2 {
		title = " "
	}
	// The box interior is cols wide; the top row is ╭ + ─ + title + trail─ + ╮, so
	// the runs after ╭ (one ─, the title, the trail) must sum to cols → trail =
	// cols - width(title) - 1. (The bottom row below uses the same arithmetic.)
	trail := max(cols-lipgloss.Width(title)-1, 0)
	top := bs.Render("╭─") + ts.Render(title) + bs.Render(strings.Repeat("─", trail)+"╮")
	vbar := bs.Render("│")

	var out strings.Builder
	out.WriteString(top + "\n")
	for _, line := range lines {
		out.WriteString(vbar + line + vbar + "\n")
	}
	out.WriteString(bs.Render("╰─") + ts.Render(" type exit to close ") + bs.Render(strings.Repeat("─", max(cols-len(" type exit to close ")-1, 0))+"╯"))
	return p.anim.renderFrame(out.String())
}

// vt10x attribute bit positions (fixed by iota order in vt10x/state.go).
const (
	vtAttrReverse   int16 = 1
	vtAttrUnderline int16 = 2
	vtAttrBold      int16 = 4
	vtAttrItalic    int16 = 16
	vtAttrAny             = vtAttrReverse | vtAttrUnderline | vtAttrBold | vtAttrItalic
)

// renderGlyph maps a vt10x cell to a styled rune. Default-everything cells emit
// the raw rune (hot-path: avoids a lipgloss allocation per cell per frame).
func renderGlyph(g vt10x.Glyph, isCursor bool) string {
	ch := string(g.Char)
	if g.Char == 0 {
		ch = " "
	}
	defaultFG := g.FG == vt10x.DefaultFG
	defaultBG := g.BG == vt10x.DefaultBG
	hasAttrs := g.Mode&vtAttrAny != 0
	if !isCursor && defaultFG && defaultBG && !hasAttrs {
		return ch
	}
	style := lipgloss.NewStyle()
	if !defaultFG {
		if fg, ok := vtColorToLipgloss(g.FG); ok {
			style = style.Foreground(fg)
		}
	}
	if !defaultBG {
		if bg, ok := vtColorToLipgloss(g.BG); ok {
			style = style.Background(bg)
		}
	}
	if g.Mode&vtAttrBold != 0 {
		style = style.Bold(true)
	}
	if g.Mode&vtAttrUnderline != 0 {
		style = style.Underline(true)
	}
	if g.Mode&vtAttrItalic != 0 {
		style = style.Italic(true)
	}
	reverse := g.Mode&vtAttrReverse != 0
	if isCursor {
		reverse = !reverse
	}
	if reverse {
		style = style.Reverse(true)
	}
	return style.Render(ch)
}

// vtColorToLipgloss maps a vt10x color to lipgloss: 0–255 palette index, 256+
// true-color RGB. Defaults return ok=false so the host default applies.
func vtColorToLipgloss(c vt10x.Color) (lipgloss.Color, bool) {
	if c == vt10x.DefaultFG || c == vt10x.DefaultBG || c == vt10x.DefaultCursor {
		return "", false
	}
	u := uint32(c)
	if u < 256 {
		return lipgloss.Color(fmt.Sprintf("%d", u)), true
	}
	return lipgloss.Color(fmt.Sprintf("#%02X%02X%02X", (u>>16)&0xFF, (u>>8)&0xFF, u&0xFF)), true
}

// buildEditorCmd runs $EDITOR (or $VISUAL, else vi) on path, with the env
// sanitised for vt10x (a basic emulator that stalls on some terminals' probes).
func buildEditorCmd(path string) *exec.Cmd {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = os.Getenv("VISUAL")
	}
	if editor == "" {
		editor = "vi"
	}
	fields := strings.Fields(editor) // EDITOR may carry args, e.g. "nvim -p"
	c := exec.Command(fields[0], append(fields[1:], path)...)
	c.Env = sanitizeEditorEnv()
	return c
}

// buildShellCmd runs the user's shell ($SHELL, else /bin/sh) with the env
// sanitised for vt10x (same as the editor). ptyPopup.start attaches it to a PTY,
// so the shell sees a tty and starts interactive, and sets the working
// directory; exit the shell to return to filu.
func buildShellCmd() *exec.Cmd {
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/sh"
	}
	c := exec.Command(shell)
	c.Env = sanitizeEditorEnv()
	return c
}

func sanitizeEditorEnv() []string {
	out := make([]string, 0, len(os.Environ())+1)
	for _, kv := range os.Environ() {
		switch k, _, _ := strings.Cut(kv, "="); k {
		case "TERM", "TERM_PROGRAM", "TERM_PROGRAM_VERSION",
			"KITTY_WINDOW_ID", "ITERM_SESSION_ID", "ITERM_PROFILE",
			"WEZTERM_PANE", "ALACRITTY_WINDOW_ID":
			continue // drop terminal-specific hints; vt10x is a plain xterm
		default:
			out = append(out, kv)
		}
	}
	return append(out, "TERM=xterm-256color")
}

// ptyKeyBytes converts a Bubble Tea KeyMsg into the raw bytes a real terminal
// writes to a process stdin. appCursor selects the DEC application-cursor
// sequences when the running app set DECCKM (vim normal mode).
func ptyKeyBytes(msg tea.KeyMsg, appCursor bool) []byte {
	b := ptyKeyBytesPlain(msg, appCursor)
	if b == nil {
		return nil
	}
	if msg.Alt {
		return append([]byte{'\x1b'}, b...)
	}
	return b
}

func ptyKeyBytesPlain(msg tea.KeyMsg, appCursor bool) []byte {
	if msg.Type == tea.KeyRunes {
		return []byte(string(msg.Runes))
	}
	if appCursor {
		if b, ok := ptyKeyBytesAppCursorMap[msg.Type]; ok {
			return b
		}
	}
	if b, ok := ptyKeyBytesMap[msg.Type]; ok {
		return b
	}
	if s := msg.String(); len(s) == 1 {
		return []byte(s)
	}
	return nil
}

var ptyKeyBytesMap = map[tea.KeyType][]byte{
	tea.KeyEnter: {'\r'}, tea.KeyTab: {'\t'}, tea.KeyBackspace: {'\x7f'},
	tea.KeyDelete: {'\x1b', '[', '3', '~'}, tea.KeySpace: {' '}, tea.KeyEscape: {'\x1b'},
	tea.KeyUp: {'\x1b', '[', 'A'}, tea.KeyDown: {'\x1b', '[', 'B'},
	tea.KeyRight: {'\x1b', '[', 'C'}, tea.KeyLeft: {'\x1b', '[', 'D'},
	tea.KeyHome: {'\x1b', '[', 'H'}, tea.KeyEnd: {'\x1b', '[', 'F'},
	tea.KeyPgUp: {'\x1b', '[', '5', '~'}, tea.KeyPgDown: {'\x1b', '[', '6', '~'},
	tea.KeyShiftTab: {'\x1b', '[', 'Z'},
	tea.KeyCtrlA:    {'\x01'}, tea.KeyCtrlB: {'\x02'}, tea.KeyCtrlC: {'\x03'},
	tea.KeyCtrlD: {'\x04'}, tea.KeyCtrlE: {'\x05'}, tea.KeyCtrlF: {'\x06'},
	tea.KeyCtrlG: {'\x07'}, tea.KeyCtrlH: {'\x08'}, tea.KeyCtrlK: {'\x0b'},
	tea.KeyCtrlL: {'\x0c'}, tea.KeyCtrlN: {'\x0e'}, tea.KeyCtrlO: {'\x0f'},
	tea.KeyCtrlP: {'\x10'}, tea.KeyCtrlR: {'\x12'}, tea.KeyCtrlU: {'\x15'},
	tea.KeyCtrlV: {'\x16'}, tea.KeyCtrlW: {'\x17'}, tea.KeyCtrlX: {'\x18'},
	tea.KeyCtrlY: {'\x19'}, tea.KeyCtrlZ: {'\x1a'},
	tea.KeyCtrlLeft: {'\x1b', '[', '1', ';', '5', 'D'}, tea.KeyCtrlRight: {'\x1b', '[', '1', ';', '5', 'C'},
	tea.KeyShiftLeft: {'\x1b', '[', '1', ';', '2', 'D'}, tea.KeyShiftRight: {'\x1b', '[', '1', ';', '2', 'C'},
	tea.KeyF1: {'\x1b', 'O', 'P'}, tea.KeyF2: {'\x1b', 'O', 'Q'},
	tea.KeyF3: {'\x1b', 'O', 'R'}, tea.KeyF4: {'\x1b', 'O', 'S'},
	tea.KeyF5: {'\x1b', '[', '1', '5', '~'}, tea.KeyF6: {'\x1b', '[', '1', '7', '~'},
	tea.KeyF7: {'\x1b', '[', '1', '8', '~'}, tea.KeyF8: {'\x1b', '[', '1', '9', '~'},
	tea.KeyF9: {'\x1b', '[', '2', '0', '~'}, tea.KeyF10: {'\x1b', '[', '2', '1', '~'},
	tea.KeyF11: {'\x1b', '[', '2', '3', '~'}, tea.KeyF12: {'\x1b', '[', '2', '4', '~'},
}

var ptyKeyBytesAppCursorMap = map[tea.KeyType][]byte{
	tea.KeyUp: {'\x1b', 'O', 'A'}, tea.KeyDown: {'\x1b', 'O', 'B'},
	tea.KeyRight: {'\x1b', 'O', 'C'}, tea.KeyLeft: {'\x1b', 'O', 'D'},
}
