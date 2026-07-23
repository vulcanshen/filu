package ui

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// carryModel is panel [4]'s bucket: files picked up with Carry, dropped with
// Land (which decides copy vs move). Completed lands live in AppModel.tasks.
type carryModel struct {
	items  []string        // full source paths
	cursor int             // cursor over items (carries tab)
	picked map[string]bool // land subset; empty = land everything
}

func (m *carryModel) toggle(path string) {
	for i, p := range m.items {
		if p == path {
			delete(m.picked, p)
			m.items = append(m.items[:i], m.items[i+1:]...)
			m.clampCursor()
			return
		}
	}
	m.items = append(m.items, path)
}

func (m *carryModel) clampCursor() {
	if m.cursor > len(m.items)-1 {
		m.cursor = len(m.items) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
}

func (m *carryModel) moveCursor(delta int) {
	m.cursor += delta
	m.clampCursor()
}

// togglePick flips the cursor item in/out of the land subset.
func (m *carryModel) togglePick() {
	if m.cursor < 0 || m.cursor >= len(m.items) {
		return
	}
	p := m.items[m.cursor]
	if m.picked[p] {
		delete(m.picked, p)
		return
	}
	if m.picked == nil {
		m.picked = map[string]bool{}
	}
	m.picked[p] = true
}

// inBucket is the set of paths currently in the bucket. Panel [2] marks these
// with a tick, so a Pick shows up the same way it does in the Carries tab —
// which doubles as multi-select.
func (m carryModel) inBucket() map[string]bool {
	s := make(map[string]bool, len(m.items))
	for _, p := range m.items {
		s[p] = true
	}
	return s
}

// landSet is the set of paths a Land acts on: the picked subset, or everything
// when nothing is picked.
func (m carryModel) landSet() map[string]bool {
	if len(m.picked) > 0 {
		return m.picked
	}
	all := make(map[string]bool, len(m.items))
	for _, p := range m.items {
		all[p] = true
	}
	return all
}

// landItems is landSet in item order — the paths a Land goroutine processes.
func (m carryModel) landItems() []string {
	set := m.landSet()
	var out []string
	for _, p := range m.items {
		if set[p] {
			out = append(out, p)
		}
	}
	return out
}

// removeItem drops path from the bucket (used when a move lands it elsewhere).
func (m *carryModel) removeItem(path string) {
	for i, p := range m.items {
		if p == path {
			delete(m.picked, p)
			m.items = append(m.items[:i], m.items[i+1:]...)
			m.clampCursor()
			return
		}
	}
}

// centeredNote renders a dim message centred both ways in a w×rows box — used
// for panel [4]'s empty states.
func centeredNote(w, rows int, text string) string {
	msg := lipgloss.NewStyle().Foreground(dimColor).Render(text)
	if w < 1 || rows < 1 {
		return msg
	}
	return lipgloss.Place(w, rows, lipgloss.Center, lipgloss.Center, msg)
}

// pickGlyph marks a picked item — a distinct signal from the lavender "in
// bucket" colour (§B: one element, one semantic).
var pickGlyph = string(rune(0xf00c)) // nf-fa-check

func (m carryModel) view(w, rows int, focused bool) string {
	if len(m.items) == 0 {
		return centeredNote(w, rows, "(empty)")
	}
	us := lipgloss.NewStyle().Foreground(userColor)                    // carried = user footprint
	check := lipgloss.NewStyle().Foreground(lipgloss.Color("#a6e3a1")) // picked = green tick
	cursorBg := handColor                                              // focused: current hand
	if !focused {
		cursorBg = userColor
	}
	cur := lipgloss.NewStyle().Foreground(lipgloss.Color(baseHex)).Background(cursorBg)

	var b strings.Builder
	n := min(len(m.items), rows)
	for i := range n {
		p := m.items[i]
		picked := m.picked[p]
		markW := 1
		if picked {
			markW = dispWidth(pickGlyph)
		}
		// Fixed prefix cells: " <mark> <icon> " — reserve them, then fit the full
		// path (home-folded) into the rest, trimmed from the left so the filename
		// stays on screen.
		prefixW := 1 + markW + 1 + dispWidth(iconFile) + 1
		path := truncPathLeft(safeName(shortPath(p)), w-prefixW)
		if focused && i == m.cursor {
			mark := " "
			if picked {
				mark = pickGlyph
			}
			b.WriteString(cur.Render(padDisp(" "+mark+" "+iconFile+" "+path, w)))
		} else {
			mark := " "
			if picked {
				mark = check.Render(pickGlyph)
			}
			b.WriteString(truncate(" "+mark+" "+us.Render(iconFile+" "+path), w))
		}
		if i < n-1 {
			b.WriteByte('\n')
		}
	}
	return b.String()
}

// --- file operations ---

func copyPath(src, dst string) error {
	info, err := os.Lstat(src)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return copyDir(src, dst)
	}
	return copyFile(src, dst, info.Mode())
}

func copyFile(src, dst string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode.Perm())
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

func copyDir(src, dst string) error {
	if err := os.MkdirAll(dst, 0o755); err != nil {
		return err
	}
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if err := copyPath(filepath.Join(src, e.Name()), filepath.Join(dst, e.Name())); err != nil {
			return err
		}
	}
	return nil
}

func movePath(src, dst string) error {
	if err := os.Rename(src, dst); err == nil {
		return nil // same filesystem
	}
	if err := copyPath(src, dst); err != nil { // cross-device fallback
		return err
	}
	return os.RemoveAll(src)
}

// uniquePath appends " copy", " copy 2", … when dst already exists.
func uniquePath(dst string) string {
	if !pathExists(dst) {
		return dst
	}
	ext := filepath.Ext(dst)
	base := strings.TrimSuffix(dst, ext)
	for i := 1; ; i++ {
		cand := base + " copy" + ext
		if i > 1 {
			cand = fmt.Sprintf("%s copy %d%s", base, i, ext)
		}
		if !pathExists(cand) {
			return cand
		}
	}
}

func pathExists(p string) bool {
	_, err := os.Lstat(p)
	return err == nil
}
