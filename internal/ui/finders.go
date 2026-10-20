package ui

import (
	"bufio"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	terminu "github.com/vulcanshen/terminu-go"
)

// finderMsg is a batch of a search's results from its goroutine; gen says
// which search they are for.
type finderMsg struct {
	gen   int
	items []terminu.FinderItem
	done  bool
}

func (a *App) waitFind() tea.Cmd {
	ch := a.finds
	return func() tea.Msg { return <-ch }
}

// handleFind adds a batch to what the search running has found, and the ones
// that match what is typed to the finder; one for a search already over is
// dropped.
func (a *App) handleFind(msg finderMsg) {
	if msg.gen != a.findGen || a.finder == nil {
		return
	}
	a.found = append(a.found, msg.items...)
	for _, it := range msg.items {
		if a.keep == nil || a.keep(it) {
			a.finder.Items = append(a.finder.Items, it)
		}
	}
	if msg.done {
		a.finder.Loading = false
	}
}

// openFinder opens f: the searches started from now on fill it.
func (a *App) openFinder(f *terminu.Finder) tea.Cmd {
	a.finder, a.keep = f, nil
	return a.t.Find(f)
}

// search runs fill on a goroutine for the finder open now, in place of the
// search running: what that one found goes.
func (a *App) search(fill func(ctx context.Context, gen int, ch chan<- finderMsg)) {
	if a.findStop != nil {
		a.findStop()
	}
	a.findGen++
	ctx, stop := context.WithCancel(context.Background())
	a.findStop = stop
	a.found, a.finder.Items, a.finder.Loading = nil, nil, true
	go fill(ctx, a.findGen, a.finds)
}

// refilter shows again the ones found that match what is typed.
func (a *App) refilter() {
	var items []terminu.FinderItem
	for _, it := range a.found {
		if a.keep == nil || a.keep(it) {
			items = append(items, it)
		}
	}
	a.finder.Items = items
}

// revealFound goes to what a finder found. Going there clears the menus the
// finder was opened from (tdp T1: a drill-down clears its source).
func (a *App) revealFound(it terminu.FinderItem) tea.Cmd {
	return tea.Batch(a.t.CloseAll(), a.reveal(it.ID))
}

// searchMenu is / on [1]: by name or by content, under the tab's directory.
func (a *App) searchMenu() tea.Cmd {
	return a.t.Menu("Search", terminu.MenuSection{Actions: []terminu.Action{
		{Key: "f", Label: "File names", Desc: "every file and directory under here, by name", Do: a.nameFinder},
		{Key: "c", Label: "Contents", Desc: "the files under here that hold some text (ripgrep)", Do: a.contentSearch},
	}})
}

// nameFinder lists everything under the tab's directory as fd finds it; the
// typing row narrows it, or names a path of its own (pathFinder).
func (a *App) nameFinder() tea.Cmd {
	root := a.cur().dir
	f := &terminu.Finder{Title: "Search", Subject: shortPath(root), Glyph: glyphSearch, Verb: "open", Files: true,
		Done: a.revealFound}
	return a.pathFinder(f, root, false)
}

