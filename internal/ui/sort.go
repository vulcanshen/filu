package ui

import (
	"path/filepath"
	"sort"
	"strconv"
	"strings"
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

// --- sort picker flow (kbu: column → direction → back to the columns, Esc to close) ---
//
// Each step is its own popup (tdp F1): the column picker stays open underneath
// while the direction picker sits over it (F4), so Esc on the direction goes
// back to the columns.
