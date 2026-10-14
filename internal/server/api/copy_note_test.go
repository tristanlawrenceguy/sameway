package api_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

// A list an agent writes of notes' titles is told, in the answer's notes
// and first, that it will go stale and what shows them current, with how
// many different notes it names; told the same again when it is changed.
func TestAListOfNotesTitlesIsToldItWillGoStale(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)
	var items []any
	for _, title := range []string{"Seeds to buy", "Beds to dig", "Tools to mend", "Seeds to buy"} {
		a.Store.Create("note", map[string]any{"title": title, "status": "draft"})
		items = append(items, title)
	}
	var made struct {
		ID    string   `json:"id"`
		Notes []string `json:"notes"`
	}
	json.Unmarshal(postJSON(t, h, http.MethodPost, "/api/block", map[string]any{"component": "list", "props": map[string]any{"label": "Drafts", "items": items}}).Body.Bytes(), &made)
	if len(made.Notes) == 0 || !strings.HasPrefix(made.Notes[0], `This "Drafts" list will go stale: it names 3 different notes`) || !strings.Contains(made.Notes[0], "collection block of note") {
		t.Fatalf("the list is told it will go stale, first: %v", made.Notes)
	}
	var changed struct {
		Notes []string `json:"notes"`
	}
	json.Unmarshal(postJSON(t, h, http.MethodPatch, "/api/block/"+made.ID, map[string]any{"span": 12}).Body.Bytes(), &changed)
	if len(changed.Notes) == 0 || !strings.Contains(changed.Notes[0], "will go stale") {
		t.Errorf("changed, it is told again: %v", changed.Notes)
	}
}
