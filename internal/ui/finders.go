package ui

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	terminu "github.com/vulcanshen/terminu-go"
)

// finderMsg is a batch of a finder's results from its goroutine; gen says
// which finder they are for.
type finderMsg struct {
	gen   int
	items []terminu.FinderItem
	lines map[string]int // a content search's first matching line of each file
	done  bool
}

func (a *App) waitFind() tea.Cmd {
	ch := a.finds
	return func() tea.Msg { return <-ch }
}

// handleFind adds a batch to the finder open now; one for a finder already
// gone is dropped.
func (a *App) handleFind(msg finderMsg) {
	if msg.gen != a.findGen || a.finder == nil {
		return
	}
	a.finder.Items = append(a.finder.Items, msg.items...)
	for k, v := range msg.lines {
		a.hits[k] = v
	}
	if msg.done {
		a.finder.Loading = false
	}
}

// openFinder opens f and starts filling it.
func (a *App) openFinder(f *terminu.Finder, fill func(gen int, ch chan<- finderMsg)) tea.Cmd {
	a.findGen++
	a.finder = f
	a.hits = map[string]int{}
	f.Loading = true
	go fill(a.findGen, a.finds)
	return a.t.Find(f)
}

// searchMenu is / on [1]: by name or by content, under the tab's directory.
func (a *App) searchMenu() tea.Cmd {
	return a.t.Menu("Search", terminu.MenuSection{Actions: []terminu.Action{
		{Key: "f", Label: "File names", Desc: "every file and directory under here, by name", Do: a.nameFinder},
		{Key: "c", Label: "Contents", Desc: "the files under here that hold some text (ripgrep)", Do: a.contentSearch},
	}})
}

// nameFinder lists everything under the tab's directory as fd finds it; the
// typing row narrows it.
// TODO(lib FL2): a query starting with / or ~/ searched from that path; the
// finder cannot tell filu what is typed.
func (a *App) nameFinder() tea.Cmd {
	root := a.cur().dir
	f := &terminu.Finder{Title: "Search", Subject: safeName(shortPath(root)), Glyph: glyphSearch, Verb: "open", Files: true,
		Done: func(it terminu.FinderItem) tea.Cmd { return a.reveal(it.ID) }}
	return a.openFinder(f, func(gen int, ch chan<- finderMsg) { streamFinder(gen, root, false, ch) })
}

// gotoMenu is go on [1]: a favorite, or any directory under the home.
func (a *App) gotoMenu() tea.Cmd {
	return a.t.Menu("Goto", terminu.MenuSection{Actions: []terminu.Action{
		{Key: "f", Label: "Favorite", Desc: "one of your starred directories", Stay: true,
			Disabled: func() bool { return len(a.favs.pinned) == 0 },
			Do:       func() tea.Cmd { return a.favoriteMenu(false) }},
		{Key: "/", Label: "Search", Desc: "any directory under your home, hidden ones too", Do: func() tea.Cmd { return a.gotoFinder(false) }},
	}})
}

// favoriteMenu lists the favorites: the tab goes there, or a new tab opens
// there.
func (a *App) favoriteMenu(newTab bool) tea.Cmd {
	var acts []terminu.Action
	for i, f := range a.favs.pinned {
		dir := f.path
		key := ""
		if i < 9 {
			key = strconv.Itoa(i + 1)
		}
		acts = append(acts, terminu.Action{Key: key, Label: safeName(filepath.Base(dir)), Desc: safeName(shortPath(dir)),
			Do: func() tea.Cmd {
				if newTab {
					return a.openTab(dir, "")
				}
				return a.cd(dir, "")
			}})
	}
	return a.t.Menu("Favorites", terminu.MenuSection{Actions: acts})
}

// gotoFinder lists every directory under the home, hidden ones too: the tab
// (or a new one) goes to the one chosen.
func (a *App) gotoFinder(newTab bool) tea.Cmd {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		home = a.cur().dir
	}
	title := "Goto"
	if newTab {
		title = "New tab"
	}
	f := &terminu.Finder{Title: title, Subject: safeName(shortPath(home)), Glyph: glyphGoto, Verb: "go", Files: true,
		Done: func(it terminu.FinderItem) tea.Cmd {
			if newTab {
				return a.openTab(it.ID, "")
			}
			return a.cd(it.ID, "")
		}}
	return a.openFinder(f, func(gen int, ch chan<- finderMsg) { streamFinder(gen, home, true, ch) })
}

