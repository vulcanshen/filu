package ui

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// searchModel is filu's native file finder (snacks/Telescope form, not the fzf
// binary). It has two modes over the same picker UI, both rooted at the active
// tab's subtree:
//   - Search (`/`, byContent=false): fuzzy-filter the file list BY NAME,
//     in-memory, ranked best-first — a single list box, no preview.
//   - Find (`f`, byContent=true): filter BY CONTENT (`rg --files-with-matches`,
//     so a file that matches many times still appears once) — a split popup with
//     the list on the left and a preview on the right (stacked when narrow), the
//     preview scrolled to the matched line.
//
// The unit you pick is always a file/dir — Enter drops focus into the list, a
// second Enter reveals the pick in the active tab (descending into its subdir).
// Drawn entirely in filu's own render loop — no PTY, so it can never break out
// of its popup; the subtree scope keeps it fast and bounded.
type searchMode int

const (
	searchInput searchMode = iota // typing edits the query (re-filters)
	searchNav                     // j/k/u/d move the result cursor
)

// grepDebounce delays the re-filter after a keystroke, so a burst of typing runs
// rg once (for the final query) instead of once per character.
const grepDebounce = 120 * time.Millisecond

// fileMatch is one list entry: a path (relative to root) and, for a content hit,
// the first matching line (1-based; 0 for a dir or an all-files entry) so the
// preview can scroll to it.
type fileMatch struct {
	path string
	line int
}

type searchModel struct {
	root      string      // absolute search root (the active tab's dir, or $HOME for goto)
	byContent bool        // true = Find (rg content + preview); false = Search / Goto (name)
	dirsOnly  bool        // true = Goto: list only directories (fd --type d), no hidden
	newTab    bool        // true = Goto opened by T: confirm opens a new tab, not a reveal
	query     string      // filter text
	allFiles  []fileMatch // every file + dir under root (the empty-query list)
	files     []fileMatch // current list: allFiles, or the name/content matches
	cursor    int         // into files
	scroll    int
	mode      searchMode
	gen       int  // bumped per content-query change; stale rg results are dropped
	openGen   int  // bumped per open; guards the fd all-files load (query-gen independent)
	loading   bool // allFiles still loading
	searching bool // an rg filter is in flight

	preview       previewModel // preview of the selected file
	previewFor    string       // abs path the preview currently holds
	previewScroll int          // preview offset (scrolls to the match line)

	blink    bool // input cursor blink phase (on while true)
	blinkGen int  // bumped per open so exactly one blink loop runs

	width, height int
	anim          popupAnimator
}

// searchBlinkMsg toggles the input cursor; gen keeps one blink loop per open.
type searchBlinkMsg struct{ gen int }

// fileBatchMsg carries one streamed chunk of the fd listing back to the model,
// tagged with the open generation + root so a reopen's stale batches are dropped.
// done marks the final (possibly empty) batch.
type fileBatchMsg struct {
	gen   int
	root  string
	batch []string
	done  bool
}

// grepFireMsg fires grepDebounce after a keystroke; if its gen still matches the
// app kicks off the actual rg filter.
type grepFireMsg struct {
	gen         int
	root, query string
}

// grepFilesMsg carries an rg run's matched files (with first-match lines).
type grepFilesMsg struct {
	gen     int
	root    string
	matches []fileMatch
}

// searchConfirmMsg asks the app to act on an absolute path chosen in a finder:
// reveal it in the active tab, or (newTab) open it as a new panel [2] tab.
type searchConfirmMsg struct {
	path   string
	newTab bool
}

func newSearch() searchModel {
	return searchModel{anim: newPopupAnimator("search", popupLayerColor(1))}
}

// openSearchMenu opens the Search chooser (`/`): a flat {filename, content} pick
// that then opens the finder in that mode over the active tab's subtree. Folding
// the old top-level Find (`f`) in here frees `f` for Favorite.
func (m *AppModel) openSearchMenu() tea.Cmd {
	m.searchMenu.setItems([]menuItem{
		{label: "filename", key: "f", hint: "fuzzy match on file names (fd)"},
		{label: "content", key: "c", hint: "grep inside files (rg), with preview"},
	}, "Search…")
	m.searchMenu.setSize(m.width)
	return m.searchMenu.open()
}

