package search

import (
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/unicode/norm"
)

// Forgiving words. A search is read the way a person means it, not letter
// for letter: case and accents do not matter, so cafe finds café, and a
// plural finds its one, so plumbers finds the plumber.

// fold is text with case and accents taken off.
func fold(s string) string {
	folded, _ := foldIndex(s)
	return folded
}

// foldIndex is text folded, with where each byte of the folded text came
// from in the text as it was, so a place found in one is a place in the
// other, whatever folding did to the lengths.
func foldIndex(s string) (string, []int) {
	var b strings.Builder
	var at []int
	for i, r := range s {
		for _, f := range norm.NFD.String(string(unicode.ToLower(r))) {
			if unicode.Is(unicode.Mn, f) {
				continue
			}
			n := utf8.RuneLen(f)
			b.WriteRune(f)
			for k := 0; k < n; k++ {
				at = append(at, i)
			}
		}
	}
	return b.String(), at
}

// Words is a search as the words it looks for: folded, and a plural taken
// back to its one.
func Words(q string) []string {
	var out []string
	for _, w := range strings.Fields(fold(q)) {
		if len(w) > 3 && strings.HasSuffix(w, "s") && !strings.HasSuffix(w, "ss") {
			w = strings.TrimSuffix(w, "s")
		}
		out = append(out, w)
	}
	return out
}

// Spans are where words are found in text, as byte ranges of the text as
// it is, in order and not overlapping: for showing the words found.
func Spans(text string, words []string) [][2]int {
	folded, at := foldIndex(text)
	var spans [][2]int
	for _, w := range words {
		if w == "" {
			continue
		}
		for from := 0; ; {
			i := strings.Index(folded[from:], w)
			if i < 0 {
				break
			}
			start, end := from+i, from+i+len(w)
			last := len(text)
			if end < len(at) {
				last = at[end]
			}
			spans = append(spans, [2]int{at[start], last})
			from = end
		}
	}
	sort.Slice(spans, func(i, j int) bool { return spans[i][0] < spans[j][0] })
	var out [][2]int
	for _, s := range spans {
		if n := len(out); n > 0 && s[0] < out[n-1][1] {
			if s[1] > out[n-1][1] {
				out[n-1][1] = s[1]
			}
			continue
		}
		out = append(out, s)
	}
	return out
}
