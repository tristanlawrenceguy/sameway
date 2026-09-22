package update

import (
	"strconv"
	"strings"
)

// Newer says whether want is a later version than have. A version is
// major.minor.patch, with or without a leading v, and what follows a - or
// a + is a prerelease that sorts before the release of the same numbers.
//
// A version that does not parse is never newer and is never replaced: a
// build that cannot say what it is does not guess.
func Newer(have, want string) bool {
	h, hpre, ok := parse(have)
	if !ok {
		return false
	}
	w, wpre, ok := parse(want)
	if !ok {
		return false
	}
	for i := range h {
		if w[i] != h[i] {
			return w[i] > h[i]
		}
	}
	// The same numbers: the release beats a prerelease of it, and between
	// two prereleases the later name wins, so rc2 comes after rc1.
	switch {
	case wpre == hpre:
		return false
	case wpre == "":
		return true
	case hpre == "":
		return false
	}
	return wpre > hpre
}

// Known says whether a version can be compared at all.
func Known(v string) bool {
	_, _, ok := parse(v)
	return ok
}

// parse reads major.minor.patch and whatever follows a - or a +. A part
// left off is zero, so "0.4" is 0.4.0.
func parse(v string) (nums [3]int, pre string, ok bool) {
	v = strings.TrimPrefix(strings.TrimSpace(v), "v")
	if i := strings.IndexAny(v, "-+"); i >= 0 {
		v, pre = v[:i], v[i+1:]
	}
	parts := strings.Split(v, ".")
	if len(parts) > 3 {
		return nums, "", false
	}
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 {
			return nums, "", false
		}
		nums[i] = n
	}
	return nums, pre, true
}