// openSearch opens the by-name finder over the active tab's directory (the
// Search chooser's `filename` pick).
func (m *AppModel) openSearch() tea.Cmd {
	return m.search.open(m.cur().dir, m.width, m.height, false, false, m.searchCh)
}

// openFind opens the by-content finder over the active tab's directory (the
// Search chooser's `content` pick).
func (m *AppModel) openFind() tea.Cmd {
	return m.search.open(m.cur().dir, m.width, m.height, true, false, m.searchCh)
}

// openGoto opens the finder over $HOME listing only directories (fuzzy on the
// path), so Enter teleports the active tab to any directory under home. The
// chord `go` and the panel [2] Space menu both route here.
func (m *AppModel) openGoto() tea.Cmd {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		home = m.cur().dir // no home known → fall back to the current dir
	}
	return m.search.open(home, m.width, m.height, false, true, m.searchCh)
}

// openGotoNewTab opens the same Goto finder as openGoto (`T`), but its confirm
// opens the chosen directory as a NEW panel [2] tab instead of teleporting the
// active one. The caller has already checked the tab count is under maxTabs.
func (m *AppModel) openGotoNewTab() tea.Cmd {
	cmd := m.openGoto()
	m.search.newTab = true
	return cmd
}

// open resets the finder over root and starts streaming the file list into ch.
// byContent selects Find (rg + preview) vs Search/Goto (name-only); dirsOnly
// restricts the listing to directories (Goto). Results appear as fd emits them —
// no sort, no wait for the full walk.
func (m *searchModel) open(root string, w, h int, byContent, dirsOnly bool, ch chan<- fileBatchMsg) tea.Cmd {
	m.root = root
	m.byContent = byContent
	m.dirsOnly = dirsOnly
	m.newTab = false // normal opens reveal in place; only openGotoNewTab flips this
	m.query = ""
	m.allFiles, m.files = nil, nil
	m.cursor, m.scroll = 0, 0
	m.mode = searchInput
	m.gen++     // invalidate any in-flight rg from a previous open
	m.openGen++ // stale stream batches (wrong gen) are dropped
	m.loading, m.searching = true, false
	m.preview, m.previewFor = previewModel{}, ""
	m.blink, m.blinkGen = true, m.blinkGen+1
	m.width, m.height = w, h
	return tea.Batch(m.anim.open(), streamFilesCmd(m.openGen, root, dirsOnly, ch), blinkTickCmd(m.blinkGen))
}

// onBlink toggles the input cursor and reschedules, as long as this is still the
// current open's blink loop.
func (m *searchModel) onBlink(msg searchBlinkMsg) tea.Cmd {
	if !m.anim.isActive() || msg.gen != m.blinkGen {
		return nil
	}
	m.blink = !m.blink
	return blinkTickCmd(msg.gen)
}

func blinkTickCmd(gen int) tea.Cmd {
	return tea.Tick(530*time.Millisecond, func(time.Time) tea.Msg { return searchBlinkMsg{gen} })
}

func (m *searchModel) setSize(w, h int)   { m.width, m.height = w, h }
func (m searchModel) isActive() bool      { return m.anim.isActive() }
func (m searchModel) isInteractive() bool { return m.anim.isInteractive() }

func (m *searchModel) handleTick(msg AnimTickMsg) tea.Cmd {
	if msg.Target != m.anim.target {
		return nil
	}
	return m.anim.tick()
}

// onStreamBatch appends one streamed chunk (dropping a stale one from a previous
// open) and re-applies the current view: an empty query shows everything so far,
// a by-name query re-filters over the grown list, and a by-content query is left
// to rg. done flips off the loading state.
func (m *searchModel) onStreamBatch(msg fileBatchMsg) {
	if !m.anim.isActive() || msg.gen != m.openGen || msg.root != m.root {
		return // a stale batch (reopened), or the finder is closed
	}
	for _, p := range msg.batch {
		m.allFiles = append(m.allFiles, fileMatch{path: p})
	}
	if msg.done {
		m.loading = false
	}
	switch {
	case m.query == "":
		m.files = m.allFiles
	case !m.byContent:
		m.filterByName()
	default:
		return // by-content: rg owns the non-empty query
	}
	m.clampCursor()
	m.refreshPreview()
}

