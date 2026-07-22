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
	lines []string // dir tree / text / hex
	note  string   // empty / unreadable
}

func loadPreview(it fileItem, parent string) previewModel {
	if it.name == "" {
		return previewModel{note: "(no selection)"}
	}
	full := filepath.Join(parent, it.name)
	if it.isDir {
		return previewModel{kind: previewDir, lines: treeLines(full, 3)}
	}
	data, err := readCapped(full, previewCap)
	if err != nil {
		return previewModel{note: "(unreadable)"}
	}
	if isText(data) {
		return previewModel{kind: previewText, lines: sanitizeLines(strings.Split(string(data), "\n"))}
	}
	return previewModel{kind: previewBinary, lines: hexDump(data)}
}

func sanitizeLines(lines []string) []string {
	for i, l := range lines {
		lines[i] = sanitizeLine(l)
	}
	return lines
}

// sanitizeLine makes a text line safe to draw: tabs → spaces, CR dropped, other
// control bytes (incl. ESC — would otherwise corrupt the whole terminal) → space.
// Capped so a pathological long line can't blow up width math.
func sanitizeLine(s string) string {
	const maxRunes = 2000
	var b strings.Builder
	n := 0
	for _, r := range s {
		if n >= maxRunes {
			break
		}
		switch {
		case r == '\t':
			b.WriteString("    ")
		case r == '\r':
			continue
		case r < 0x20 || r == 0x7f:
			b.WriteByte(' ')
		default:
			b.WriteRune(r)
		}
		n++
	}
	return b.String()
}

// contentLines returns the preview's display lines (a note when empty/unreadable).
func (p previewModel) contentLines() []string {
	if p.kind == previewNone {
		return []string{lipgloss.NewStyle().Foreground(dimColor).Render(p.note)}
	}
	return p.lines
}

// tree branch pieces (rune values so no box glyph sits in source).
var (
	treeTee = string(rune(0x251c)) + string(rune(0x2500)) + " " // "├─ "
	treeEnd = string(rune(0x2514)) + string(rune(0x2500)) + " " // "└─ "
	treeBar = string(rune(0x2502)) + "  "                       // "│  "
	treeGap = "   "
)

const treeMaxLines = 200

// treeLines renders a directory as a tree up to maxDepth levels, capped at
// treeMaxLines rows.
func treeLines(root string, maxDepth int) []string {
	var lines []string
	var walk func(dir, prefix string, depth int)
	walk = func(dir, prefix string, depth int) {
		items, _ := readEntries(dir, false)
		for i, it := range items {
			if len(lines) >= treeMaxLines {
				return
			}
			icon := iconFile
			if it.isDir {
				icon = iconDir
			}
			label := icon + " " + it.name
			if it.isDir {
				label = lipgloss.NewStyle().Foreground(dirColor).Render(label) // sky
			}
			if depth == 1 { // top level: a plain list, no branch guide
				lines = append(lines, " "+label)
				if it.isDir && depth < maxDepth {
					walk(filepath.Join(dir, it.name), "  ", depth+1)
				}
				continue
			}
			branch, ext := treeTee, treeBar
			if i == len(items)-1 {
				branch, ext = treeEnd, treeGap
			}
			lines = append(lines, prefix+branch+label)
			if it.isDir && depth < maxDepth {
				walk(filepath.Join(dir, it.name), prefix+ext, depth+1)
			}
		}
	}
	walk(root, "", 1)
	if len(lines) == 0 {
		return []string{lipgloss.NewStyle().Foreground(dimColor).Render("(empty)")}
	}
	return lines
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

// renderLinesFrom renders up to rows lines starting at offset, each clipped to w.
func renderLinesFrom(lines []string, offset, w, rows int) string {
	if offset < 0 {
		offset = 0
	}
	end := min(offset+rows, len(lines))
	var b strings.Builder
	for i := offset; i < end; i++ {
		b.WriteString(truncate(lines[i], w))
		if i < end-1 {
			b.WriteByte('\n')
		}
	}
	return b.String()
}

func renderLines(lines []string, w, rows int) string {
	return renderLinesFrom(lines, 0, w, rows)
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
