package ui

import (
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// sortCol is a file attribute the list can be ordered by.
type sortCol int

const (
	sortName sortCol = iota
	sortSize         // kept for value stability (persisted col ints) — no longer offered
	sortMtime
	sortExt // kept for value stability — no longer offered
	sortPerm
	sortOwner
)

// sortColDef pairs a column with its display title and picker hotkey.
type sortColDef struct {
	col   sortCol
	title string
	key   string
}

// sortCols is the offered picker list — only the columns the list actually shows
// (Name / Modified / Permissions / Owner / Size). Extension remains in the enum
// but is not sortable now that it is not displayed.
var sortCols = []sortColDef{
	{sortName, "Name", "n"},
	{sortMtime, "Modified", "m"},
	{sortPerm, "Permissions", "p"},
	{sortOwner, "Owner", "o"},
	{sortSize, "Size", "s"},
}

// sortRule is one tier of the sort: a column and its direction.
type sortRule struct {
	col sortCol
	asc bool
}

// sortByDir maps an exact directory path to its multi-tier sort chain. A dir with
// no entry uses the default order (directories first, name ascending); directories
// are always floated to the top on top of any chain. Persisted in state.yaml, so a
// dir's sort survives whether or not a tab is currently on it.
var sortByDir = map[string][]sortRule{}

// cleanDir normalises a path to the map key (so /a/b and /a/b/ resolve the same).
func cleanDir(dir string) string { return filepath.Clean(dir) }

// sortRulesFor is the sort chain stored for dir (nil = the default order).
func sortRulesFor(dir string) []sortRule { return sortByDir[cleanDir(dir)] }

// sort arrows (Nerd Font, so the display-width layer counts them correctly).
var (
	sortAscGlyph  = string(rune(0xf161)) // nf-fa-sort_amount_asc
	sortDescGlyph = string(rune(0xf160)) // nf-fa-sort_amount_desc
)

func sortColByKey(key string) (sortCol, bool) {
	for _, d := range sortCols {
		if d.key == key {
			return d.col, true
		}
	}
	return 0, false
}

func sortColTitle(c sortCol) string {
	for _, d := range sortCols {
		if d.col == c {
			return d.title
		}
	}
	return ""
}

// sortRuleIndex finds col within a chain (-1 when absent).
func sortRuleIndex(rules []sortRule, c sortCol) int {
	for i, r := range rules {
		if r.col == c {
			return i
		}
	}
	return -1
}

// setSortFor upserts a column into dir's chain (updates its direction if already
// present, else appends a new tier).
func setSortFor(dir string, c sortCol, asc bool) {
	key := cleanDir(dir)
	rules := sortByDir[key]
	if i := sortRuleIndex(rules, c); i >= 0 {
		rules[i].asc = asc
	} else {
		rules = append(rules, sortRule{c, asc})
	}
	sortByDir[key] = rules
}

// unsetSortFor drops a column from dir's chain, removing the entry when it empties.
func unsetSortFor(dir string, c sortCol) {
	key := cleanDir(dir)
	rules := sortByDir[key]
	if i := sortRuleIndex(rules, c); i >= 0 {
		rules = append(rules[:i], rules[i+1:]...)
	}
	if len(rules) == 0 {
		delete(sortByDir, key)
	} else {
		sortByDir[key] = rules
	}
}

// resetSortFor clears dir's chain entirely (back to the default order).
func resetSortFor(dir string) { delete(sortByDir, cleanDir(dir)) }

// sortItems orders entries in place by the given chain: directories first, then
// each active tier in order, with a case-insensitive name-ascending final
// tiebreak. A nil/empty chain is just the default (dirs first, name asc).
func sortItems(items []fileItem, rules []sortRule) {
	sort.SliceStable(items, func(i, j int) bool {
		a, b := items[i], items[j]
		if a.isDir != b.isDir {
			return a.isDir
		}
		for _, r := range rules {
			c := compareCol(a, b, r.col)
			if !r.asc {
				c = -c
			}
			if c != 0 {
				return c < 0
			}
		}
		return strings.ToLower(a.name) < strings.ToLower(b.name)
	})
}

func compareCol(a, b fileItem, c sortCol) int {
	switch c {
	case sortSize:
		return cmpInt64(a.size, b.size)
	case sortMtime:
		switch {
		case a.mtime.Before(b.mtime):
			return -1
		case a.mtime.After(b.mtime):
			return 1
		default:
			return 0
		}
	case sortExt:
		return strings.Compare(fileExt(a.name), fileExt(b.name))
	case sortPerm:
		return strings.Compare(a.perm, b.perm)
	case sortOwner:
		return strings.Compare(a.owner, b.owner)
	default: // sortName
		return strings.Compare(strings.ToLower(a.name), strings.ToLower(b.name))
	}
}

func cmpInt64(a, b int64) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	default:
		return 0
	}
}

