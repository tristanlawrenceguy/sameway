package trim

import "unicode"

// Where a title may be cut. Cut by characters, a title ended inside what a
// person sees as one: a family emoji, man, woman and girl joined, cut to a
// man and a woman; a flag's two letters cut to one, shown as a boxed P; an
// accent written as its own mark left behind without its letter. A cut
// moves back to where the next character starts something new.

// cutAt is the most characters of r, at most n, that end where a person
// would see one thing end.
func cutAt(r []rune, n int) int {
	if n >= len(r) {
		return len(r)
	}
	for n > 0 && !boundary(r, n) {
		n--
	}
	return n
}

// boundary says whether a cut before r[i] keeps every character whole.
func boundary(r []rune, i int) bool {
	next, prev := r[i], r[i-1]
	switch {
	case prev == zwj, joins(next):
		return false
	case regional(next):
		// A flag is two regional letters: inside one when an odd number of
		// them run up to the cut.
		n := 0
		for j := i - 1; j >= 0 && regional(r[j]); j-- {
			n++
		}
		return n%2 == 0
	}
	return true
}

const zwj = '‍'

// joins says a character belongs with the one before it: a joiner, a
// variation selector, a skin tone, a keycap, a tag, or a combining mark.
func joins(c rune) bool {
	switch {
	case c == zwj, c >= 0xFE00 && c <= 0xFE0F, c >= 0x1F3FB && c <= 0x1F3FF, c == 0x20E3, c >= 0xE0020 && c <= 0xE007F:
		return true
	}
	return unicode.In(c, unicode.Mn, unicode.Me, unicode.Mc)
}

func regional(c rune) bool { return c >= 0x1F1E6 && c <= 0x1F1FF }
