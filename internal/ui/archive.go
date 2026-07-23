package ui

import (
	"archive/tar"
	"archive/zip"
	"compress/bzip2"
	"compress/gzip"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

const (
	archiveMaxEntries = 500             // stop after this many entries
	archiveByteCap    = 4 * 1024 * 1024 // stop decompressing a stream past this
)

type archEntry struct {
	name  string // full path within the archive, e.g. "src/main.go"
	isDir bool
}

// archiveTree lists a supported archive's contents as a tree, matching the dir
// preview. ok is false when name isn't a supported archive or it can't be read
// (the caller then falls back to a text/hex preview).
func archiveTree(fullPath, name string) (lines []string, ok bool) {
	entries, ok := archiveEntries(fullPath, name)
	if !ok {
		return nil, false
	}
	return renderArchiveTree(entries), true
}

func archiveEntries(fullPath, name string) ([]archEntry, bool) {
	lower := strings.ToLower(name)
	switch {
	case strings.HasSuffix(lower, ".zip"):
		return zipEntries(fullPath)
	case strings.HasSuffix(lower, ".tar"):
		return tarEntries(fullPath, "")
	case strings.HasSuffix(lower, ".tar.gz"), strings.HasSuffix(lower, ".tgz"):
		return tarEntries(fullPath, "gz")
	case strings.HasSuffix(lower, ".tar.bz2"), strings.HasSuffix(lower, ".tbz"), strings.HasSuffix(lower, ".tbz2"):
		return tarEntries(fullPath, "bz2")
	case strings.HasSuffix(lower, ".gz"): // single-file gzip
		return []archEntry{{name: name[:len(name)-3]}}, true
	case strings.HasSuffix(lower, ".bz2"): // single-file bzip2
		return []archEntry{{name: name[:len(name)-4]}}, true
	}
	return nil, false
}

func zipEntries(p string) ([]archEntry, bool) {
	r, err := zip.OpenReader(p)
	if err != nil {
		return nil, false
	}
	defer r.Close()
	out := make([]archEntry, 0, len(r.File))
	for _, f := range r.File {
		if len(out) >= archiveMaxEntries {
			break
		}
		out = append(out, archEntry{name: f.Name, isDir: f.FileInfo().IsDir()})
	}
	return out, true
}

// tarEntries streams a (optionally compressed) tar, bounded by archiveByteCap so
// a huge archive can't stall the preview; a partial read still yields a tree.
func tarEntries(p, comp string) ([]archEntry, bool) {
	f, err := os.Open(p)
	if err != nil {
		return nil, false
	}
	defer f.Close()
	var r io.Reader = io.LimitReader(f, archiveByteCap)
	switch comp {
	case "gz":
		gz, err := gzip.NewReader(r)
		if err != nil {
			return nil, false
		}
		defer gz.Close()
		r = gz
	case "bz2":
		r = bzip2.NewReader(r)
	}
	tr := tar.NewReader(r)
	var out []archEntry
	for len(out) < archiveMaxEntries {
		hdr, err := tr.Next()
		if err != nil {
			break // EOF, cap hit mid-stream, or corrupt — show what we have
		}
		out = append(out, archEntry{name: hdr.Name, isDir: hdr.FileInfo().IsDir()})
	}
	if len(out) == 0 {
		return nil, false
	}
	return out, true
}

// atNode is a node in the tree built from flat archive paths.
type atNode struct {
	children map[string]*atNode
	order    []string // child insertion order
	isDir    bool
}

func (n *atNode) child(name string) *atNode {
	if n.children == nil {
		n.children = map[string]*atNode{}
	}
	c, ok := n.children[name]
	if !ok {
		c = &atNode{}
		n.children[name] = c
		n.order = append(n.order, name)
	}
	return c
}

// renderArchiveTree builds a tree from the flat entry paths and renders it with
// the same branch glyphs / icons / colours as the directory preview.
func renderArchiveTree(entries []archEntry) []string {
	root := &atNode{}
	for _, e := range entries {
		var parts []string
		for p := range strings.SplitSeq(e.name, "/") {
			if p != "" && p != "." {
				parts = append(parts, p)
			}
		}
		if len(parts) == 0 {
			continue
		}
		cur := root
		for i, part := range parts {
			cur = cur.child(part)
			if i < len(parts)-1 {
				cur.isDir = true // intermediate segments are directories
			}
		}
		if e.isDir {
			cur.isDir = true
		}
	}

	var lines []string
	var walk func(n *atNode, prefix string, depth int)
	walk = func(n *atNode, prefix string, depth int) {
		names := append([]string(nil), n.order...)
		sort.Slice(names, func(i, j int) bool {
			ci, cj := n.children[names[i]], n.children[names[j]]
			if ci.isDir != cj.isDir {
				return ci.isDir // directories first
			}
			return names[i] < names[j]
		})
		for i, name := range names {
			if len(lines) >= treeMaxLines {
				return
			}
			c := n.children[name]
			icon := iconFile
			if c.isDir {
				icon = iconDir
			}
			label := lipgloss.NewStyle().
				Foreground(fileColor(fileItem{name: name, isDir: c.isDir})).
				Render(icon + " " + safeName(name))
			if depth == 1 { // top level: a plain list, no branch guide
				lines = append(lines, " "+label)
				if c.isDir {
					walk(c, "  ", depth+1)
				}
				continue
			}
			branch, ext := treeTee, treeBar
			if i == len(names)-1 {
				branch, ext = treeEnd, treeGap
			}
			lines = append(lines, prefix+branch+label)
			if c.isDir {
				walk(c, prefix+ext, depth+1)
			}
		}
	}
	walk(root, "", 1)
	if len(lines) == 0 {
		return []string{lipgloss.NewStyle().Foreground(dimColor).Render("(empty archive)")}
	}
	return lines
}
