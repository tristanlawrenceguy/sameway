package prose

import "testing"

// Plain words keep their sentence: bold before a full stop leaves no
// space before it, and a line break still parts two words.
func TestPlainKeepsTheSentence(t *testing.T) {
	t.Parallel()
	for md, want := range map[string]string{
		"I have asked for **Tuesday at three**.":            "I have asked for Tuesday at three.",
		"See [the page](https://example.com), then *call*!": "See the page, then call!",
		"One line  \nand the next":                          "One line and the next",
		"# Heading\n\nA paragraph.":                         "Heading A paragraph.",
	} {
		if got := Plain(md); got != want {
			t.Errorf("Plain(%q) = %q, want %q", md, got, want)
		}
	}
}
