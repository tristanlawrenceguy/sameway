// Package fold is text as a person means it when they look for it: case
// and accents taken off, so cafe finds café and Sao finds São. Search
// folded; a condition on a list (title~sao) did not, so the same words
// found a note in search and missed it in a collection, a list page and
// the API (where=title~sao). Both fold here now.
package fold

import (
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

// String is s with case and accents taken off.
func String(s string) string {
	var b strings.Builder
	for _, r := range s {
		for _, f := range norm.NFD.String(string(unicode.ToLower(r))) {
			if !unicode.Is(unicode.Mn, f) {
				b.WriteRune(f)
			}
		}
	}
	return b.String()
}
