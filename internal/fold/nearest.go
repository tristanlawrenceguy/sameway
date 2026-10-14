package fold

import "strings"

// Nearest is the one of among that name was most likely meant to be, for
// "did you mean": the same as found (case and accents aside, as String
// has it), the singular of a plural (tasks for task), or a slip of a letter
// or two. Empty when none is close, since a wrong guess sends whoever
// fixes the call the wrong way. A short word is close only by one letter,
// and never by half of it: fit and fix, ad and add are not one word
// mistyped. One helper, so the API, MCP, the CLI and the assistant
// suggest alike.
func Nearest(name string, among []string) string {
	want := String(strings.TrimSpace(name))
	if want == "" {
		return ""
	}
	best, bestDist := "", 3
	for _, c := range among {
		have := String(c)
		switch {
		case have == want, have+"s" == want, have+"es" == want, strings.TrimSuffix(have, "y")+"ies" == want:
			return c
		}
		limit := 2
		if len([]rune(have)) <= 4 || len([]rune(want)) <= 4 {
			limit = 1
		}
		if d := distance(want, have); d <= limit && d < bestDist && d*2 < len([]rune(want)) {
			best, bestDist = c, d
		}
	}
	return best
}

// distance is the edit distance between two words, by letter: letters
// put in, taken out or changed.
func distance(a, b string) int {
	ra, rb := []rune(a), []rune(b)
	prev := make([]int, len(rb)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(ra); i++ {
		cur := make([]int, len(rb)+1)
		cur[0] = i
		for j := 1; j <= len(rb); j++ {
			cost := 1
			if ra[i-1] == rb[j-1] {
				cost = 0
			}
			cur[j] = min(prev[j]+1, cur[j-1]+1, prev[j-1]+cost)
		}
		prev = cur
	}
	return prev[len(rb)]
}
