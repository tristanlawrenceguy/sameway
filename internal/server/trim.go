package server

import "strings"

// Shortening words for places that have one line.

// maxTitleRunes is the character ceiling for a heading title. Six medium
// words fit comfortably under 80 runes, so this catches runaway single-word
// titles while leaving normal six-word titles alone.
const maxTitleRunes = 75

// trimLabel cuts a label to at most three words.
func trimLabel(s string) string {
	fields := strings.Fields(s)
	if len(fields) <= 3 {
		return s
	}
	return strings.Join(fields[:3], " ")
}

// trimWords cuts a string to at most n words.
func trimWords(s string, n int) string {
	fields := strings.Fields(s)
	if len(fields) <= n {
		return s
	}
	return strings.Join(fields[:n], " ")
}

// trimTitle cuts a title to at most six words and at most maxTitleRunes
// characters, for where it has one line: a window title, a row, a crumb.
// A page's own heading is the whole title. An ellipsis says it was cut.
func trimTitle(s string) string {
	fields := strings.Fields(s)
	if len(fields) <= 6 && len([]rune(s)) <= maxTitleRunes {
		return s
	}
	if len(fields) > 6 {
		return strings.Join(fields[:6], " ") + "…"
	}
	// ≤ 6 words but too many characters: cut at the last space that fits,
	// so no word is left half said; a single long word is cut where it must.
	runes := []rune(s)
	cut := string(runes[:maxTitleRunes-1])
	if i := strings.LastIndex(cut, " "); i > 0 {
		cut = cut[:i]
	}
	return cut + "…"
}
