package automate

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
)

// A suggestion's line is kept and read days later, so its day is said in
// full, never "tomorrow", and named by its field's label, not its stored
// name ("due", never "at").
func TestASuggestionSaysItsDayInFull(t *testing.T) {
	t.Parallel()
	types, err := schema.Load(filepath.Join("..", "..", "examples", "workspaces", "starter", "schema"))
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 10, 14, 9, 0, 0, 0, time.UTC)
	s := &Service{Book: &records.Book{Clock: func() time.Time { return now }}}
	cases := []struct {
		typ    string
		fields map[string]any
		want   string
	}{
		{"task", map[string]any{"title": "Buy milk", "due": "2026-10-15T17:00:00Z"}, "Buy milk, due Thu 15 Oct 2026 at 5pm"},
		{"reminder", map[string]any{"title": "Call mum", "at": "2026-10-15T07:00:00Z"}, "Call mum, rings Thu 15 Oct 2026 at 7am"},
	}
	for _, c := range cases {
		typ, _ := types.Get(c.typ)
		if got := s.suggestedWords(typ, c.fields); got != c.want {
			t.Errorf("a suggested %s says %q, want %q", c.typ, got, c.want)
		}
	}
}
