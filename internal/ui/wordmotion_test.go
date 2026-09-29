package ui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// wordView is a viewport over lines, cont marking hard-wrap continuations.
func wordView(lines []string, cont []bool) detailYank {
	m := newDetailYank()
	m.setSize(100, 40)
	m.open("t", lines, false, cont)
	m.anim.state = popupOpen
	return m
}

// The word motions move as vim does — each landing spot written out: word
// characters and punctuation are separate words, blanks and line ends are
// skipped, an empty line is a word of its own, and a hard-wrapped word is one
// word across the wrap.
func TestWordMotions(t *testing.T) {
	text := []string{"foo bar.baz  qux", "", "  end_word"}
	wrap := []string{"hello wor", "ld again"}
	for _, c := range []struct {
		name  string
		lines []string
		cont  []bool
		key   string
		from  wordPos
		want  wordPos
	}{
		{"w: next word", text, nil, "w", wordPos{0, 0}, wordPos{0, 4}},
		{"w: word to punctuation", text, nil, "w", wordPos{0, 4}, wordPos{0, 7}},
		{"w: punctuation to word", text, nil, "w", wordPos{0, 7}, wordPos{0, 8}},
		{"w: from mid-word, over two blanks", text, nil, "w", wordPos{0, 9}, wordPos{0, 13}},
		{"w: stops on an empty line", text, nil, "w", wordPos{0, 13}, wordPos{1, 0}},
		{"w: from an empty line, past indent", text, nil, "w", wordPos{1, 0}, wordPos{2, 2}},
		{"w: last word runs to the end", text, nil, "w", wordPos{2, 2}, wordPos{2, 9}},
		{"e: end of this word", text, nil, "e", wordPos{0, 0}, wordPos{0, 2}},
		{"e: on the end, the next one's", text, nil, "e", wordPos{0, 2}, wordPos{0, 6}},
		{"e: punctuation is a word", text, nil, "e", wordPos{0, 6}, wordPos{0, 7}},
		{"e: over the empty line", text, nil, "e", wordPos{0, 15}, wordPos{2, 9}},
		{"e: at the very end stays", text, nil, "e", wordPos{2, 9}, wordPos{2, 9}},
		{"b: start of the word before", text, nil, "b", wordPos{0, 4}, wordPos{0, 0}},
		{"b: from mid-word, its start", text, nil, "b", wordPos{0, 10}, wordPos{0, 8}},
		{"b: word to punctuation", text, nil, "b", wordPos{0, 8}, wordPos{0, 7}},
		{"b: stops on an empty line", text, nil, "b", wordPos{2, 2}, wordPos{1, 0}},
		{"b: from an empty line, the line above", text, nil, "b", wordPos{1, 0}, wordPos{0, 13}},
		{"b: at the start stays", text, nil, "b", wordPos{0, 0}, wordPos{0, 0}},
		{"w: across a wrap is one word", wrap, []bool{false, true}, "w", wordPos{0, 6}, wordPos{1, 3}},
		{"e: a wrapped word ends past the wrap", wrap, []bool{false, true}, "e", wordPos{0, 6}, wordPos{1, 1}},
		{"b: back over a wrap to the word start", wrap, []bool{false, true}, "b", wordPos{1, 1}, wordPos{0, 6}},
		{"w: a real line end splits the word", wrap, nil, "w", wordPos{0, 6}, wordPos{1, 0}},
	} {
		m := wordView(c.lines, c.cont)
		m.cursorLine, m.cursorCol = c.from.line, c.from.col
		m, _ = m.update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(c.key)})
		if got := (wordPos{m.cursorLine, m.cursorCol}); got != c.want {
			t.Errorf("%s: %s from %v → %v, want %v", c.name, c.key, c.from, got, c.want)
		}
	}
}

// The word motions extend a selection like any other motion (tdp K11: the mode's
// keys are pressed directly).
func TestWordMotionsExtendTheSelection(t *testing.T) {
	m := wordView([]string{"foo bar baz"}, nil)
	for _, k := range []string{"v", "w", "e"} {
		m, _ = m.update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(k)})
	}
	if got := m.selectionText(); got != "foo bar" {
		t.Errorf("v w e selected %q, want %q", got, "foo bar")
	}
}

// A word motion past the bottom of the box scrolls it, as j does.
func TestWordMotionScrolls(t *testing.T) {
	lines := make([]string, 60)
	for i := range lines {
		lines[i] = "word"
	}
	m := wordView(lines, nil)
	for range 40 {
		m, _ = m.update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("w")})
	}
	if m.cursorLine != 40 || m.scroll == 0 || m.cursorLine >= m.scroll+m.contentRows() {
		t.Errorf("cursor on line %d, box from %d, %d rows: the cursor should be in view", m.cursorLine, m.scroll, m.contentRows())
	}
}
