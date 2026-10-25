package server_test

// Search result snippet text must label state/status enum values with a visible
// "Status:" prefix in both HTML and JSON surfaces. Acceptance item 2 of task 0708.

import (
	"net/http"
	"strings"
	"testing"
)

// TestSearchResultAPIHasLabeledStatus verifies that the JSON API also returns
// snippets with "Status:" prefix before status values, not bare enum words.
func TestSearchResultAPIHasLabeledStatus(t *testing.T) {
	t.Parallel()
	_, h := newApp(t)

	var rec struct{ ID string }
	decode(t, postJSON(t, h, http.MethodPost, "/api/note", map[string]any{
		"title":  "API status label check",
		"body":   "This note is about the kitchen sink.",
		"status": "draft",
	}), &rec)

	var out struct {
		Count int `json:"count"`
		Hits  []struct {
			Type    string `json:"type"`
			Title   string `json:"title"`
			Snippet string `json:"snippet,omitempty"`
			Href    string `json:"href"`
		}
	}
	decode(t, get(t, h, "/api/search?q=kitchen"), &out)

	if out.Count == 0 {
		t.Fatalf("API search should find at least one result; got count %d", out.Count)
	}

	hit := out.Hits[0]

	// The snippet must contain "Status:" as a visible prefix.
	if !strings.Contains(hit.Snippet, "Status:") {
		t.Errorf("API search snippet must show 'Status:' label before status value; got: %q", hit.Snippet)
	}

	// If the snippet contains "Draft" it should not be bare after middots.
	if strings.Contains(hit.Snippet, `· Draft`) || (strings.Contains(hit.Snippet, "Draft") && !strings.Contains(hit.Snippet, "Status:")) {
		t.Errorf("API search snippet must not show bare 'Draft' — it should have a 'Status:' prefix; got: %q", hit.Snippet)
	}

	if hit.Title != "API status label check" {
		t.Errorf("API search result title mismatch; got: %q", hit.Title)
	}
}

// TestSearchResultHTMLNoTypeMetaInBody verifies that the HTML search page body does not
// contain a bare type-name word floating in the card body text — acceptance item 1.
func TestSearchResultHTMLNoTypeMetaInBody(t *testing.T) {
	t.Parallel()
	_, h := newApp(t)

	var rec struct{ ID string }
	decode(t, postJSON(t, h, http.MethodPost, "/api/note", map[string]any{
		"title": "Note without meta check",
		"body":  "A note about nothing.",
	}), &rec)

	body := get(t, h, "/search?q=nothing").Body.String()

	if !strings.Contains(body, `href="/t/note/`+rec.ID) {
		t.Fatalf("could not find search result for note; found in:\n%s", truncate(body))
	}

	// The card body must not contain a bare "Note" word standing alone as the only content.
	if strings.Contains(body, ">Note</p>") || strings.Contains(body, "<p>Note") {
		t.Errorf("HTML search result must not contain a bare type-name paragraph; found in:\n%s", truncate(body))
	}

	if !strings.Contains(body, "sw-card__body") {
		t.Error("search result card must show its body/snippet")
	}
}