// contentSearch asks what to look for, then lists the files under the tab's
// directory that hold it, each previewed at its first match.
// TODO(lib FL2): the finder cannot tell filu what is typed, so the text is
// asked for first instead of searched as it is typed.
func (a *App) contentSearch() tea.Cmd {
	if _, err := lookPath("rg"); err != nil {
		return a.t.Error("Search failed", "Cannot search file contents: ripgrep (rg) is not installed")
	}
	root := a.cur().dir
	where := safeName(shortPath(root))
	return a.t.Text(terminu.TextInput{Title: "Search contents", Subject: where, Glyph: glyphSearch, Name: "The text",
		Prompt: "The text to find in the files under " + where, Verb: "search",
		Check: func(v string) error {
			if v == "" {
				return fmt.Errorf("Type the text to find first")
			}
			return nil
		},
		Done: func(q string) tea.Cmd {
			f := &terminu.Finder{Title: "Contents", Subject: q, Glyph: glyphSearch, Verb: "open",
				Done: func(it terminu.FinderItem) tea.Cmd { return a.reveal(it.ID) }}
			f.Preview = func(it terminu.FinderItem) (string, []string) { return it.Text, a.hitPreview(it.ID) }
			return a.openFinder(f, func(gen int, ch chan<- finderMsg) { streamGrep(gen, root, q, ch) })
		}})
}

// hitPreview is a file from a few lines above its first match, numbered, as
// components/input/file-picker previews a text file: secrets kept out when
// hide_secrets_in_preview is on, control characters named in Red, a file
// that cannot be read in grey.
// TODO(lib FL16): a finder's file preview cannot start at a line, nor show
// the file's facts under it from here.
func (a *App) hitPreview(path string) []string {
	grey := func(s string) []string { return []string{terminu.Fg(terminu.Overlay0).Render(s)} }
	st, err := os.Stat(path)
	if err != nil {
		return grey("(unreadable)")
	}
	data, err := readCapped(path, previewCap)
	if err != nil {
		return grey("(unreadable)")
	}
	if a.t.HideSecretsInPreview {
		first, _, _ := strings.Cut(string(data), "\n")
		first = strings.TrimRight(first, "\r")
		if strings.HasPrefix(first, "-----BEGIN") && strings.Contains(first, "PRIVATE KEY") || strings.HasPrefix(first, "PuTTY-User-Key-File-") {
			return []string{terminu.Fg(terminu.Text).Render(first), terminu.Fg(terminu.Overlay0).Render("(sensitive content not shown)")}
		}
		if st.Mode().Perm()&0o044 == 0 {
			return grey("(sensitive content not shown)")
		}
	}
	raw := strings.Split(string(data), "\n")
	lines := make([]string, len(raw))
	control := false
	for i, l := range raw {
		lines[i] = strings.ReplaceAll(strings.TrimSuffix(l, "\r"), "\t", strings.Repeat(" ", inputTabWidth))
		control = control || strings.ContainsFunc(lines[i], isControl)
	}
	if control {
		for i, l := range lines {
			lines[i] = namedControls(l)
		}
	} else if hl, ok := highlight(filepath.Base(path), strings.Join(lines, "\n")); ok {
		lines = hl
	}
	from := max(a.hits[path]-4, 0)
	numW := max(len(strconv.Itoa(len(lines))), 3)
	var out []string
	for i := from; i < len(lines); i++ {
		out = append(out, terminu.Fg(terminu.Overlay0).Render(fmt.Sprintf("%*d", numW, i+1))+" "+lines[i])
	}
	return out
}

// isControl reports a control character: C0, DEL, C1.
func isControl(r rune) bool { return r < 0x20 || r == 0x7f || r >= 0x80 && r < 0xa0 }

// namedControls draws a line's control characters as their names in Red
// (\r, \x07), the rest in Text (components/input/README, pasted control
// characters).
func namedControls(l string) string {
	var b strings.Builder
	for _, r := range l {
		switch {
		case r == '\r':
			b.WriteString(terminu.Fg(terminu.Red).Render(`\r`))
		case r == '\n':
			b.WriteString(terminu.Fg(terminu.Red).Render(`\n`))
		case isControl(r):
			b.WriteString(terminu.Fg(terminu.Red).Render(fmt.Sprintf(`\x%02x`, r)))
		default:
			b.WriteString(terminu.Fg(terminu.Text).Render(string(r)))
		}
	}
	return b.String()
}

