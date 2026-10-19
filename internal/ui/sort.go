package ui

import (
	"path/filepath"
	"sort"
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

// sortItems orders entries by name, A to Z whatever the case: a directory's
// own order, which the list keeps when it is not sorted. The list keeps the
// directories first and sorts by the directory's chain (fileRows, sortKeys).
func sortItems(items []fileItem) {
	sort.SliceStable(items, func(i, j int) bool { return strings.ToLower(items[i].name) < strings.ToLower(items[j].name) })
}

// --- sort picker flow (kbu: column → direction → back to the columns, Esc to close) ---
//
// Each step is its own popup (tdp F1): the column picker stays open underneath
// while the direction picker sits over it (tdp F4), so Esc on the direction goes
// back to the columns.
