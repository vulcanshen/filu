package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// metaPopup is the file information box Enter opens on a file row (tdp K3,
// 2026-09-28 decision): every fact about the file, read only. It is its own
// popup class (tdp F1) — no cursor, nothing runs; long values wrap instead of
// being cut, and the box scrolls when it is taller than the screen. Esc closes
// it; ? lists its keys.
type metaPopup struct {
	anim    popupAnimator
	title   string
	rows    []metaRow
	top     int // first line shown when the box is taller than the screen
	screenW int
	screenH int
}

// metaRow is one fact: a label and its value.
type metaRow struct{ label, value string }

func newMetaPopup() metaPopup {
	return metaPopup{anim: newPopupAnimator("meta", popupLayerColor(1))}
}

func (m *metaPopup) setSize(w, h int)   { m.screenW, m.screenH = w, h }
func (m metaPopup) isActive() bool      { return m.anim.isActive() }
func (m metaPopup) owns() bool          { return m.anim.owns() }
func (m metaPopup) isInteractive() bool { return m.anim.isInteractive() }
func (m *metaPopup) close() tea.Cmd     { return m.anim.close() }
func (m metaPopup) renderPopup() string { return m.anim.renderFrame(m.renderFull()) }
func (m *metaPopup) handleTick(msg AnimTickMsg) tea.Cmd {
	if msg.Target != m.anim.target {
		return nil
	}
	return m.anim.tick()
}

// open reads path and shows what it finds. A fact that can't be read says so in
// the box (tdp F5) rather than leaving a blank.
func (m *metaPopup) open(path string) tea.Cmd {
	m.title = filepath.Base(path)
	m.rows = fileFacts(path)
	m.top = 0
	return m.anim.open()
}

// metaLabelW is the label column: the longest label and a two-space gap.
func (m metaPopup) metaLabelW() int {
	w := 0
	for _, r := range m.rows {
		w = max(w, dispWidth(r.label))
	}
	return w + 2
}

// width is the family inner width (tdp F7): the value wraps inside it rather
// than widening the box.
func (m metaPopup) width() int { return popupInnerWidth(m.screenW) }

// metaHint is the box's bottom border.
func metaHint() string { return keyLegend([][2]string{{"j/k", "scroll"}, {"Esc", "close"}}) }

// lines lays the facts out at the box width: label column, then the value
// wrapped under itself so every character shows.
func (m metaPopup) lines() []string {
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#7f849c"))
	labelW := m.metaLabelW()
	valueW := max(m.width()-2-labelW-1, 8)
	var out []string
	for _, r := range m.rows {
		for i, part := range wrapHard(r.value, valueW) {
			lead := strings.Repeat(" ", labelW)
			if i == 0 {
				lead = labelStyle.Render(r.label) + strings.Repeat(" ", labelW-dispWidth(r.label))
			}
			out = append(out, " "+lead+part)
		}
	}
	return out
}

// visible is how many lines fit on screen (all of them when the height is not
// known yet).
func (m metaPopup) visible() int {
	if m.screenH <= 0 {
		return len(m.lines()) + 1
	}
	return max(m.screenH-menuChrome, 3)
}

func (m metaPopup) update(msg tea.KeyMsg) (metaPopup, tea.Cmd) {
	if !m.anim.isInteractive() {
		return m, nil
	}
	n := len(m.lines())
	switch msg.String() {
	case "esc":
		return m, m.anim.close()
	case "j", "down":
		m.top = max(0, min(m.top+1, n-m.visible()))
	case "k", "up":
		m.top = max(0, m.top-1)
	case "g":
		m.top = 0
	case "G":
		m.top = max(0, n-m.visible())
	}
	return m, nil
}

func (m metaPopup) renderFull() string {
	bc := popupLayerColor(m.anim.layer)
	rows := m.lines()
	if vis := m.visible(); len(rows) > vis {
		top := max(0, min(m.top, len(rows)-vis))
		rows = rows[top : top+vis]
	}
	return drawPopupBox(bc, " "+safeName(m.title), metaHint(), rows, m.width())
}

// wrapHard cuts s into pieces at most w cells wide, breaking anywhere — a path
// has no spaces to break at, and every character has to show.
func wrapHard(s string, w int) []string {
	if w < 1 || dispWidth(s) <= w {
		return []string{s}
	}
	var out []string
	var cur strings.Builder
	cw := 0
	for _, r := range s {
		rw := dispWidth(string(r))
		if cw+rw > w {
			out = append(out, cur.String())
			cur.Reset()
			cw = 0
		}
		cur.WriteRune(r)
		cw += rw
	}
	return append(out, cur.String())
}

