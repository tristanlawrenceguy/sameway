package fold

import (
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/unicode/norm"
)

// Key is a name as two names are compared: two that look and sound the
// same have one key. That is the W3C's canonical case fold (normalise,
// fold case in full, normalise again), so Café typed on a Mac (é as e
// and an accent) and Café typed on Windows (é as one letter) are one
// name, and STRASSE is straße. Spaces are one space, and quotes and
// dashes as any keyboard types them, since a screen reader says Mum’s and
// Mum's alike. Accents and width are kept: Cafe and Café are two names.
func Key(s string) string {
	folded := cases.Fold().String(norm.NFD.String(s)) // a Caser is not shared between goroutines
	var b strings.Builder
	for i, w := range strings.Fields(folded) {
		if i > 0 {
			b.WriteByte(' ')
		}
		for _, r := range w {
			b.WriteRune(Plain(r))
		}
	}
	return norm.NFC.String(b.String())
}

// Same says whether two names are one name, as Key has it.
func Same(a, b string) bool { return a == b || Key(a) == Key(b) }
