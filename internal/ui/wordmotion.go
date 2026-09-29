package ui

import "unicode"

// Word motions for the preview viewport (w, e, b), as vim moves over words: a
// word is a run of word characters (letters, digits, _) or a run of other
// punctuation, blanks between them. They cross line ends — a line end is a
// blank, and an empty line is a word of its own (w and b stop on it) — but not
// a hard wrap: a continuation line (cont) goes on from the line above, so a word
// the display split in two is still one word.

// wordPos is a place in the viewport text. col == len(line) is the line end
// (a blank); on an empty line that is col 0.
type wordPos struct{ line, col int }

// wordClass sorts a character: 0 a blank, 1 a word character, 2 punctuation.
func wordClass(r rune) int {
	switch {
	case unicode.IsSpace(r):
		return 0
	case r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r):
		return 1
	}
	return 2
}

func (m detailYank) lineLen(l int) int { return len([]rune(m.plain[l])) }

// at is the character at p: '\n' at a line end.
func (m detailYank) at(p wordPos) rune {
	if r := []rune(m.plain[p.line]); p.col < len(r) {
		return r[p.col]
	}
	return '\n'
}

func (m detailYank) emptyLine(p wordPos) bool { return m.lineLen(p.line) == 0 }

// fwd moves p one place on; false at the end of the text. The last character of
// a line steps to its line end, or straight onto the next line when that one
// continues it.
func (m detailYank) fwd(p wordPos) (wordPos, bool) {
	n := m.lineLen(p.line)
	switch {
	case p.col < n-1:
		return wordPos{p.line, p.col + 1}, true
	case p.line >= m.lastLine():
		return p, false
	case p.col == n-1 && !m.contAt(p.line+1):
		return wordPos{p.line, n}, true
	}
	return wordPos{p.line + 1, 0}, true
}

// bwd moves p one place back; false at the start of the text.
func (m detailYank) bwd(p wordPos) (wordPos, bool) {
	switch {
	case p.col > 0:
		return wordPos{p.line, p.col - 1}, true
	case p.line == 0:
		return p, false
	}
	prev := p.line - 1
	n := m.lineLen(prev)
	if m.contAt(p.line) && n > 0 {
		return wordPos{prev, n - 1}, true
	}
	return wordPos{prev, n}, true
}

// textEnd is the last character of the text, where a motion that runs out stops.
func (m detailYank) textEnd() wordPos {
	return wordPos{m.lastLine(), m.lastCol(m.lastLine())}
}

// wordForward is w: the start of the next word, or the next empty line.
func (m detailYank) wordForward(p wordPos) wordPos {
	start := p
	if cls := wordClass(m.at(p)); cls != 0 {
		for wordClass(m.at(p)) == cls { // the rest of this word
			q, ok := m.fwd(p)
			if !ok {
				return m.textEnd()
			}
			p = q
		}
	}
	for wordClass(m.at(p)) == 0 {
		if m.emptyLine(p) && p != start {
			return p
		}
		q, ok := m.fwd(p)
		if !ok {
			return m.textEnd()
		}
		p = q
	}
	return p
}

// wordEnd is e: the end of this word if the cursor is not already on it, else
// of the next one.
func (m detailYank) wordEnd(p wordPos) wordPos {
	q, ok := m.fwd(p)
	if !ok {
		return p
	}
	p = q
	for wordClass(m.at(p)) == 0 {
		if q, ok = m.fwd(p); !ok {
			return m.textEnd()
		}
		p = q
	}
	cls := wordClass(m.at(p))
	for {
		q, ok := m.fwd(p)
		if !ok || wordClass(m.at(q)) != cls {
			return p
		}
		p = q
	}
}

// wordBack is b: the start of this word if the cursor is not already on it,
// else of the one before, or an empty line on the way.
func (m detailYank) wordBack(p wordPos) wordPos {
	q, ok := m.bwd(p)
	if !ok {
		return p
	}
	p = q
	for wordClass(m.at(p)) == 0 {
		if m.emptyLine(p) {
			return p
		}
		if q, ok = m.bwd(p); !ok {
			return wordPos{0, 0}
		}
		p = q
	}
	cls := wordClass(m.at(p))
	for {
		q, ok := m.bwd(p)
		if !ok || wordClass(m.at(q)) != cls {
			return p
		}
		p = q
	}
}