// fileFacts reads everything the information box shows about path.
func fileFacts(path string) []metaRow {
	abs, err := filepath.Abs(path)
	if err != nil {
		abs = path
	}
	rows := []metaRow{{"Path", safeName(abs)}}
	fi, err := os.Lstat(path)
	if err != nil {
		return append(rows, metaRow{"Error", opFailedText("read it", err)})
	}
	if fi.Mode()&os.ModeSymlink != 0 {
		target, lerr := os.Readlink(path)
		if lerr != nil {
			target = opFailedText("read the link", lerr)
		}
		rows = append(rows, metaRow{"Link to", safeName(target)})
	}
	rows = append(rows,
		metaRow{"Type", fileType(path, fi)},
		metaRow{"Size", humanSize(fi.Size()) + " (" + groupDigits(fi.Size()) + " bytes)"})
	const stamp = "2006-01-02 15:04:05"
	rows = append(rows, metaRow{"Modified", fi.ModTime().Format(stamp)})
	st, ok := osStat(fi)
	if ok {
		rows = append(rows, metaRow{"Accessed", st.atime.Format(stamp)})
		if runtime.GOOS == "darwin" && !st.btime.IsZero() {
			rows = append(rows, metaRow{"Created", st.btime.Format(stamp)})
		} else {
			rows = append(rows, metaRow{"Changed", st.ctime.Format(stamp)})
		}
	}
	rows = append(rows, metaRow{"Permissions", fi.Mode().String() + " (" + fmt.Sprintf("%04o", fi.Mode().Perm()|specialBits(fi.Mode())) + ")"})
	if ok {
		rows = append(rows, metaRow{"Owner", userName(st.uid) + ":" + groupName(st.gid)})
	} else {
		rows = append(rows, metaRow{"Owner", "unknown on this system"})
	}
	return rows
}

// specialBits maps the setuid / setgid / sticky mode flags onto their octal
// digit, so the octal reads as chmod would take it (e.g. 4755).
func specialBits(m os.FileMode) os.FileMode {
	var b os.FileMode
	if m&os.ModeSetuid != 0 {
		b |= 0o4000
	}
	if m&os.ModeSetgid != 0 {
		b |= 0o2000
	}
	if m&os.ModeSticky != 0 {
		b |= 0o1000
	}
	return b
}

// fileType names what the file is, the way the preview tells them apart: the
// file kind, then images, archives and PDFs by name, then text or binary by its
// first bytes.
func fileType(path string, fi os.FileInfo) string {
	mode := fi.Mode()
	switch {
	case mode&os.ModeSymlink != 0:
		if t, err := os.Stat(path); err == nil && t.IsDir() {
			return "Symbolic link to a directory"
		} else if err != nil {
			return "Symbolic link (broken)"
		}
		return "Symbolic link to a file"
	case mode.IsDir():
		return "Directory"
	case mode&os.ModeNamedPipe != 0:
		return "Named pipe"
	case mode&os.ModeSocket != 0:
		return "Socket"
	case mode&os.ModeDevice != 0:
		return "Device"
	}
	name := strings.ToLower(fi.Name())
	exec := ""
	if mode&0o111 != 0 {
		exec = ", executable"
	}
	switch {
	case isImage(name):
		return "Image (" + strings.TrimPrefix(filepath.Ext(name), ".") + ")" + exec
	case strings.HasSuffix(name, ".pdf"):
		return "PDF document" + exec
	}
	if _, ok := archiveEntries(path, fi.Name()); ok {
		return "Archive (" + strings.TrimPrefix(filepath.Ext(name), ".") + ")" + exec
	}
	data, err := readCapped(path, 8192)
	switch {
	case err != nil:
		return opFailedText("read it", err)
	case len(data) == 0:
		return "Empty file" + exec
	case isText(data):
		return "Text" + exec
	}
	return "Binary" + exec
}

// groupDigits writes n with thousands separators: 12601 → 12,601.
func groupDigits(n int64) string {
	s := strconv.FormatInt(n, 10)
	for i := len(s) - 3; i > 0; i -= 3 {
		s = s[:i] + "," + s[i:]
	}
	return s
}
