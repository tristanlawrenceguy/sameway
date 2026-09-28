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
// it is, in order and not overlapping: for showing the words found. A span
// is the whole word a match falls in, so groceries, found as grocerie, is
// marked groceries and not grocerie with its s left over.
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
			spans = append(spans, wordAround(text, at[start], last))
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

// wordAround widens a byte range of text to the word it falls in: back to
// where the word starts and on to where it ends, a letter at a time, so a
// letter of more than one byte, or an accent kept as its own mark, is never
// cut.
func wordAround(text string, start, end int) [2]int {
	for start > 0 {
		r, n := utf8.DecodeLastRuneInString(text[:start])
		if !inWord(r) {
			break
		}
		start -= n
	}
	for end < len(text) {
		r, n := utf8.DecodeRuneInString(text[end:])
		if !inWord(r) {
			break
		}
		end += n
	}
	return [2]int{start, end}
}

// inWord says whether a letter belongs to a word: letters, digits, and the
// marks that sit on them. Writing without spaces between its words, such
// as Chinese or Japanese, has no word to widen to, so a match there is
// marked as found.
func inWord(r rune) bool {
	if unicode.In(r, unicode.Han, unicode.Hiragana, unicode.Katakana, unicode.Thai) {
		return false
	}
	return unicode.IsLetter(r) || unicode.IsDigit(r) || unicode.Is(unicode.Mn, r)
}
