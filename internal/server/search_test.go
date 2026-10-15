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
	t.Parallel()
	_, h := newApp(t)
	wantStatus(t, postJSON(t, h, http.MethodPost, "/api/note", map[string]any{"title": "Call the plumber", "body": "About the kitchen tap."}), http.StatusCreated)
	wantStatus(t, postJSON(t, h, http.MethodPost, "/api/note", map[string]any{"title": "Garden", "body": "Weed the beds."}), http.StatusCreated)

	page := get(t, h, "/search?q=plumber")
	wantStatus(t, page, http.StatusOK)
	body := page.Body.String()
	if !strings.Contains(body, `Call the <mark class="sw-search__hit">plumber</mark>`) || strings.Contains(body, "Garden") || !strings.Contains(body, "1 thing found") {
		t.Errorf("the page should show the one hit and say so: %.400s", body)
	}
	if !strings.Contains(body, `role="search"`) || !strings.Contains(body, `value="plumber"`) {
		t.Error("the search form is a search landmark and keeps the words typed")
	}
	if o, _ := look.Page(body); len(o.Problems) != 0 {
		t.Errorf("the search page should read cleanly, got %v", o.Problems)
	}
	if empty := get(t, h, "/search?q=zebra").Body.String(); !strings.Contains(strings.ToLower(empty), "no results") && !strings.Contains(strings.ToLower(empty), "try different words") {
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
	t.Parallel()
	_, h := newApp(t)

	page := get(t, h, "/search?q=nonexistent")
	wantStatus(t, page, http.StatusOK)
	body := page.Body.String()

	if !strings.Contains(body, "<h2 ") && !strings.Contains(body, "<h2>") {
		t.Error("empty search should have an h2 heading for the no-results state")
	}
	if strings.Contains(body, `<h1>Search`) && !strings.Contains(body, `Search: nonexistent`) {
		t.Error("search with a query should show 'Search: <query>' as the h1 title")
	}
	if !strings.Contains(body, "nonexistent") && !strings.Contains(body, `%q`) {
		t.Error("empty state should mention the query term so the person knows what was searched")
	}
	if !strings.Contains(strings.ToLower(body), "try") || !strings.Contains(strings.ToLower(body), "different words") && !strings.Contains(strings.ToLower(body), "spelling") {
		t.Error("empty state should suggest trying different words or checking spelling")
	}
}

// Search result links do not append raw internal type names to their
// accessible name. The visible card meta paragraph already shows the kind
// label ("Note", "Block") so a visually hidden suffix was redundant and
// leaked an internal identifier into screen-reader output (backlog 0761).
func TestSearchResultLinksDoNotAppendTypeSuffix(t *testing.T) {
	t.Parallel()
	_, h := newApp(t)
	wantStatus(t, postJSON(t, h, http.MethodPost, "/api/note", map[string]any{"title": "Call the plumber", "body": "About the kitchen tap."}), http.StatusCreated)

	page := get(t, h, "/search?q=plumber")
	wantStatus(t, page, http.StatusOK)
	body := page.Body.String()

	// The link should contain only the record title with the search word marked.
	if !strings.Contains(body, `Call the <mark class="sw-search__hit">plumber</mark>`) {
		t.Error("result link should contain the record title, the word searched for marked")
	}
	// The heading link must not include a visually hidden type suffix.
	if strings.Contains(body, `<span class="sw-visually-hidden"> — Note`) ||
		strings.Contains(body, `<span class="sw-visually-hidden"> — note`) {
		t.Error("result links must not append raw internal type names like '— Note' to the accessible name")
	}
	// The visible card meta paragraph still shows the kind label.
	if !strings.Contains(body, `<p class="sw-card__meta">Note</p>`) &&
		!strings.Contains(body, `<p class="sw-card__meta">Note,`) {
		t.Error("the card meta should display the visible kind label")
	}
}

// Search results page has proper heading structure: h1 for the page title,
// h2 for the "Results" section before the list of hits.
func TestSearchResultsPageHeadingStructure(t *testing.T) {
	t.Parallel()
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