// onGrepFire runs rg once the debounce elapses, unless a newer keystroke has
// superseded this generation.
func (m *searchModel) onGrepFire(msg grepFireMsg) tea.Cmd {
	if !m.anim.isActive() || msg.gen != m.gen || msg.root != m.root {
		return nil
	}
	return grepFilesCmd(msg.gen, msg.root, msg.query)
}

// onGrepResult installs an rg run's files, dropping a stale one.
func (m *searchModel) onGrepResult(msg grepFilesMsg) {
	if !m.anim.isActive() || msg.gen != m.gen || msg.root != m.root {
		return
	}
	m.files = msg.matches
	m.searching = false
	m.cursor, m.scroll = 0, 0
	m.refreshPreview()
}

// update runs the modal keymap: input mode edits the query (and re-filters); nav
// mode moves the cursor. Enter/Esc switch modes (or confirm/close).
func (m searchModel) update(msg tea.KeyMsg) (searchModel, tea.Cmd) {
	if !m.anim.isInteractive() {
		return m, nil
	}
	if m.mode == searchInput {
		switch msg.Type {
		case tea.KeyEsc:
			return m, m.anim.close()
		case tea.KeyEnter:
			if len(m.files) > 0 { // hand focus to the list
				m.mode = searchNav
			}
			return m, nil
		case tea.KeyBackspace:
			if r := []rune(m.query); len(r) > 0 {
				m.query = string(r[:len(r)-1])
				return m, m.queryChanged()
			}
			return m, nil
		case tea.KeySpace:
			m.query += " "
			return m, m.queryChanged()
		case tea.KeyRunes:
			m.query += string(msg.Runes)
			return m, m.queryChanged()
		}
		return m, nil
	}
	// nav mode
	switch msg.String() {
	case "esc": // leave the finder, like every other popup in the app
		return m, m.anim.close()
	case "q": // back to the input to refine the query
		m.mode = searchInput
	case "enter": // confirm → reveal in the active tab, then close
		if p := m.selectedAbs(); p != "" {
			return m, tea.Batch(m.anim.close(), func() tea.Msg { return searchConfirmMsg{path: p, newTab: m.newTab} })
		}
		return m, m.anim.close()
	case "j", "down":
		m.moveCursor(1)
	case "k", "up":
		m.moveCursor(-1)
	case "d", "ctrl+d":
		m.moveCursor(max(m.listRows()/2, 1))
	case "u", "ctrl+u":
		m.moveCursor(-max(m.listRows()/2, 1))
	case "g":
		m.cursor, m.scroll = 0, 0
		m.refreshPreview()
	case "G":
		m.cursor = max(len(m.files)-1, 0)
		m.ensureVisible()
		m.refreshPreview()
	}
	return m, nil
}

// queryChanged re-filters after a keystroke. An empty query restores the full
// list; by-name (Search) filters allFiles in-memory instantly; by-content (Find)
// bumps the generation and schedules a debounced rg run.
func (m *searchModel) queryChanged() tea.Cmd {
	m.cursor, m.scroll = 0, 0
	if m.query == "" {
		m.files = m.allFiles
		m.searching = false
		m.refreshPreview()
		return nil
	}
	if !m.byContent { // by-name: synchronous, in-memory, no rg
		m.filterByName()
		m.searching = false
		m.refreshPreview()
		return nil
	}
	m.gen++ // by-content: invalidate older rg runs
	m.searching = true
	return grepDebounceCmd(m.gen, m.root, m.query)
}

