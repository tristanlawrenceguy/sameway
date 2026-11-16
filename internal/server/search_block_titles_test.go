package server_test

import (
	"net/http"
	"regexp"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/look"
)

// stripHTML removes HTML tags from s for plain-text assertions.
func stripHTML(s string) string {
	return regexp.MustCompile(`<[^>]*>`).ReplaceAllString(s, " ")
}

// stripAndTrim is like stripHTML but collapses runs of whitespace and trims.
func stripAndTrim(s string) string {
	return strings.TrimSpace(regexp.MustCompile(`\s+`).ReplaceAllString(stripHTML(s), " "))
}

// Search results for canvas blocks show the actual content text as their
// heading, not internal component names like "card", "collection" or
// "heading". This covers acceptance item 3: searching for "community cleanup"
// and confirming real text appears.
func TestSearchResultsForBlocksShowContentText(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)

	// Create a heading block with searchable content.
	headingID := storedBlock(t, a, "heading", map[string]any{
		"title": "Community cleanup on Saturday",
	})

	// A card block with text content matching the search term.
	cardID := storedBlock(t, a, "card", map[string]any{
		"text": "Pack book collection for the house move tomorrow",
	})

	_ = headingID
	_ = cardID

	page := get(t, h, "/search?q=community+cleanup")
	wantStatus(t, page, http.StatusOK)
	body := page.Body.String()

	if strings.Contains(body, ">heading — Block</a>") ||
		strings.Contains(body, `aria-label="heading — Block"`) {
		t.Error("search result shows internal block type 'heading' instead of content text")
	}

	// The search result should contain the actual heading title.
	// Check that the heading text appears as content, accounting for <mark> tags
	// added by marked() around search words. Strip HTML and trim whitespace.
	if !strings.Contains(stripAndTrim(body), "Community cleanup on Saturday") {
		t.Error("search results do not show the heading's actual content: expected 'Community cleanup on Saturday' in page")
	}

	o, _ := look.Page(body)
	if len(o.Problems) != 0 {
		t.Errorf("the search page should read cleanly, got %v", o.Problems)
	}
}

// A card block found by searching for its text shows that text as the result
// heading, not "card — Block".
func TestSearchResultsForCardBlocksShowCardContent(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)
	cardID := storedBlock(t, a, "card", map[string]any{
		"text": "Pack book collection for the house move tomorrow",
	})
	_ = cardID

	page := get(t, h, "/search?q=house+move")
	wantStatus(t, page, http.StatusOK)
	body := page.Body.String()

	if strings.Contains(body, ">card — Block</a>") ||
		strings.Contains(body, `aria-label="card — Block"`) {
		t.Error("search result shows internal block type 'card' instead of content text")
	}

	if !strings.Contains(body, "Pack book collection for the house move") &&
		!strings.Contains(body, "house move") {
		t.Error("search results do not show card content: expected 'Pack book collection' or related text in page")
	}

	o, _ := look.Page(body)
	if len(o.Problems) != 0 {
		t.Errorf("the search page should read cleanly, got %v", o.Problems)
	}
}

// A collection block with a label shows the label as the result heading.
func TestSearchResultsForCollectionBlocksShowLabel(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)
	storedBlock(t, a, "collection", map[string]any{
		"type":  "task",
		"label": "Up next items for the move",
	})

	page := get(t, h, "/search?q=up+next")
	wantStatus(t, page, http.StatusOK)
	body := page.Body.String()

	if strings.Contains(body, ">collection — Block</a>") ||
		strings.Contains(body, `aria-label="collection — Block"`) {
		t.Error("search result shows internal block type 'collection' instead of content text")
	}

	if !strings.Contains(body, "Up next items for the move") &&
		!strings.Contains(body, "up next") {
		t.Error("search results do not show collection label: expected 'Up next items for the move' in page")
	}

	o, _ := look.Page(body)
	if len(o.Problems) != 0 {
		t.Errorf("the search page should read cleanly, got %v", o.Problems)
	}
}
