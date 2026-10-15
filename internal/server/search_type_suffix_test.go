package server_test

// Search result heading links must not append raw internal type names like
// "— Note" or "— Block" to the accessible name. The visible meta paragraph
// already shows the kind label, so a visually hidden suffix is redundant and
// leaks an internal identifier into screen-reader output. Acceptance 1 of
// task 0286 (backlog 0761).

import (
	"net/http"
	"strings"
	"testing"
)

// TestSearchResultHeadingDoesNotContainRawTypeSuffix verifies that search
// result heading links do not include a visually hidden "— Type" suffix.
// The type label already appears visibly in the card meta paragraph, so it
// should not also be repeated inside the accessible name of the heading link.
func TestSearchResultHeadingDoesNotContainRawTypeSuffix(t *testing.T) {
	t.Parallel()
	_, h := newApp(t)
	wantStatus(t, postJSON(t, h, http.MethodPost, "/api/note", map[string]any{
		"title": "Call the plumber",
		"body":  "About the kitchen tap.",
	}), http.StatusCreated)

	page := get(t, h, "/search?q=plumber")
	wantStatus(t, page, http.StatusOK)
	body := page.Body.String()

	// The heading link should be just <a href="...">Call the plumber</a> —
	// no visually hidden span with a type suffix inside it.
	if strings.Contains(body, `<span class="sw-visually-hidden"> — Note`) ||
		strings.Contains(body, `<span class="sw-visually-hidden"> — note`) {
		t.Errorf("search heading link must not contain a raw type suffix like '— Note'; found in:\n%s", truncate(body))
	}

	// The visible meta paragraph should still show the kind label.
	if !strings.Contains(body, `<p class="sw-card__meta">Note</p>`) &&
		!strings.Contains(body, `<p class="sw-card__meta">Note,`) {
		t.Errorf("the card meta paragraph should still display the visible kind label; found in:\n%s", truncate(body))
	}

	// The heading link's text content must be present and unmodified.
	if !strings.Contains(body, `Call the <mark class="sw-search__hit">plumber</mark>`) {
		t.Error("result link should contain the record title with the search word marked")
	}
}

// TestSearchResultHeadingNoTypeSuffixForBlock verifies that block results
// also do not append a raw type suffix to their heading accessible name.
func TestSearchResultHeadingNoTypeSuffixForBlock(t *testing.T) {
	t.Parallel()
	_, h := newApp(t)
	wantStatus(t, postJSON(t, h, http.MethodPost, "/api/block", map[string]any{
		"component": "card",
		"props": map[string]any{
			"title": "Important notice",
			"body":  "Read this carefully.",
		},
	}), http.StatusCreated)

	page := get(t, h, "/search?q=notice")
	wantStatus(t, page, http.StatusOK)
	body := page.Body.String()

	if strings.Contains(body, `<span class="sw-visually-hidden"> — Block`) ||
		strings.Contains(body, `<span class="sw-visually-hidden"> — block`) {
		t.Errorf("block search result heading must not contain a raw type suffix; found in:\n%s", truncate(body))
	}
}
