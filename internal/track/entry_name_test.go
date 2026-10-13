package track

import (
	"testing"
	"time"
)

// EntryName says a habit and its amount with a comma, not a colon —
// natural language rather than machine format. This covers acceptance items 1
// (natural language headings) and 3 (distinguishable entries). For an entry
// without a habit it falls back to the note or "Entry, <date>".
func TestEntryNameUsesCommaNotColon(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		habit, unit string
		fields      map[string]any
		want        string
	}{
		{"8 glasses of water a day", "glasses", map[string]any{"amount": 1.0}, "8 glasses of water a day, 1 glass"},
		{"Read", "minutes", map[string]any{"amount": 25.0}, "Read, 25 minutes"},
		{"Stretch", "", map[string]any{"amount": 1.0}, "Stretch, done"},
		{"Drink tea", "", map[string]any{"amount": 1.0}, "Drink tea, done"},
	} {
		got := EntryName(c.habit, c.unit, c.fields, time.Local)
		if got != c.want {
			t.Errorf("EntryName(%q, %q, %v) = %q, want %q", c.habit, c.unit, c.fields, got, c.want)
		}
		// The separator must be a comma, never a colon.
		if i := len(c.habit); i > 0 && got[i] == ':' {
			t.Errorf("EntryName uses a colon as separator: %q", got)
		}
	}
}