// gotoMenu is go on [1]: a favorite, or any directory under the home.
func (a *App) gotoMenu() tea.Cmd {
	return a.t.Menu("Goto", terminu.MenuSection{Actions: []terminu.Action{
		{Key: "f", Label: "Favorite", Desc: "one of your starred directories",
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
		acts = append(acts, terminu.Action{Key: key, Label: filepath.Base(dir), Desc: shortPath(dir),
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
	f := &terminu.Finder{Title: title, Subject: shortPath(home), Glyph: glyphGoto, Verb: "go", Files: true,
		Done: func(it terminu.FinderItem) tea.Cmd {
			if newTab {
				return tea.Batch(a.t.CloseAll(), a.openTab(it.ID, ""))
			}
			return tea.Batch(a.t.CloseAll(), a.cd(it.ID, ""))
		}}
	return a.pathFinder(f, home, true)
}

// pathFinder opens f on everything under base as fd finds it (directories
// only for Goto), narrowed by what is typed. What is typed starting with / or
// ~/ is a path: f then lists what is under the deepest directory it names, a
// few levels down, matched by the rest of it — outside base too.
func (a *App) pathFinder(f *terminu.Finder, base string, dirsOnly bool) tea.Cmd {
	root, depth := base, 0
	scan := func(r string, d int) {
		root, depth = r, d
		a.search(func(ctx context.Context, gen int, ch chan<- finderMsg) {
			streamFinder(ctx, gen, r, d, r != base, dirsOnly, ch)
		})
	}
	f.Search = func(typed string) tea.Cmd {
		r, d, words := base, 0, typed
		if q := expandHome(typed); strings.HasPrefix(q, "/") {
			r, d = absAnchor(q)
			words = strings.TrimPrefix(strings.TrimPrefix(q, r), "/")
		}
		a.keep = func(it terminu.FinderItem) bool { return terminu.Matches(it.Match, words) }
		if r != root || d != depth {
			scan(r, d)
		} else {
			a.refilter()
		}
		return nil
	}
	cmd := a.openFinder(f)
	scan(base, 0)
	return cmd
}

// absAnchorDepth is how far under the directory a typed path names its search
// goes: fd over / three levels down takes about a second.
const absAnchorDepth = 3

// absAnchor is where the search for a typed path lists from: the deepest
// directory the path names, its last part always left to match ("/usr/lo"
// lists under /usr, matching "lo"). Ending in / it lists that directory alone;
// otherwise absAnchorDepth levels down.
func absAnchor(query string) (root string, depth int) {
	root = "/"
	rest := strings.TrimPrefix(query, "/")
	for {
		seg, tail, ok := strings.Cut(rest, "/")
		if !ok {
			break
		}
		cand := filepath.Join(root, seg)
		if fi, err := os.Stat(cand); err != nil || !fi.IsDir() {
			break
		}
		root, rest = cand, tail
	}
	if rest == "" {
		return root, 1
	}
	return root, absAnchorDepth
}

// expandHome is what is typed with a leading ~ as the home directory.
func expandHome(typed string) string {
	if typed == "~" || strings.HasPrefix(typed, "~/") {
		if home, err := os.UserHomeDir(); err == nil && home != "" {
			return home + strings.TrimPrefix(typed, "~")
		}
	}
	return typed
}

// grepPause is how long the content search waits after a key before it
// searches, so that a word typed quickly is searched for once.
const grepPause = 150 * time.Millisecond

// contentSearch lists the files under the tab's directory that hold what is
// typed, as ripgrep finds them, each previewed from its first match; with
// nothing typed, everything under it, as File names does.
func (a *App) contentSearch() tea.Cmd {
	if _, err := lookPath("rg"); err != nil {
		return a.t.Error("Search failed", "Cannot search file contents: ripgrep (rg) is not installed")
	}
	root := a.cur().dir
	f := &terminu.Finder{Title: "Contents", Subject: shortPath(root), Glyph: glyphSearch, Verb: "open", Files: true,
		Done: a.revealFound}
	f.Search = func(typed string) tea.Cmd {
		if strings.TrimSpace(typed) == "" {
			a.search(func(ctx context.Context, gen int, ch chan<- finderMsg) {
				streamFinder(ctx, gen, root, 0, false, false, ch)
			})
			return nil
		}
		a.search(func(ctx context.Context, gen int, ch chan<- finderMsg) {
			select {
			case <-time.After(grepPause):
			case <-ctx.Done():
				return
			}
			streamGrep(ctx, gen, root, typed, ch)
		})
		return nil
	}
	cmd := a.openFinder(f)
	f.Search("")
	return cmd
}

// send hands a batch to the main loop, unless the search is over.
func send(ctx context.Context, ch chan<- finderMsg, msg finderMsg) {
	select {
	case ch <- msg:
	case <-ctx.Done():
	}
}

// streamFinder sends what fd finds under root to ch in batches (directories
// only for Goto, hidden ones too), depth levels down when depth is set, up to
// finderCap; without fd a Go walk does it in one go. Each is shown by its path
// under root, or with abs by its whole path, and matched by its path under
// root. It always ends with a done batch.
func streamFinder(ctx context.Context, gen int, root string, depth int, abs, dirsOnly bool, ch chan<- finderMsg) {
	var batch []terminu.FinderItem
	flush := func(done bool) {
		send(ctx, ch, finderMsg{gen: gen, items: batch, done: done})
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
		text := rel
		if abs {
			text = shortPath(p)
		}
		batch = append(batch, terminu.FinderItem{ID: p, Text: text, Match: rel, Icon: fileIcon(it), IconColor: terminu.Hex(string(fileColor(it)))})
		if len(batch) >= streamBatch {
			flush(false)
		}
	}
	if _, err := exec.LookPath("fd"); err != nil {
		for _, rel := range walkDirFiles(root, dirsOnly, depth) {
			add(rel)
		}
		flush(true)
		return
	}
	args := []string{"--type", "f", "--type", "d", "--hidden"}
	if dirsOnly {
		args = []string{"--type", "d", "--hidden"}
	}
	if depth > 0 {
		args = append(args, "--max-depth", strconv.Itoa(depth))
	}
	args = append(args, "--strip-cwd-prefix")
	for _, ig := range finderIgnoreDirs {
		args = append(args, "--exclude", ig)
	}
	lines(ctx, root, 15*time.Second, "fd", args, func(line string) bool {
		add(line)
		return true
	})
	flush(true)
}

// streamGrep sends the files under root that hold q to ch, one per file with
// its first matching line, in the order ripgrep finds them.
func streamGrep(ctx context.Context, gen int, root, q string, ch chan<- finderMsg) {
	var batch []terminu.FinderItem
	flush := func(done bool) {
		send(ctx, ch, finderMsg{gen: gen, items: batch, done: done})
		batch = nil
	}
	seen := map[string]bool{}
	args := []string{"--line-number", "--no-heading", "--color", "never", "--smart-case", "--", q}
	lines(ctx, root, 30*time.Second, "rg", args, func(line string) bool {
		rel, n, ok := parseGrepLine(line)
		if !ok || seen[rel] {
			return true
		}
		seen[rel] = true
		p := filepath.Join(root, rel)
		it := fileItem{name: filepath.Base(p)}
		batch = append(batch, terminu.FinderItem{ID: p, Text: rel, Icon: fileIcon(it), IconColor: terminu.Hex(string(fileColor(it))), Line: n})
		if len(batch) >= streamBatch {
			flush(false)
		}
		return true
	})
	flush(true)
}

// lines runs name with args in dir and hands each line of its output to each,
// up to finderCap lines, until each says to stop or the search is over; then
// the program is ended.
func lines(ctx context.Context, dir string, limit time.Duration, name string, args []string, each func(string) bool) {
	ctx, cancel := context.WithTimeout(ctx, limit)
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
