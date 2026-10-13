package trim

import (
	"strings"
	"testing"
)

// Title cuts a long habit name at a word with an ellipsis but keeps the
// amount visible when the separator is ", " — natural language truncation.
func TestTitleCutsAtCommaToKeepAmountVisible(t *testing.T) {
	t.Parallel()
	for _, c := range []struct{ in, want string }{
		{"8 glasses of water a day, 1 glass", "8 glasses of water…, 1 glass"},
		{"Read, 25 minutes", "Read, 25 minutes"},
	} {
		if got := Title(c.in); got != c.want {
			t.Errorf("Title(%q) = %q, want %q", c.in, got, c.want)
		}
	}
	// A long comma-separated entry must cut at the comma to keep the amount.
	longHabit := strings.Repeat("very long habit name ", 3) + "habit"
	entryTitle := longHabit + ", 1 glass"
	got := Title(entryTitle)
	if !strings.Contains(got, "1 glass") {
		t.Errorf("truncation must keep the amount visible: got %q", got)
	}
	if n := len([]rune(got)); n > MaxRunes {
		t.Errorf("Title(%q) is %d characters, more than %d", entryTitle, n, MaxRunes)
	}
}
