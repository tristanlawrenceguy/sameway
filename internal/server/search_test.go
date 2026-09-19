package server_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/look"
)

// A person searches on one page and lands on what they found; an agent
// gets the same hits as JSON; the page reads cleanly to a screen reader.
func TestOneSearchOverEverything(t *testing.T) {
	_, h := newApp(t)
	wantStatus(t, postJSON(t, h, http.MethodPost, "/api/note", map[string]any{"title": "Call the plumber", "body": "About the kitchen tap."}), http.StatusCreated)
	wantStatus(t, postJSON(t, h, http.MethodPost, "/api/note", map[string]any{"title": "Garden", "body": "Weed the beds."}), http.StatusCreated)

	page := get(t, h, "/search?q=plumber")
	wantStatus(t, page, http.StatusOK)
	body := page.Body.String()
	if !strings.Contains(body, "Call the plumber") || strings.Contains(body, "Garden") || !strings.Contains(body, "1 thing found") {
		t.Errorf("the page should show the one hit and say so: %.400s", body)
	}
	if !strings.Contains(body, `role="search"`) || !strings.Contains(body, `value="plumber"`) {
		t.Error("the search form is a search landmark and keeps the words typed")
	}
	if o, _ := look.Page(body); len(o.Problems) != 0 {
		t.Errorf("the search page should read cleanly, got %v", o.Problems)
	}
	if empty := get(t, h, "/search?q=zebra").Body.String(); !strings.Contains(empty, "Nothing has zebra in it") {
		t.Error("no hits should say so in plain words")
	}
	if home := get(t, h, "/").Body.String(); !strings.Contains(home, `data-component="search"`) {
		t.Error("search should be reachable from the canvas, as the block in the header")
	}

	var out struct {
		Count int
		Hits  []struct{ Type, Title, Href, Snippet string }
	}
	decode(t, get(t, h, "/api/search?q=kitchen"), &out)
	if out.Count != 1 || out.Hits[0].Title != "Call the plumber" || !strings.Contains(out.Hits[0].Snippet, "kitchen") {
		t.Errorf("the API should give the same hit with its snippet, got %+v", out)
	}
}

// Empty state for zero-result searches shows an h2 heading and a helpful
// message that explains nothing matched and suggests trying different words.
func TestSearchEmptyStateHasH2AndSuggestion(t *testing.T) {
	_, h := newApp(t)

	page := get(t, h, "/search?q=nonexistent")
	wantStatus(t, page, http.StatusOK)
	body := page.Body.String()

	if !strings.Contains(body, "<h2>") {
		t.Error("empty search should have an h2 heading for the no-results state")
	}
	if strings.Contains(body, `<h1>Search`) && !strings.Contains(body, `Search: nonexistent`) {
		t.Error("search with a query should show 'Search: <query>' as the h1 title")
	}
	if !strings.Contains(body, "Nothing has") || !strings.Contains(body, "nonexistent") {
		t.Error("empty state should mention the query term so the person knows what was searched")
	}
	if !strings.Contains(strings.ToLower(body), "try") || !strings.Contains(strings.ToLower(body), "different words") && !strings.Contains(strings.ToLower(body), "spelling") {
		t.Error("empty state should suggest trying different words or checking spelling")
	}
}

// Search result links include the content type in their accessible name so a
// screen reader user knows what kind of thing they are clicking before navigating.
func TestSearchResultLinksIncludeContentTypeInAccessibleName(t *testing.T) {
	_, h := newApp(t)
	wantStatus(t, postJSON(t, h, http.MethodPost, "/api/note", map[string]any{"title": "Call the plumber", "body": "About the kitchen tap."}), http.StatusCreated)

	page := get(t, h, "/search?q=plumber")
	wantStatus(t, page, http.StatusOK)
	body := page.Body.String()

	// The link for "Call the plumber" should include its content type.
	if !strings.Contains(body, `Call the plumber`) {
		t.Error("result link should contain the record title")
	}
	if !strings.Contains(body, `<span class="sw-visually-hidden">`) {
		t.Error("result links should embed the content type in a visually hidden span so screen readers announce it as part of the accessible name")
	}
	// The card title link should have the pattern: Title<span class="sw-visually-hidden"> — note</span> or similar.
	if !strings.Contains(body, "— Note") && !strings.Contains(body, "— note") {
		t.Error("result links should include both the record type and its title in the accessible name, e.g., 'Call the plumber — Note'")
	}
}

// Search results page has proper heading structure: h1 for the page title,
// h2 for the "Results" section before the list of hits.
func TestSearchResultsPageHeadingStructure(t *testing.T) {
	_, h := newApp(t)
	wantStatus(t, postJSON(t, h, http.MethodPost, "/api/note", map[string]any{"title": "Call the plumber", "body": "About the kitchen tap."}), http.StatusCreated)

	page := get(t, h, "/search?q=plumber")
	wantStatus(t, page, http.StatusOK)
	body := page.Body.String()

	if o, _ := look.Page(body); len(o.Problems) != 0 {
		t.Errorf("the search results page should have clean heading structure, got %v", o.Problems)
	}

	// Check for h2 "Results" section before the list.
	if !strings.Contains(body, `<h2>Results</h2>`) && !strings.Contains(body, "<h2>Results") {
		t.Error("search results page should have an h2 'Results' heading before the list of hits")
	}

	// The page title (in the h1) should include the query when there is one.
	if !strings.Contains(body, "Search: plumber") && !strings.Contains(body, "<h1>Search: plumber</h1>") {
		t.Error("search results page h1 should say 'Search: <query>' so screen readers announce what was searched")
	}

	// The existing look.Page check also verifies no heading-level skips.
	// If there is an h2 "Results" between h1 and the card titles, that is correct.
}