func fileExt(name string) string {
	return strings.ToLower(strings.TrimPrefix(filepath.Ext(name), "."))
}

// sortBadgeText is the ASCII tier badge shown in the picker popup (kept ASCII so
// it can't break the popup's lipgloss-width layout): "asc" / "desc", prefixed
// "(n) " when more than one tier is active.
func sortBadgeText(rules []sortRule, c sortCol) string {
	i := sortRuleIndex(rules, c)
	if i < 0 {
		return ""
	}
	dir := "asc"
	if !rules[i].asc {
		dir = "desc"
	}
	if len(rules) > 1 {
		return "(" + strconv.Itoa(i+1) + ") " + dir
	}
	return dir
}

// --- sort picker flow (kbu: column → direction → loop, Esc to close) ---

// openSortColumnPicker opens the sort picker on the column step (animated).
func (m *AppModel) openSortColumnPicker() tea.Cmd {
	m.sortStep = sortStepColumn
	m.setSortColumnItems()
	m.sortMenu.setSize(m.width)
	return m.sortMenu.open()
}

// setSortColumnItems (re)populates the picker with the sortable columns plus a
// Reset entry when a sort is active; the menu stays open across step swaps. The
// badges reflect the sort of the active tab's directory (the one being edited).
func (m *AppModel) setSortColumnItems() {
	rules := sortRulesFor(m.cur().dir)
	items := make([]menuItem, 0, len(sortCols)+3)
	for _, d := range sortCols {
		items = append(items, menuItem{label: d.title, key: d.key, hint: sortBadgeText(rules, d.col)})
	}
	if len(rules) > 0 {
		items = append(items, menuItem{separator: true})
		items = append(items, menuItem{label: "Reset", key: "r", hint: "default order"})
	}
	m.sortMenu.setItems(items, "Sort by…")
}

// setSortDirectionItems shows Ascending/Descending, plus Unset when the column
// is already part of the active dir's chain.
func (m *AppModel) setSortDirectionItems(col sortCol) {
	items := []menuItem{
		{label: "Ascending", key: "a"},
		{label: "Descending", key: "d"},
	}
	if sortRuleIndex(sortRulesFor(m.cur().dir), col) >= 0 {
		items = append(items, menuItem{separator: true})
		items = append(items, menuItem{label: "Unset", key: "u", hint: "remove from sort"})
	}
	m.sortMenu.setItems(items, "Sort "+sortColTitle(col)+"…")
}

// advanceSortFlow handles a committed picker key: on the column step it either
// resets or steps to direction; on the direction step it applies asc/desc/unset,
// re-sorts, persists, then loops back to the column picker (kbu chain building).
func (m *AppModel) advanceSortFlow(key string) tea.Cmd {
	dir := m.cur().dir // the picker edits the active tab's directory
	switch m.sortStep {
	case sortStepColumn:
		if key == "r" {
			resetSortFor(dir)
			m.reloadAllTabs()
			saveState(m.snapshotState())
			m.setSortColumnItems()
			return nil
		}
		if col, ok := sortColByKey(key); ok {
			m.sortFlowCol = col
			m.sortStep = sortStepDirection
			m.setSortDirectionItems(col)
		}
		return nil
	case sortStepDirection:
		switch key {
		case "a":
			setSortFor(dir, m.sortFlowCol, true)
		case "d":
			setSortFor(dir, m.sortFlowCol, false)
		case "u":
			unsetSortFor(dir, m.sortFlowCol)
		}
		m.reloadAllTabs()
		saveState(m.snapshotState())
		m.sortStep = sortStepColumn
		m.setSortColumnItems()
		return nil
	}
	return nil
}

// reloadAllTabs re-sorts every tab (cursor preserved) after a sort change.
func (m *AppModel) reloadAllTabs() {
	for i := range m.tabs {
		m.tabs[i].reloadPreserving()
	}
	m.cur().ensureVisible(m.listRows())
	m.refreshPreview()
}
