package ui

// selectModeName names the selection mode where it shows: at the top right of
// the viewport while selecting, and in its key reference's title (tdp K11).
const selectModeName = "Selection"

// modeKey is one key of the yank viewport's selection mode (tdp K11): what to
// press, what it does, and the keystrokes it is made of.
type modeKey struct {
	keys string   // as shown in the help: "j/↓"
	desc string   //
	run  []string // the keystrokes: {"g", "g"} for gg
}

// selectKeys is the one table of the selection mode: its help (?) is built from
// it, and the viewport's own key reference takes its movement rows, so the two
// cannot disagree (tdp K11, M3). The mode has no key list to run rows from:
// its keys are pressed directly.
var selectKeys = []modeKey{
	{"h/←", "move left", []string{"h"}},
	{"l/→", "move right", []string{"l"}},
	{"j/↓", "move down", []string{"j"}},
	{"k/↑", "move up", []string{"k"}},
	{"0", "start of the line", []string{"0"}},
	{"$", "end of the line", []string{"$"}},
	{"gg", "top", []string{"g", "g"}},
	{"G", "bottom", []string{"G"}},
	{"u", "half a page up", []string{"u"}},
	{"d", "half a page down", []string{"d"}},
	{"y", "copy the selection", []string{"y"}},
	{"v/Esc", "leave the selection", []string{"v"}},
}

// isMoveKey reports a row that only moves the cursor — the rows the viewport
// shares with the mode.
func (k modeKey) isMoveKey() bool { return k.run[0] != "y" && k.run[0] != "v" }

// trigger is the first keystroke of the key.
func (k modeKey) trigger() string { return k.run[0] }

// selectHelpRows is the selection mode's key reference (?): the table, then
// the keys that are not the mode's own.
func selectHelpRows() []helpRow {
	var rows []helpRow
	for _, k := range selectKeys {
		rows = append(rows, helpRow{key: k.keys, desc: k.desc})
	}
	return append(rows,
		helpRow{key: "?", desc: "these keys"},
		helpRow{key: "q", desc: "quit — pick a directory to cd to"})
}