// filterByName narrows allFiles to entries whose relative path fuzzy-matches the
// query, ranked best-first. Builds a fresh slice so it never mutates allFiles
// (which m.files may alias when the query is empty).
func (m *searchModel) filterByName() {
	type scored struct {
		f     fileMatch
		score int
	}
	var hits []scored
	for _, f := range m.allFiles {
		if s, ok := fuzzyMatch(f.path, m.query); ok {
			hits = append(hits, scored{f, s})
		}
	}
	sort.SliceStable(hits, func(i, j int) bool { return hits[i].score > hits[j].score })
	out := make([]fileMatch, len(hits))
	for i, h := range hits {
		out[i] = h.f
	}
	m.files = out
}

// fuzzyMatch reports whether query is a case-insensitive subsequence of target,
// with a score that favours matches at word boundaries (start / after a
// /_-. separator / camelCase), contiguous runs, and the basename over deep path
// segments, while penalising gaps. Higher is better; ok=false when query is not
// a subsequence. Greedy left-to-right — correct for detection, good enough for
// ranking a file list.
func fuzzyMatch(target, query string) (int, bool) {
	q := []rune(strings.ToLower(query))
	if len(q) == 0 {
		return 0, true
	}
	t := []rune(target)
	baseStart := 0
	for i, r := range t {
		if r == '/' {
			baseStart = i + 1
		}
	}
	score, qi, prev := 0, 0, -2
	for ti := 0; ti < len(t) && qi < len(q); ti++ {
		if unicode.ToLower(t[ti]) != q[qi] {
			continue
		}
		s := 1
		switch {
		case prev >= 0 && ti == prev+1:
			s += 6 // contiguous with the previous match
		case prev >= 0:
			s -= min(ti-prev-1, 4) // gap penalty (capped)
		}
		if ti == 0 || isSep(t[ti-1]) || (unicode.IsUpper(t[ti]) && unicode.IsLower(t[ti-1])) {
			s += 8 // word boundary
		}
		if ti >= baseStart {
			s += 3 // in the basename
		}
		score += s
		prev = ti
		qi++
	}
	if qi < len(q) {
		return 0, false
	}
	return score, true
}

func isSep(r rune) bool {
	return r == '/' || r == '_' || r == '-' || r == '.' || r == ' '
}

func (m *searchModel) moveCursor(delta int) {
	if len(m.files) == 0 {
		return
	}
	m.cursor += delta
	m.clampCursor()
	m.ensureVisible()
	m.refreshPreview()
}

