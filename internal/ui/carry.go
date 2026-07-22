package ui

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// carryModel is panel [4]'s bucket: files picked up with Carry, dropped with
// Land (which decides copy vs move). It also records completed lands as history.
type carryModel struct {
	items   []string       // full source paths
	history []historyEntry // completed lands, newest first
}

type historyEntry struct {
	action string // "cp" / "mv"
	count  int
	dest   string
	when   time.Time
}

func (m *carryModel) toggle(path string) {
	for i, p := range m.items {
		if p == path {
			m.items = append(m.items[:i], m.items[i+1:]...)
			return
		}
	}
	m.items = append(m.items, path)
}

// land copies (move=false) or moves (move=true) every carried item into
// destDir. Copies stay in the bucket (paste again elsewhere); moved items
// leave it. Errors are swallowed for now (TODO: surface + progress tab).
func (m *carryModel) land(destDir string, move bool) {
	var remaining []string
	done := 0
	for _, src := range m.items {
		dst := uniquePath(filepath.Join(destDir, filepath.Base(src)))
		var err error
		if move {
			err = movePath(src, dst)
		} else {
			err = copyPath(src, dst)
		}
		if err == nil {
			done++
		}
		if err != nil || !move {
			remaining = append(remaining, src)
		}
	}
	m.items = remaining
	if done > 0 {
		act := "cp"
		if move {
			act = "mv"
		}
		m.history = append([]historyEntry{{act, done, filepath.Base(destDir), time.Now()}}, m.history...)
	}
}

func (m carryModel) view(w, rows int, _ bool) string {
	if len(m.items) == 0 {
		return lipgloss.NewStyle().Foreground(dimColor).Render("empty")
	}
	lines := make([]string, len(m.items))
	for i, p := range m.items {
		lines[i] = truncate(" "+iconFile+"  "+filepath.Base(p), w)
	}
	return renderLines(lines, w, rows)
}

func (m carryModel) historyView(w, rows int) string {
	if len(m.history) == 0 {
		return lipgloss.NewStyle().Foreground(dimColor).Render("(no history)")
	}
	lines := make([]string, len(m.history))
	for i, h := range m.history {
		lines[i] = truncate(fmt.Sprintf("%s %d %s %s", h.action, h.count, h.dest, h.when.Format("15:04")), w)
	}
	return renderLines(lines, w, rows)
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
