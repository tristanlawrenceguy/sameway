// Package trim shortens a title for a place that has one line: a window
// title, a row, a crumb, a log entry, a button's hidden name. It is the one
// rule for that, so every place cuts the same title the same way. A page's
// own heading is the whole title and does not come here.
package trim

import "strings"

// MaxWords and MaxRunes bound a shortened title. Six medium words fit well
// under 75 characters, so the second only catches a runaway single word.
const (
	MaxWords = 6
	MaxRunes = 75
)

// Title gives the first line of s, cut to at most MaxWords words and
// MaxRunes characters. It cuts at a word, so none is left half said, and
// an ellipsis says it was cut; a single long word is cut where it must.
func Title(s string) string {
	line, rest, _ := strings.Cut(strings.TrimSpace(s), "\n")
	line = strings.TrimSpace(line)
	cut := rest != ""
	if fields := strings.Fields(line); len(fields) > MaxWords {
		line, cut = strings.Join(fields[:MaxWords], " "), true
	}
	if runes := []rune(line); len(runes) > MaxRunes {
		line, cut = string(runes[:MaxRunes-1]), true
		if i := strings.LastIndex(line, " "); i > 0 {
			line = line[:i]
		}
	}
	if cut {
		return strings.TrimRight(line, " ") + "…"
	}
	return line
}