func (m *searchModel) clampCursor() {
	if m.cursor > len(m.files)-1 {
		m.cursor = len(m.files) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
}

func (m *searchModel) ensureVisible() {
	rows := m.listRows()
	if m.cursor < m.scroll {
		m.scroll = m.cursor
	}
	if m.cursor >= m.scroll+rows {
		m.scroll = m.cursor - rows + 1
	}
	if m.scroll < 0 {
		m.scroll = 0
	}
}

// selectedAbs is the absolute path under the cursor, "" when none.
func (m searchModel) selectedAbs() string {
	if m.cursor < 0 || m.cursor >= len(m.files) {
		return ""
	}
	return filepath.Join(m.root, m.files[m.cursor].path)
}

// selectedLine is the first-match line of the cursor entry (0 = none).
func (m searchModel) selectedLine() int {
	if m.cursor < 0 || m.cursor >= len(m.files) {
		return 0
	}
	return m.files[m.cursor].line
}

// refreshPreview loads the selected entry's preview when the selection changes.
// All three finders preview: Search / Find show the selected file (Find scrolls
// to the match line; Search has none, so it shows the top), Goto shows the
// selected directory's tree.
func (m *searchModel) refreshPreview() {
	abs := m.selectedAbs()
	if abs != m.previewFor {
		m.previewFor = abs
		if abs == "" {
			m.preview = previewModel{}
		} else if info, err := os.Stat(abs); err != nil {
			m.preview = previewModel{note: "(unreadable)"}
		} else {
			it := fileItem{name: filepath.Base(abs), isDir: info.IsDir()}
			m.preview = loadPreview(it, filepath.Dir(abs), m.previewW())
		}
	}
	m.previewScroll = max(m.selectedLine()-1-3, 0) // match line, minus a little context
}

// --- geometry ---

// geometry lays out the two boxes. A wide screen puts the file list and the
// preview side by side; a narrow one stacks them. Returns each box's inner width
// and content-row count (the content hugs the borders — no pad rows).
func (m searchModel) geometry() (side bool, sW, sRows, pW, pRows int) {
	if m.width >= 96 { // room for a list box + a useful preview box
		side = true
		totalW := min(m.width-2, m.width*19/20)
		H := min(m.height-2, m.height*9/10)
		sOuter := max(totalW*2/5, 32)
		sW = max(sOuter-2, 20)
		pW = max(totalW-sOuter-2, 20)
		sRows = max(H-2, 4)
		pRows = sRows
		return
	}
	W := min(m.width-2, m.width*9/10)
	H := min(m.height-2, m.height*9/10)
	sH := max(H*11/20, 8)
	sW, pW = max(W-2, 20), max(W-2, 20)
	sRows = max(sH-2, 4)
	pRows = max(H-sH-2, 3)
	return
}

func (m searchModel) previewW() int { _, _, _, pW, _ := m.geometry(); return pW }

// listRows is how many file rows are visible (the input bar + divider take 2).
func (m searchModel) listRows() int { _, _, sRows, _, _ := m.geometry(); return max(sRows-2, 1) }

func (m searchModel) previewTitle() string {
	if abs := m.selectedAbs(); abs != "" {
		return " " + safeName(filepath.Base(abs))
	}
	return " preview"
}

func (m searchModel) renderPopup() string { return m.anim.renderFrame(m.renderFull()) }

// renderFull draws the finder: a list box and a preview box joined side by side
// (wide) or stacked (narrow). Find previews the file's content, Search the file
// from the top, Goto the selected directory's tree.
func (m searchModel) renderFull() string {
	bc := popupLayerColor(1)
	side, sW, sRows, pW, pRows := m.geometry()
	title := " Search"
	switch {
	case m.dirsOnly:
		title = " Goto"
	case m.byContent:
		title = " Find"
	}
	sb := drawPopupBoxPad(bc, title, m.hint(), m.listColumn(sW, sRows), sW, false)
	pb := drawPopupBoxPad(bc, m.previewTitle(), "", m.previewColumn(pW, pRows), pW, false)
	if side {
		h := strings.Count(sb, "\n") + 1 // a 1-col gap so the two boxes read as separate panels
		gap := strings.TrimSuffix(strings.Repeat(" \n", h), "\n")
		return joinH(sb, gap, pb)
	}
	return joinV(sb, pb)
}

// listColumn is the input bar + a divider + the file list, exactly rows tall.
func (m searchModel) listColumn(w, rows int) []string {
	out := make([]string, 0, rows)
	out = append(out, m.inputBar(w))
	out = append(out, lipgloss.NewStyle().Foreground(dimColor).Render(strings.Repeat("─", w)))

	listRows := rows - len(out)
	cursorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(baseHex)).Background(handColor)
	dim := lipgloss.NewStyle().Foreground(dimColor)
	switch {
	case m.loading:
		out = append(out, dim.Render(" (indexing…)"))
	case m.searching && len(m.files) == 0:
		out = append(out, dim.Render(" (searching…)"))
	case len(m.files) == 0:
		note := " (no matches)"
		if m.query == "" {
			note = " (empty)"
		}
		out = append(out, dim.Render(note))
	default:
		end := min(m.scroll+listRows, len(m.files))
		for i := m.scroll; i < end; i++ {
			out = append(out, fileRow(m.files[i].path, w, i == m.cursor, cursorStyle))
		}
	}
	for len(out) < rows {
		out = append(out, "")
	}
	return out[:rows]
}

// fileRow renders one search-list entry with its Nerd Font type icon + eza
// colour, matching panel [2]. fd marks directories with a trailing "/".
func fileRow(rel string, w int, cursor bool, cursorStyle lipgloss.Style) string {
	isDir := strings.HasSuffix(rel, "/")
	name := safeName(strings.TrimSuffix(rel, "/"))
	it := fileItem{name: filepath.Base(name), isDir: isDir}
	content := " " + fileIcon(it) + " " + name
	if cursor {
		return cursorStyle.Render(padDisp(content, w))
	}
	return truncate(lipgloss.NewStyle().Foreground(fileColor(it)).Render(content), w)
}

