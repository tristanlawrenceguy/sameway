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
		// "Name: how much", an entry, is cut in its name, so a list of
		// them does not read as the same cut-off name over and over.
		if head, tail, ok := strings.Cut(line, ": "); ok && !cut {
			words := strings.Fields(head)
			if n := MaxWords - len(strings.Fields(tail)); n >= 3 && n < len(words) {
				if short := strings.Join(words[:n], " ") + "…: " + tail; len([]rune(short)) <= MaxRunes {
					return short
				}
			}
		}
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

// Clip cuts s to at most n characters, counted as characters and not
// bytes, so an accent or an emoji is never split. Like Title, an ellipsis
// says it was cut and counts towards n. Unlike Title it cuts where it must,
// for text that is not a title: a log line, a quoted value, an error.
func Clip(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	if n < 1 {
		n = 1
	}
	return strings.TrimRight(string(r[:n-1]), " ") + "…"
}

// Line is the first line of s, clipped to n characters.
func Line(s string, n int) string {
	line, _, _ := strings.Cut(s, "\n")
	return Clip(strings.TrimSpace(line), n)
}

// Flat is all of s on one line, its runs of space and line breaks made one
// space, clipped to n characters: what a notification or a status says.
func Flat(s string, n int) string {
	return Clip(strings.Join(strings.Fields(s), " "), n)
}
