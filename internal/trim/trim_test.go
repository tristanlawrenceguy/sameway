package trim

import (
	"strings"
	"testing"
	"unicode/utf8"
)

// A title is cut at a word with an ellipsis, or left whole when it fits.
func TestTitleCutsAtAWordWithAnEllipsis(t *testing.T) {
	t.Parallel()
	long := strings.Repeat("a", 160)
	for _, c := range []struct{ in, want string }{
		{"Call the dentist", "Call the dentist"},
		{"This is a note with a very long title", "This is a note with a…"},
		{"Exercise for thirty minutes every day", "Exercise for thirty minutes every day"},
		{"First line\nsecond line", "First line…"},
		{"8 glasses of water a day: 1 glass", "8 glasses of water…: 1 glass"},
		{"Search: one two three four five six seven", "Search: one two three four five…"},
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

// Clip, Line and Flat count characters, so é and an emoji are never split,
// and say with an ellipsis, within n, that they cut.
func TestClipCountsCharactersNotBytes(t *testing.T) {
	t.Parallel()
	for _, c := range []struct{ got, want string }{
		{Clip("café au lait", 5), "café…"},
		{Clip("éééééé", 4), "ééé…"},
		{Clip("🎉🎉🎉🎉", 3), "🎉🎉…"},
		{Clip("short", 5), "short"},
		{Clip("two words", 5), "two…"},
		{Line("  première ligne\nsecond", 40), "première ligne"},
		{Line("ééééé\nx", 3), "éé…"},
		{Flat("a\n  b\tc 🎉🎉🎉", 7), "a b c…"},
	} {
		if c.got != c.want {
			t.Errorf("got %q, want %q", c.got, c.want)
		}
	}
	for _, s := range []string{"ééééééééé", "🎉🎉🎉🎉🎉🎉", "naïve résumé café"} {
		for n := 1; n < 8; n++ {
			if got := Clip(s, n); !utf8.ValidString(got) || len([]rune(got)) > n {
				t.Errorf("Clip(%q, %d) = %q", s, n, got)
			}
		}
	}
}
