package server_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/look"
)

// Search result headings show only the record title with no appended type
// name, and their accessible names match the visible text exactly.
func TestSearchResultHeadingHasNoTypeSuffix(t *testing.T) {
	t.Parallel()
	_, h := newApp(t)
	wantStatus(t, postJSON(t, h, http.MethodPost, "/api/note", map[string]any{"title": "Call the plumber"}), http.StatusCreated)

	pageRes := get(t, h, "/search?q=plumber")
	wantStatus(t, pageRes, http.StatusOK)
	page := pageRes.Body.String()

	// The visible title must be present with marks.
	if !strings.Contains(page, `Call the <mark class="sw-search__hit">plumber</mark>`) {
		t.Error("heading should contain the record title with marked search words")
	}

	// No type name appended anywhere in the card: no "— Note", no bare "Note"
	// inside the h3 link text or a visually hidden span within it.
	if strings.Contains(page, " — Note") || strings.Contains(page, " — note") {
		t.Errorf("heading must not append a type suffix like ' — Note'; found in:\n%s", truncate(page))
	}

	// The card's h3 link text and its accessible name should match: no
	// sw-visually-hidden span inside the <a> that adds extra words.
	if strings.Contains(page, `<span class="sw-visually-hidden">`) {
		t.Errorf("search result card must not contain a visually hidden span; found in:\n%s", truncate(page))
	}

	// The link's accessible name (from look) should equal its visible text.
	o, err := look.Page(page)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range o.Controls {
		if c.Kind == "link" && strings.Contains(c.Name, "plumber") {
			visible := extractLinkText(page, c.Href)
			if visible != "" && c.Name != visible {
				t.Errorf("link accessible name %q does not match visible text %q", c.Name, visible)
			}
		}
	}

	if len(o.Problems) > 0 {
		t.Errorf("search page should read cleanly, got problems: %v", o.Problems)
	}
}

// extractLinkText returns the plain text between <a ...> and </a> for a given href.
func extractLinkText(html, href string) string {
	linkStart := strings.Index(html, `href="`+href+`"`)
	if linkStart == -1 {
		return ""
	}
	tagOpen := strings.Index(html[linkStart:], ">")
	if tagOpen == -1 {
		return ""
	}
	contentStart := linkStart + tagOpen + 1
	tagClose := strings.Index(html[contentStart:], "</a>")
	if tagClose == -1 {
		return ""
	}
	raw := html[contentStart : contentStart+tagClose]
	// Strip HTML tags to get visible text.
	for strings.Contains(raw, "<") {
		s := strings.Index(raw, "<")
		e := strings.Index(raw[s:], ">")
		if e == -1 {
			break
		}
		raw = raw[:s] + raw[s+e+1:]
	}
	return strings.TrimSpace(raw)
}
