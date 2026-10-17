package server_test

import (
	"net/http"
	"regexp"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/look"
)

// h3TitleRe matches an <h3 class="sw-card__title">...</h3> block and captures
// everything between the tags. Used by tests to inspect heading link contents.
var h3TitleRe = regexp.MustCompile(`(?s)<h3[^>]*class="sw-card__title"[^>]*>(.*?)</h3>`)

// extractH3Title returns the raw HTML inside the first search result card's h3.
func extractH3Title(body string) string {
	m := h3TitleRe.FindStringSubmatch(body)
	if len(m) < 2 {
		return ""
	}
	return strings.TrimSpace(m[1])
}

// Search result card heading links show only the record title, with no raw
// type name suffix like "— Habit" or "— Note" appended to the link text. The
// type name appears as a separate paragraph below the card (sw-card__meta),
// not inside the heading link's accessible name. Covers acceptance item 1.
func TestSearchResultHeadingHasNoRawTypeNameSuffix(t *testing.T) {
	t.Parallel()
	_, h := newApp(t)
	wantStatus(t, postJSON(t, h, http.MethodPost, "/api/note", map[string]any{
		"title": "Call the plumber", "body": "About the kitchen tap.",
	}), http.StatusCreated)

	page := get(t, h, "/search?q=plumber")
	wantStatus(t, page, http.StatusOK)
	body := page.Body.String()

	// The visible link text should be just the title with marked words.
	if !strings.Contains(body, `Call the <mark class="sw-search__hit">plumber</mark>`) {
		t.Error("result link should contain the record title with the search word marked")
	}

	// No raw type name suffix should appear in any heading link's text or
	// visually-hidden span. After the fix, "— Note" must not be inside an h3
	// that wraps a card title link.
	if strings.Contains(body, `">— Note`) || strings.Contains(body, `">— note`) {
		t.Error("heading link text should not contain a raw type name suffix like — Note")
	}

	// The type name must appear as a separate <p> below the card title, not
	// inside the heading. The card meta paragraph is how the type is shown.
	if !strings.Contains(body, `<p class="sw-card__meta">Note</p>`) {
		t.Error("type name should appear in a separate sw-card__meta paragraph below each result")
	}

	// The visually-hidden span must not be inside the heading link text.
	h3Title := extractH3Title(body)
	if h3Title != "" && strings.Contains(h3Title, `sw-visually-hidden`) {
		t.Errorf("heading link should not contain sw-visually-hidden; got: %s", h3Title)
	}

	// The heading link must use aria-label for its accessible name.
	if !strings.Contains(body, `aria-label=`) || !strings.Contains(body, "Call the") {
		t.Error("result card links should carry an aria-label for their full accessible name")
	}
}

// Twin search results are told apart by disambiguation context, not raw type
// identifiers. When two records share a title, each heading link carries an
// aria-label with just the title plus "(added …)" — no "— Task" or similar.
// The visible text is only the title; the separate <p class="sw-card__meta">
// shows "Task". Covers acceptance items 2 and 3.
func TestSearchTwinsHaveCleanHeadingsWithAriaLabel(t *testing.T) {
	t.Parallel()
	_, h := newAppAt(t, tonight)
	_, _ = seedTwins(t, h)

	results := get(t, h, "/search?q=plumber")
	wantStatus(t, results, http.StatusOK)
	body := results.Body.String()

	// The accessible names from look should not contain "— Task".
	linkNames := names(t, body, "link")
	for _, ln := range linkNames {
		if strings.Contains(ln, "— Task") || strings.Contains(ln, "— task") {
			t.Errorf("link name must not include raw type suffix; got %q", ln)
		}
	}

	// The accessible names should be just title + context.
	hasAny := false
	for _, ln := range linkNames {
		if strings.HasPrefix(ln, "Call plumber (") && !strings.Contains(ln, "— Task") {
			hasAny = true
		}
	}
	if !hasAny {
		t.Errorf("twin search results should have accessible names like 'Call plumber (%s)'; got %q", linkNames[0], linkNames)
	}

	// The visible heading text must be clean.
	h3Text := h3Said(body)
	for _, txt := range h3Text {
		if strings.Contains(txt, "— Task") || strings.Contains(txt, "— Habit") {
			t.Errorf("heading link visible text should not contain raw type suffix; got %q", txt)
		}
	}

	// The page must read cleanly through look.
	o, err := look.Page(body)
	if err != nil {
		t.Fatalf("page parse failed: %v", err)
	}
	if len(o.Problems) != 0 {
		t.Errorf("search results with twins should have no a11y problems; got %v", o.Problems)
	}

	// The two twin links must still be told apart.
	if same := sameNames(t, body); len(same) > 0 {
		t.Errorf("twin search results share an accessible name: %v", same)
	}

	// The type paragraph is visible below each card.
	if !strings.Contains(body, `<p class="sw-card__meta">Task</p>`) {
		t.Error("type 'Task' should appear in a sw-card__meta paragraph below each search result")
	}
}
