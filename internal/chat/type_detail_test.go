package chat_test

import (
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/records"
)

// Sentence resolves raw type identifiers in an activity entry's detail to
// human-readable display names when the target is "type". This covers
// acceptance items 1 and 2: API summaries must say "System added type Test Type",
// not "System added type test_type". The fix lives in records/names.go where
// Sentence() calls schema.DisplayName for type-setting detail.
func TestSentenceResolvesTypeDetail(t *testing.T) {
	for _, c := range []struct {
		fields map[string]any
		want   string
	}{
		{
			map[string]any{"actor": "system", "action": "added", "target": "type", "detail": "test_type"},
			"System added type Test Type",
		},
		{
			map[string]any{"actor": "system", "action": "added", "target": "type", "detail": "meeting_notes_template"},
			"System added type Meeting Notes Template",
		},
		{
			map[string]any{"actor": "system", "action": "added", "target": "field", "detail": "test_field on test_type"},
			"System added field Test Field on Test Type",
		},
	} {
		st := newFullService(t).Store
		got := records.Sentence(st, c.fields)
		if got != c.want {
			t.Errorf("Sentence(%v) = %q, want %q", c.fields, got, c.want)
		}
	}
}
