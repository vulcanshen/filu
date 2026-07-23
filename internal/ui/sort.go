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
	sortSize
	sortMtime
	sortExt
)

// sortColDef pairs a column with its display title and picker hotkey.
type sortColDef struct {
	col   sortCol
	title string
	key   string
}

var sortCols = []sortColDef{
	{sortName, "Name", "n"},
	{sortSize, "Size", "s"},
	{sortMtime, "Modified", "m"},
	{sortExt, "Extension", "e"},
}

// sortRule is one tier of the sort: a column and its direction.
type sortRule struct {
	col sortCol
	asc bool
}

// sortChain is the active multi-tier sort (a global preference, like kbu's
// per-kind sort). Empty means the default order: directories first, name
// ascending. Directories are always floated to the top on top of any chain.
var sortChain []sortRule

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

func sortChainIndex(c sortCol) int {
	for i, r := range sortChain {
		if r.col == c {
			return i
		}
	}
	return -1
}

// sortChainSet upserts a column into the chain (updates its direction if already
// present, else appends a new tier).
func sortChainSet(c sortCol, asc bool) {
	if i := sortChainIndex(c); i >= 0 {
		sortChain[i].asc = asc
		return
	}
	sortChain = append(sortChain, sortRule{c, asc})
}

// sortChainUnset drops a column from the chain if present.
func sortChainUnset(c sortCol) {
	if i := sortChainIndex(c); i >= 0 {
		sortChain = append(sortChain[:i], sortChain[i+1:]...)
	}
}

// sortItems orders entries in place: directories first, then each active tier in
// order, with a case-insensitive name-ascending final tiebreak.
func sortItems(items []fileItem) {
	sort.SliceStable(items, func(i, j int) bool {
		a, b := items[i], items[j]
		if a.isDir != b.isDir {
			return a.isDir
		}
		for _, r := range sortChain {
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
func sortBadgeText(c sortCol) string {
	i := sortChainIndex(c)
	if i < 0 {
		return ""
	}
	dir := "asc"
	if !sortChain[i].asc {
		dir = "desc"
	}
	if len(sortChain) > 1 {
		return "(" + strconv.Itoa(i+1) + ") " + dir
	}
	return dir
}

// sortHeaderSuffix is the compact indicator appended to the panel [2] "Files"
// header, e.g. "  size⏷ name⏶" (Nerd Font arrows; empty when unsorted).
func sortHeaderSuffix() string {
	if len(sortChain) == 0 {
		return ""
	}
	parts := make([]string, 0, len(sortChain))
	for _, r := range sortChain {
		arrow := sortAscGlyph
		if !r.asc {
			arrow = sortDescGlyph
		}
		parts = append(parts, strings.ToLower(sortColTitle(r.col))+arrow)
	}
	return "  " + strings.Join(parts, " ")
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
// Reset entry when a sort is active; the menu stays open across step swaps.
func (m *AppModel) setSortColumnItems() {
	items := make([]menuItem, 0, len(sortCols)+3)
	for _, d := range sortCols {
		items = append(items, menuItem{label: d.title, key: d.key, hint: sortBadgeText(d.col)})
	}
	if len(sortChain) > 0 {
		items = append(items, menuItem{separator: true})
		items = append(items, menuItem{label: "Reset", key: "r", hint: "default order"})
	}
	m.sortMenu.setItems(items, "Sort by…")
}

// setSortDirectionItems shows Ascending/Descending, plus Unset when the column
// is already part of the chain.
func (m *AppModel) setSortDirectionItems(col sortCol) {
	items := []menuItem{
		{label: "Ascending", key: "a"},
		{label: "Descending", key: "d"},
	}
	if sortChainIndex(col) >= 0 {
		items = append(items, menuItem{separator: true})
		items = append(items, menuItem{label: "Unset", key: "u", hint: "remove from sort"})
	}
	m.sortMenu.setItems(items, "Sort "+sortColTitle(col)+"…")
}

// advanceSortFlow handles a committed picker key: on the column step it either
// resets or steps to direction; on the direction step it applies asc/desc/unset,
// re-sorts, persists, then loops back to the column picker (kbu chain building).
func (m *AppModel) advanceSortFlow(key string) tea.Cmd {
	switch m.sortStep {
	case sortStepColumn:
		if key == "r" {
			sortChain = nil
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
			sortChainSet(m.sortFlowCol, true)
		case "d":
			sortChainSet(m.sortFlowCol, false)
		case "u":
			sortChainUnset(m.sortFlowCol)
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