// streamFinder sends what fd finds under root to ch in batches (directories
// only for Goto, hidden ones too), up to finderCap; without fd a Go walk does
// it in one go. It always ends with a done batch.
func streamFinder(gen int, root string, dirsOnly bool, ch chan<- finderMsg) {
	var batch []terminu.FinderItem
	send := func(done bool) {
		ch <- finderMsg{gen: gen, items: batch, done: done}
		batch = nil
	}
	add := func(rel string) {
		p := filepath.Join(root, rel)
		fi, err := os.Lstat(p)
		if err != nil {
			return
		}
		it := fileItem{name: filepath.Base(p), isDir: fi.IsDir(), isLink: fi.Mode()&os.ModeSymlink != 0,
			isExec: !fi.IsDir() && fi.Mode()&0o111 != 0}
		batch = append(batch, terminu.FinderItem{ID: p, Text: safeName(rel), Icon: fileIcon(it), IconColor: terminu.Hex(string(fileColor(it)))})
		if len(batch) >= streamBatch {
			send(false)
		}
	}
	if _, err := exec.LookPath("fd"); err != nil {
		for _, rel := range walkDirFiles(root, dirsOnly, 0) {
			add(rel)
		}
		send(true)
		return
	}
	args := []string{"--type", "f", "--type", "d", "--hidden"}
	if dirsOnly {
		args = []string{"--type", "d", "--hidden"}
	}
	args = append(args, "--strip-cwd-prefix")
	for _, ig := range finderIgnoreDirs {
		args = append(args, "--exclude", ig)
	}
	lines(root, 15*time.Second, "fd", args, func(line string) bool {
		add(line)
		return true
	})
	send(true)
}

// streamGrep sends the files under root that hold q to ch, one per file with
// its first matching line, in the order ripgrep finds them.
func streamGrep(gen int, root, q string, ch chan<- finderMsg) {
	var batch []terminu.FinderItem
	hits := map[string]int{}
	send := func(done bool) {
		ch <- finderMsg{gen: gen, items: batch, lines: hits, done: done}
		batch, hits = nil, map[string]int{}
	}
	seen := map[string]bool{}
	args := []string{"--line-number", "--no-heading", "--color", "never", "--smart-case", "--", q}
	lines(root, 30*time.Second, "rg", args, func(line string) bool {
		rel, n, ok := parseGrepLine(line)
		if !ok || seen[rel] {
			return true
		}
		seen[rel] = true
		p := filepath.Join(root, rel)
		it := fileItem{name: filepath.Base(p)}
		batch = append(batch, terminu.FinderItem{ID: p, Text: safeName(rel), Icon: fileIcon(it), IconColor: terminu.Hex(string(fileColor(it)))})
		hits[p] = n
		if len(batch) >= streamBatch {
			send(false)
		}
		return true
	})
	send(true)
}

// lines runs name with args in dir and hands each line of its output to each,
// up to finderCap lines or until each says to stop; then the program is ended.
func lines(dir string, limit time.Duration, name string, args []string, each func(string) bool) {
	ctx, cancel := context.WithTimeout(context.Background(), limit)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	out, err := cmd.StdoutPipe()
	if err != nil {
		return
	}
	if err := cmd.Start(); err != nil {
		return
	}
	sc := bufio.NewScanner(out)
	sc.Buffer(make([]byte, 64*1024), 1<<20)
	n := 0
	for sc.Scan() {
		if l := sc.Text(); l != "" {
			n++
			if !each(l) || n >= finderCap {
				break
			}
		}
	}
	cancel()
	_ = cmd.Wait()
}

// parseGrepLine splits an rg "--no-heading --line-number" line: "path:line:text".
func parseGrepLine(s string) (path string, line int, ok bool) {
	parts := strings.SplitN(s, ":", 3)
	if len(parts) < 3 {
		return "", 0, false
	}
	n, err := strconv.Atoi(parts[1])
	if err != nil {
		return "", 0, false
	}
	return parts[0], n, true
}

func walkDirFiles(root string, dirsOnly bool, maxDepth int) []string {
	ignore := make(map[string]bool, len(finderIgnoreDirs))
	for _, ig := range finderIgnoreDirs {
		if !strings.Contains(ig, "/") { // path-glob entries (e.g. go/pkg) are honoured by fd only
			ignore[ig] = true
		}
	}
	var out []string
	_ = filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if p == root {
			return nil
		}
		if d.IsDir() && ignore[d.Name()] {
			return filepath.SkipDir
		}
		if dirsOnly && !d.IsDir() {
			return nil // Goto lists directories only (hidden dirs included, matching fd)
		}
		rel, e := filepath.Rel(root, p)
		if e != nil {
			return nil
		}
		out = append(out, rel)
		if maxDepth > 0 && d.IsDir() && strings.Count(rel, string(os.PathSeparator))+1 >= maxDepth {
			return filepath.SkipDir // don't descend past maxDepth (1 = direct children only)
		}
		if len(out) >= finderCap {
			return filepath.SkipAll
		}
		return nil
	})
	return out
}

// lookPath finds a helper program; a variable so that tests can pretend.
var lookPath = exec.LookPath

// streamBatch is how many results are sent at a time: few enough that the
// first ones show almost at once.
const streamBatch = 256
