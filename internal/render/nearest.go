package render

import "strings"

// Nearest is the one of among that name was most likely meant to be: the
// same but for case, the singular of a plural (tasks for task), or a slip
// of a letter or two. Empty when none is close, since a wrong guess sends
// whoever fixes the call the wrong way.
func Nearest(name string, among []string) string {
	want := strings.ToLower(strings.TrimSpace(name))
	if want == "" {
		return ""
	}
	best, bestDist := "", 3
	for _, c := range among {
		have := strings.ToLower(c)
		switch {
		case have == want, have+"s" == want, have+"es" == want, strings.TrimSuffix(have, "y")+"ies" == want:
			return c
		}
		// A short name is close only by one letter: fit and fix are not
		// one word mistyped.
		limit := 2
		if len(have) <= 4 || len(want) <= 4 {
			limit = 1
		}
		if d := distance(want, have); d <= limit && d < bestDist {
			best, bestDist = c, d
		}
	}
	return best
}

// distance is the edit distance between two words: letters put in, taken
// out or changed.
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
