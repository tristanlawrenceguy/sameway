package trim

import (
	"strings"
	"testing"
)

// A title is cut at a word with an ellipsis, or left whole when it fits.
func TestTitleCutsAtAWordWithAnEllipsis(t *testing.T) {
	long := strings.Repeat("a", 160)
	for _, c := range []struct{ in, want string }{
		{"Call the dentist", "Call the dentist"},
		{"This is a note with a very long title", "This is a note with a…"},
		{"Exercise for thirty minutes every day", "Exercise for thirty minutes every day"},
		{"First line\nsecond line", "First line…"},
		{long, long[:MaxRunes-1] + "…"},
		{"Supercalifragilisticexpialidocious " + strings.Repeat("b", 60), "Supercalifragilisticexpialidocious…"},
	} {
		if got := Title(c.in); got != c.want {
			t.Errorf("Title(%q) = %q, want %q", c.in, got, c.want)
		}
		if n := len([]rune(Title(c.in))); n > MaxRunes {
			t.Errorf("Title(%q) is %d characters, more than %d", c.in, n, MaxRunes)
		}
	}
}