// previewColumn is the selected file's preview, exactly rows tall, scrolled to
// the matched line and marking it with a reverse-video bar.
func (m searchModel) previewColumn(w, rows int) []string {
	out := make([]string, 0, rows)
	lines := m.preview.contentLines()
	if m.selectedAbs() == "" {
		lines = []string{lipgloss.NewStyle().Foreground(dimColor).Render("(no selection)")}
	}
	start := min(max(m.previewScroll, 0), max(len(lines)-rows, 0))
	matchIdx := m.selectedLine() - 1 // 0-based line of the hit; -1 when none
	markStyle := lipgloss.NewStyle().Background(userColor).Foreground(lipgloss.Color(baseHex))
	for i := range rows {
		li := start + i
		switch {
		case li >= len(lines):
			out = append(out, "")
		case li == matchIdx:
			// lavender bar over the PLAIN line — a bg over the syntax colours
			// would be cleared by their inner resets.
			out = append(out, markStyle.Render(padDisp(ansi.Strip(lines[li]), w)))
		default:
			out = append(out, truncate(lines[li], w))
		}
	}
	return out
}

// inputBar is the query row (snacks form): a peach chevron prompt, the query
// with a blinking block cursor, and the count on the right — no background bar,
// since the blinking cursor already marks it as an input.
func (m searchModel) inputBar(w int) string {
	glyph := lipgloss.NewStyle().Foreground(lipgloss.Color("#fab387")).Bold(true).Render(inputGlyph)
	gW := dispWidth(inputGlyph)

	cur := " "
	if m.mode == searchInput && m.blink {
		cur = "█"
	}
	left := " " + m.query + cur
	count := fmt.Sprintf("%d", len(m.files))
	if m.mode == searchNav && len(m.files) > 0 {
		count = fmt.Sprintf("%d/%d", m.cursor+1, len(m.files))
	}
	count += " "

	avail := w - gW
	leftW, countW := dispWidth(left), dispWidth(count)
	if leftW+countW > avail { // query too long: keep the tail (cursor) visible
		left = ansi.TruncateLeft(left, leftW-(avail-countW-1), "…")
		leftW = dispWidth(left)
	}
	gap := max(avail-leftW-countW, 0)
	return glyph + left + strings.Repeat(" ", gap) + lipgloss.NewStyle().Foreground(dimColor).Render(count)
}

func (m searchModel) hint() string {
	if m.mode == searchNav {
		return " j/k/u/d · Enter=go · q=input · Esc=close "
	}
	return " Enter=list · Esc=close "
}

// --- fd / ripgrep ---

// streamBatch is how many fd lines are gathered before a chunk is sent to the UI
// — small enough that the first results show almost immediately.
const streamBatch = 256

// streamFilesCmd launches the fd stream in the background; batches arrive on ch
// and are read by the app's waitSearch loop.
func streamFilesCmd(gen int, root string, dirsOnly bool, ch chan<- fileBatchMsg) tea.Cmd {
	return func() tea.Msg {
		go streamDirFiles(gen, root, dirsOnly, ch)
		return nil
	}
}

