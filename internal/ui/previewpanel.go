package ui

import (
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	terminu "github.com/vulcanshen/terminu-go"
)

// previewBody is [2]: the preview of the entry under [1]'s cursor, its details
// (components/layout/panel). The lines depend on the width (an image's data URI
// folds to it), so they are made when it is drawn.
// TODO(lib FL3): terminu's file preview is the finder's only; [2] keeps filu's.
type previewBody struct {
	*terminu.Pager
	app    *App
	item   fileItem
	parent string
	key    string // the entry and width the lines are for
	w      int    // the width it was last drawn at
	model  previewModel
}

func (b *previewBody) Draw(w, h int, focused bool) []string {
	b.w = w
	b.ensure()
	return b.Pager.Draw(w, h, focused)
}

// ensure makes the lines for the entry and the width they are not made for
// yet: at once when the entry changes (what [2] can do depends on them), again
// when it is drawn wider or narrower.
func (b *previewBody) ensure() {
	w := b.w
	if w <= 0 {
		w = 40 // not drawn yet
	}
	key := filepath.Join(b.parent, b.item.name) + "\x00" + itoa(w)
	if key != b.key {
		b.key = key
		b.load(w - 2) // a cell of padding either side (the Pager's)
	}
}

// load makes the lines for an inside w cells wide. A file that cannot be read
// is the panel's error: the fact in Red, a hint naming a key
// (components/layout/panel, loading and errors).
func (b *previewBody) load(w int) {
	b.model = loadPreview(b.item, b.parent, w)
	b.app.preview.Fail = nil
	switch {
	case b.item.name == "":
		b.EmptyFact, b.EmptyHint = "Nothing to preview", "Move the cursor in [1] onto a file or a directory"
	case b.model.kind == previewNone:
		fact := "Cannot read " + safeName(b.item.name)
		if _, err := readCapped(filepath.Join(b.parent, b.item.name), 1); err != nil {
			fact = opFailedText("read "+safeName(b.item.name), err)
		}
		b.app.preview.Fail = &terminu.Failure{Fact: fact, Hint: "Press [1], then [Enter] on it for what filu knows about it"}
	}
	if b.model.kind == previewNone {
		b.SetLines(nil)
		return
	}
	b.SetLines(b.model.lines)
}

// refreshPreview shows the entry now under [1]'s cursor, from its top.
func (a *App) refreshPreview() {
	it, _, ok := a.item()
	if !ok {
		it = fileItem{}
	}
	b := a.pb
	if b == nil {
		return
	}
	if it.name != b.item.name || a.cur().dir != b.parent {
		b.Pager.Key("gg", tea.KeyMsg{})
	}
	b.item, b.parent, b.key = it, a.cur().dir, ""
	if a.preview != nil {
		b.ensure()
	}
}

// previewText is the preview as text to copy: the lines without their numbers,
// an image's data URI in one piece.
func (b *previewBody) previewText() []string {
	if b.model.cont == nil {
		return b.model.body
	}
	var out []string
	for i, l := range b.model.body {
		if i > 0 && b.model.cont[i] {
			out[len(out)-1] += l
			continue
		}
		out = append(out, l)
	}
	return out
}

// hasPreview reports something to view or copy in [2].
func (a *App) hasPreview() bool {
	return a.pb.item.name != "" && a.pb.model.kind != previewNone
}

func (a *App) previewActions() []terminu.Action {
	return []terminu.Action{
		{Key: "Enter", Label: "View", Desc: "the preview in a viewer, to scroll and copy from", Scope: terminu.PanelScope, Hint: "view",
			Disabled: func() bool { return !a.hasPreview() }, Do: a.viewPreview},
		{Key: "y", Label: "yank", Desc: "all of the preview, to the clipboard", Scope: terminu.PanelScope, Hint: "yank",
			Disabled: func() bool { return !a.hasPreview() },
			Do: func() tea.Cmd {
				return copyToClipboardCmd(plainJoin(a.pb.previewText()), "Copied the preview of "+safeName(a.pb.item.name))
			}},
		a.zoomAction(),
	}
}

// viewerCap is how much of a text file the viewer reads: far more than the
// preview's, still bounded.
const viewerCap = 4 << 20

// viewPreview opens the preview in a viewer (components/dialog/note): it
// scrolls, and its selection mode copies a part of it. A text file goes in as
// it is — tabs, control characters, line ends and long lines are the
// viewer's to draw, and a copy gives them back unchanged.
func (a *App) viewPreview() tea.Cmd {
	b := a.pb
	lang := ""
	lines := b.previewText()
	path := filepath.Join(b.parent, b.item.name)
	if b.model.kind == previewText && !strings.HasSuffix(strings.ToLower(b.item.name), ".pdf") {
		if data, err := readCapped(path, viewerCap); err == nil && isText(data) {
			lang = b.item.name // coloured as the preview is
			lines = strings.Split(strings.TrimSuffix(string(data), "\n"), "\n")
		}
	}
	name := safeName(b.item.name)
	all := plainJoin(lines) // filu's own lines carry its colours
	if lang != "" {
		all = strings.Join(lines, "\n") // the file as it is
	}
	return a.t.Viewer(&terminu.Viewer{
		Title: "Preview", Subject: name, Glyph: glyphView, Lines: lines, Lang: lang,
		Keys: []terminu.ViewerKey{{Key: "y", Desc: "copy all", Do: func() tea.Cmd {
			return copyToClipboardCmd(all, "Copied "+name)
		}}},
		Select: &terminu.Selection{Key: "v", Name: "Visual", Copy: func(text string) tea.Cmd {
			return copyToClipboardCmd(text, "Copied the selection")
		}},
	})
}

// plainLines drops the colours of lines.
func plainLines(lines []string) []string {
	out := make([]string, len(lines))
	for i, l := range lines {
		out[i] = terminu.StripStyles(l)
	}
	return out
}

// plainJoin is lines as text to copy.
func plainJoin(lines []string) string { return strings.Join(plainLines(lines), "\n") }
