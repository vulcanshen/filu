package ui

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"
)

const previewCap = 64 * 1024 // read at most this many bytes for a preview

type previewKind int

const (
	previewNone previewKind = iota
	previewDir
	previewText
	previewBinary
)

// previewModel is panel [3]'s Preview tab: a classified view of the cursor item.
// image / archive classes come later; for now dir / text / binary.
type previewModel struct {
	kind  previewKind
	dir   listModel // when kind == previewDir
	lines []string  // when kind == previewText / previewBinary
	note  string    // empty / unreadable
}

func loadPreview(it fileItem, parent string) previewModel {
	if it.name == "" {
		return previewModel{note: "(無選取)"}
	}
	full := filepath.Join(parent, it.name)
	if it.isDir {
		return previewModel{kind: previewDir, dir: newList(full)}
	}
	data, err := readCapped(full, previewCap)
	if err != nil {
		return previewModel{note: "(無法讀取)"}
	}
	if isText(data) {
		return previewModel{kind: previewText, lines: strings.Split(string(data), "\n")}
	}
	return previewModel{kind: previewBinary, lines: hexDump(data)}
}

func (p previewModel) view(w, rows int) string {
	switch p.kind {
	case previewDir:
		return p.dir.view(w, rows, false) // read-only, unfocused
	case previewText, previewBinary:
		return renderLines(p.lines, w, rows)
	default:
		return lipgloss.NewStyle().Foreground(dimColor).Render(p.note)
	}
}

func readCapped(path string, n int) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return io.ReadAll(io.LimitReader(f, int64(n)))
}

// isText treats a sample as text when it has no NUL byte and is valid UTF-8.
func isText(data []byte) bool {
	if bytes.IndexByte(data, 0) >= 0 {
		return false
	}
	return utf8.Valid(data)
}

func renderLines(lines []string, w, rows int) string {
	var b strings.Builder
	n := min(len(lines), rows)
	for i := range n {
		b.WriteString(truncate(lines[i], w))
		if i < n-1 {
			b.WriteByte('\n')
		}
	}
	return b.String()
}

// hexDump renders data xxd-style: offset, 16 hex bytes, ASCII gutter.
func hexDump(data []byte) []string {
	var out []string
	for off := 0; off < len(data); off += 16 {
		row := data[off:min(off+16, len(data))]
		var hex, asc strings.Builder
		for i := range 16 {
			if i < len(row) {
				fmt.Fprintf(&hex, "%02x ", row[i])
				if c := row[i]; c >= 0x20 && c < 0x7f {
					asc.WriteByte(c)
				} else {
					asc.WriteByte('.')
				}
			} else {
				hex.WriteString("   ")
			}
		}
		out = append(out, fmt.Sprintf("%08x  %s|%s|", off, hex.String(), asc.String()))
	}
	return out
}
