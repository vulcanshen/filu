package ui

import "strconv"

// The ? key reference lists the keys of the frontmost surface (tdp K6, M4): the
// popup on top, or the focused panel when none is open. Each list is built from
// what that surface actually does — a panel's from its Space menu rows — so the
// reference and the menu cannot drift apart.

// keyRef returns the title and rows of the key reference for the frontmost
// surface, walking the stack top-first the way Update routes keys.
func (m AppModel) keyRef() (string, []helpRow) {
	switch {
	case m.search.owns():
		return "Finder keys", finderKeyRef()
	case m.meta.owns():
		return "File information keys", metaKeyRef()
	case m.detailYank.owns():
		return "Preview viewport keys", yankKeyRef()
	case m.breadcrumb.owns():
		return "Breadcrumb keys", breadcrumbKeyRef()
	case m.confirm.owns():
		return "Confirm keys", confirmKeyRef(m.confirm.verb)
	case m.openWithMenu.owns():
		return "Open with keys", menuKeyRef(m.openWithMenu, nil)
	case m.searchMenu.owns():
		return "Search keys", menuKeyRef(m.searchMenu, nil)
	case m.openInMenu.owns():
		return "Open in keys", menuKeyRef(m.openInMenu, nil)
	case m.gotoMenu.owns():
		var extra []helpRow
		if m.gotoStep == gotoStepPinned && len(m.places.pinned) > 0 {
			extra = []helpRow{{key: "f", desc: "unfavorite the highlighted directory"}}
		}
		return "Goto keys", menuKeyRef(m.gotoMenu, extra)
	case m.sortMenu.owns():
		return "Sort keys", menuKeyRef(m.sortMenu, nil)
	case m.globalMenu.owns():
		return "Global operation keys", menuKeyRef(m.globalMenu, nil)
	case m.spaceMenu.owns():
		return "Space menu keys", menuKeyRef(m.spaceMenu, nil)
	}
	return m.panelKeyRef()
}

// panelKeyRef lists a panel's keys: its Space menu's item and panel rows (only
// the keys that can be pressed — the Global operation row has none), then the
// core and navigation keys.
func (m AppModel) panelKeyRef() (string, []helpRow) {
	items, title := m.buildSpaceMenu()
	rows := menuRows(items)
	rows = append(rows, helpRow{header: true, desc: "keys"},
		helpRow{key: "Tab", desc: "focus the next panel"},
		helpRow{key: "1 2 3", desc: "focus a panel directly"},
		helpRow{key: "j k", desc: "move down / up"},
		helpRow{key: "g G", desc: "top / bottom"},
		helpRow{key: "u d", desc: "half a page up / down"})
	if m.focus == panelList {
		rows = append(rows, helpRow{key: "Enter", desc: "go into the directory"})
	}
	rows = append(rows,
		helpRow{key: "Esc", desc: backDesc(m.focus)},
		helpRow{key: "Space", desc: "the menu of what you can do here"},
		helpRow{key: "?", desc: "these keys"},
		helpRow{key: "q", desc: "quit — pick a directory to cd to"},
		helpRow{key: "Ctrl+C", desc: "same as q, even while typing"})
	return title + " keys", rows
}

// backDesc is what Esc does on a panel.
func backDesc(p panelID) string {
	if p == panelList {
		return "up to the parent directory"
	}
	return "nothing here (it closes popups)"
}

// menuRows turns menu items into key-reference rows: each row that has a key
// the user can press, under the region headers it sits in. Rows with no
// pressable key (the Global operation row) and regions left empty are dropped.
func menuRows(items []menuItem) []helpRow {
	var rows []helpRow
	var pending *helpRow // a header, written only once a row under it is
	for _, it := range items {
		switch {
		case it.separator:
			continue
		case it.header:
			if it.warn {
				continue
			}
			h := helpRow{header: true, desc: it.label}
			pending = &h
			continue
		case it.key == "" || it.key == globalOpKey:
			continue
		}
		if pending != nil {
			rows = append(rows, *pending)
			pending = nil
		}
		desc := it.label
		if it.hint != "" && !isGlyphHint(it.hint) {
			desc += " — " + it.hint
		}
		rows = append(rows, helpRow{key: it.key, desc: desc})
	}
	return rows
}

// isGlyphHint reports a hint that is only a glyph marker (the quit picker's
// launch / tab marks), not words worth repeating in the reference.
func isGlyphHint(h string) bool { return len([]rune(h)) <= 3 }

// menuKeyRef lists a menu's own keys: its rows, any extra keys it answers to,
// then how to move and leave.
func menuKeyRef(menu spaceMenu, extra []helpRow) []helpRow {
	rows := append(menuRows(menu.items), extra...)
	rows = append(rows, helpRow{header: true, desc: "keys"},
		helpRow{key: "j k", desc: "move down / up"},
		helpRow{key: "g G", desc: "first / last row"},
		helpRow{key: "Enter", desc: "run the highlighted row"})
	if menu.spaceToggle {
		rows = append(rows, helpRow{key: "Space", desc: "close this menu"})
	}
	return append(rows, helpRow{key: "Esc", desc: "close, back to where you were"},
		helpRow{key: "?", desc: "these keys"})
}

// quitKeyRef is the quit picker's key reference.
func quitKeyRef(targets int) []helpRow {
	keys := "1"
	if targets > 1 {
		keys = "1–" + strconv.Itoa(targets)
	}
	return []helpRow{
		{key: keys, desc: "leave, with the shell in that directory"},
		{key: "j k", desc: "move down / up"},
		{key: "Enter", desc: "leave, with the shell in the highlighted directory"},
		{key: "Ctrl+C", desc: "leave now"},
		{key: "Esc", desc: "stay — back to where you were"},
		{key: "?", desc: "these keys"},
	}
}

func confirmKeyRef(verb string) []helpRow {
	return []helpRow{
		{key: "Enter y", desc: verb},
		{key: "Esc n", desc: "cancel"},
		{key: "?", desc: "these keys"},
	}
}

func breadcrumbKeyRef() []helpRow {
	return []helpRow{
		{key: "j k", desc: "move down / up"},
		{key: "g G", desc: "first / last level"},
		{key: "Enter", desc: "take this tab to the highlighted directory"},
		{key: "Esc", desc: "close"},
		{key: "?", desc: "these keys"},
	}
}

func finderKeyRef() []helpRow {
	return []helpRow{
		{key: "j k", desc: "move down / up"},
		{key: "u d", desc: "half a page up / down"},
		{key: "g G", desc: "first / last result"},
		{key: "Enter", desc: "go to the highlighted result"},
		{key: "Tab", desc: "back to the query"},
		{key: "Esc", desc: "close"},
		{key: "?", desc: "these keys"},
	}
}

func yankKeyRef() []helpRow {
	return []helpRow{
		{key: "h j k l", desc: "move"},
		{key: "0 $", desc: "start / end of the line"},
		{key: "gg G", desc: "top / bottom"},
		{key: "u d", desc: "half a page up / down"},
		{key: "v", desc: "start selecting"},
		{key: "y", desc: "copy everything"},
		{key: "Esc", desc: "close"},
		{key: "?", desc: "these keys"},
	}
}

func metaKeyRef() []helpRow {
	return []helpRow{
		{key: "j k", desc: "scroll down / up"},
		{key: "g G", desc: "top / bottom"},
		{key: "Esc", desc: "close"},
		{key: "?", desc: "these keys"},
	}
}
