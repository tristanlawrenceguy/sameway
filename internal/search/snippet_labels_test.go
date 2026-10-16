package search_test

import (
	"strings"
	"testing"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/search"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// Snippet body text does not contain raw field-label prefixes like "Status:",
// "Type:", or "State:" before enum values (acceptance 1, 3). The enum value
// itself is shown without the field name — e.g. "To do" instead of "Status: To
// do".
func TestSnippetBodyHasNoRawFieldLabelPrefix(t *testing.T) {
	t.Parallel()
	types, err := schema.Load("../../examples/workspaces/starter/schema")
	if err != nil {
		t.Fatal(err)
	}
	st, err := store.Open(":memory:", types)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	// Note with status = published.
	st.Create("note", map[string]any{
		"title":  "Garden plan",
		"body":   "Plant the roses in spring.",
		"status": "published",
	})

	hits, _ := search.Matches(st, types, "roses", search.Reader{Now: time.Now()})
	if len(hits) != 1 {
		t.Fatalf("one note matches 'roses', got %d hits", len(hits))
	}
	snippet := hits[0].Snippet

	// The raw field label prefix must not appear.
	for _, label := range []string{"Status:", "Type:", "State:"} {
		if strings.Contains(snippet, label) {
			t.Errorf("snippet body must not contain the raw field-label prefix %q; got: %q", label, snippet)
		}
	}

	// The enum value label itself should still appear somewhere in the text.
	if !strings.Contains(hits[0].Snippet, "Published") {
		t.Errorf("the status value 'Published' should appear in snippet body; got: %q", hits[0].Snippet)
	}
}
