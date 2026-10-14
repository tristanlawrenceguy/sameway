// Package fold is how Sameway compares words, in one place, so search, a
// condition, a list's order and two names that look the same all agree.
// It has three rules, for three jobs:
//
//   - String, for finding: case, accents and width taken off, quotes and
//     dashes as any keyboard types them, so cafe finds Café, sao finds
//     São, strasse finds Straße and lodz finds Łódź. Search and a
//     condition on words (title~sao) both read words this way.
//   - Key, for one name: two names that look and sound the same are one
//     name, in any case and whatever form their letters are stored in. It
//     keeps accents: Cafe and Café are two names.
//   - Compare, for order: the Unicode collation order, so Émile sorts
//     with E and Item 2 comes before Item 10.
//
// The rules and their sources are in design/foundations/text.md.
package fold

import (
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

// String is s as a person means it when they look for it: case, accents
// and width taken off.
func String(s string) string {
	var b strings.Builder
	for _, r := range s {
		Each(r, func(f rune) { b.WriteRune(f) })
	}
	return b.String()
}

// Each gives what one character of s folds to, in order, for String and
// for search's index, which keeps where each came from. A character folds
// to none (an accent), one, or several (ß to ss).
func Each(r rune, emit func(rune)) {
	// Compatibility decomposition takes width and ligatures off as well as
	// accents: a full-width ＣＡＦＥ from a Japanese keyboard, ﬁ from a
	// PDF. Search is the one place this is wanted (UTR #30, ICU folding).
	for _, f := range norm.NFKD.String(string(r)) {
		if unicode.Is(unicode.Mn, f) {
			continue
		}
		f = unicode.ToLower(f)
		if s, ok := spelled[f]; ok {
			for _, p := range s {
				emit(p)
			}
			continue
		}
		emit(Plain(f))
	}
}

// spelled is letters that are not a letter with an accent to Unicode,
// but that a person without them on their keyboard types as plain
// letters: Straße as strasse, Łódź as lodz, København as kobenhavn, and
// Turkish dotless ı as i. Capitals are lowered to these first.
var spelled = map[rune]string{
	'ß': "ss", 'ı': "i", 'ł': "l", 'ø': "o", 'đ': "d", 'ħ': "h",
	'æ': "ae", 'œ': "oe", 'ς': "σ",
}

// Plain is a character as it is typed on any keyboard: a phone turns
// ' into ’ and " into “”, and a word processor - into – or —, so "Mum’s
// birthday" typed on a phone missed "Mum's birthday" typed on a laptop.
// A non-breaking space is a space.
func Plain(r rune) rune {
	switch r {
	case '‘', '’', '‚', '‛', '′', 'ʼ':
		return '\''
	case '“', '”', '„', '‟', '″':
		return '"'
	case '‐', '‑', '‒', '–', '—', '―', '−':
		return '-'
	case ' ', ' ', ' ':
		return ' '
	}
	return r
}