// streamDirFiles runs fd under root and streams its output to ch in batches,
// tagged with gen+root so a reopen's stale batches are dropped. Search / Find
// list files + dirs (incl. hidden); Goto (dirsOnly) lists directories only, no
// hidden, with the ignore list applied. It stops at finderCap and always ends
// with a done batch. No fd → a one-shot Go walk.
func streamDirFiles(gen int, root string, dirsOnly bool, ch chan<- fileBatchMsg) {
	send := func(batch []string, done bool) {
		ch <- fileBatchMsg{gen: gen, root: root, batch: batch, done: done}
	}
	if _, err := exec.LookPath("fd"); err != nil {
		send(walkDirFiles(root, dirsOnly), true)
		return
	}
	args := []string{"--type", "f", "--type", "d", "--hidden"}
	if dirsOnly {
		args = []string{"--type", "d"} // dirs only, no --hidden
	}
	args = append(args, "--strip-cwd-prefix")
	for _, ig := range finderIgnoreDirs {
		args = append(args, "--exclude", ig)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "fd", args...)
	cmd.Dir = root
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		send(walkDirFiles(root, dirsOnly), true)
		return
	}
	if err := cmd.Start(); err != nil {
		send(walkDirFiles(root, dirsOnly), true)
		return
	}
	sc := bufio.NewScanner(stdout)
	sc.Buffer(make([]byte, 64*1024), 1<<20)
	batch := make([]string, 0, streamBatch)
	total := 0
	for sc.Scan() {
		line := sc.Text()
		if line == "" {
			continue
		}
		batch = append(batch, line)
		total++
		if len(batch) >= streamBatch {
			send(batch, false)
			batch = make([]string, 0, streamBatch)
		}
		if total >= finderCap {
			break
		}
	}
	cancel()
	_ = cmd.Wait()
	send(batch, true) // final (possibly empty) chunk marks done
}

func grepDebounceCmd(gen int, root, query string) tea.Cmd {
	return tea.Tick(grepDebounce, func(time.Time) tea.Msg {
		return grepFireMsg{gen: gen, root: root, query: query}
	})
}

func grepFilesCmd(gen int, root, query string) tea.Cmd {
	return func() tea.Msg {
		return grepFilesMsg{gen: gen, root: root, matches: rgMatches(root, query)}
	}
}

// rgMatches returns the files under root whose CONTENT matches query, one entry
// per file (deduped) carrying that file's first-match line so the preview can
// scroll to it.
func rgMatches(root, query string) []fileMatch {
	if _, err := exec.LookPath("rg"); err != nil {
		return nil
	}
	lines, _ := streamLines(root, "rg", "--line-number", "--no-heading",
		"--color", "never", "--smart-case", "--", query)
	seen := map[string]bool{}
	var out []fileMatch
	for _, l := range lines {
		path, line, ok := parseGrepLine(l)
		if !ok || seen[path] {
			continue
		}
		seen[path] = true
		out = append(out, fileMatch{path: path, line: line})
	}
	return out
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

// streamLines runs name+args with cwd=root, reading up to finderCap lines and
// then killing the process — so we pay for the first N results, not a full scan.
func streamLines(root, name string, args ...string) ([]string, bool) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = root
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, false
	}
	if err := cmd.Start(); err != nil {
		return nil, false
	}
	var out []string
	sc := bufio.NewScanner(stdout)
	sc.Buffer(make([]byte, 64*1024), 1<<20)
	for sc.Scan() {
		if line := sc.Text(); line != "" {
			out = append(out, line)
			if len(out) >= finderCap {
				break
			}
		}
	}
	cancel()
	_ = cmd.Wait()
	return out, true
}

func walkDirFiles(root string, dirsOnly bool) []string {
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
		if dirsOnly {
			if !d.IsDir() {
				return nil // Goto lists directories only
			}
			if strings.HasPrefix(d.Name(), ".") {
				return filepath.SkipDir // skip hidden dirs (and their subtrees), matching fd
			}
		}
		if rel, e := filepath.Rel(root, p); e == nil {
			out = append(out, rel)
		}
		if len(out) >= finderCap {
			return filepath.SkipAll
		}
		return nil
	})
	return out
}

// revealPath points the active tab at path: a directory is entered, a file's
// parent is entered with the cursor landing on the file.
func (m *AppModel) revealPath(path string) {
	info, err := os.Stat(path)
	if err != nil {
		return
	}
	l := m.cur()
	if info.IsDir() {
		l.dir = path
	} else {
		l.dir = filepath.Dir(path)
	}
	l.cursor, l.offset = 0, 0
	l.reload()
	if !info.IsDir() {
		name := filepath.Base(path)
		for i, it := range l.items {
			if it.name == name {
				l.cursor = i
				break
			}
		}
	}
	l.ensureVisible(m.listRows())
}
