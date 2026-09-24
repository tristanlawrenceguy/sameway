package server_test

import (
	"golang.org/x/net/html"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/render/htmltest"
)

// headingWords returns the number of space-delimited words in a node's visible
// text only, ignoring any .sw-visually-hidden descendants.
func headingWords(n *html.Node) int {
	return len(strings.Fields(htmltest.VisibleText(n)))
}

// TestSearchCardH2TrimsLongTitle verifies that a search result card's h2
// heading is trimmed to at most six words when the note title exceeds them.
// (Acceptance items 1, 3.)
func TestSearchCardH2TrimsLongTitle(t *testing.T) {
	a, h := newApp(t)

	// Create a note with a long title — more than six words.
	wantFull := "This is a very long title that definitely exceeds six"
	rec, err := a.Store.Create("note", map[string]any{
		"title": wantFull,
	})
	if err != nil {
		t.Fatal(err)
	}

	// Search for the note so it appears in results.
	page := get(t, h, "/search?q=long+title")
	wantStatus(t, page, 200)
	body := page.Body.String()

	// The card should be present at all (sanity).
	if !strings.Contains(body, "sw-card__title") {
		t.Fatalf("expected search result card; body: %.400s", truncate(body))
	}

	doc := parse(t, page)
	h2s := doc.Elements("h2")
	if len(h2s) == 0 {
		t.Fatal("expected at least one h2 in search results")
	}

	// Find all h2 nodes with class sw-card__title and check their text content.
	for _, node := range doc.Elements("h2") {
		class, _ := htmltest.Attr(node, "class")
		if class != "sw-card__title" {
			continue
		}
		text := strings.TrimSpace(htmltest.VisibleText(node))
		if headingWords(node) > 6 {
			t.Errorf("search card h2 should be trimmed to at most 6 words, got %d: %q", headingWords(node), text)
		}
	}

	// The full title must still appear somewhere on the page (link href or body).
	if !strings.Contains(body, "/t/note/"+rec.ID) {
		t.Errorf("search result should link to the note at /t/note/%s\n%s", rec.ID, truncate(body))
	}

	// The full untrimmed title should not be present as visible heading text.
	for _, node := range doc.Elements("h2") {
		class, _ := htmltest.Attr(node, "class")
		if class != "sw-card__title" {
			continue
		}
		text := strings.TrimSpace(htmltest.Text(node))
		if text == wantFull {
			t.Errorf("search card h2 should NOT contain the full untrimmed %d-word title: %q", len(strings.Fields(wantFull)), wantFull)
		}
	}

}

// TestSearchCardH2LeavesShortTitleUntouched verifies that a short title within
// the six-word limit is not modified in the search result card h2. (Acceptance 1.)
func TestSearchCardH2LeavesShortTitleUntouched(t *testing.T) {
	a, h := newApp(t)

	wantTitle := "Three words only"
	_, err := a.Store.Create("note", map[string]any{
		"title": wantTitle,
	})
	if err != nil {
		t.Fatal(err)
	}

	page := get(t, h, "/search?q=words")
	wantStatus(t, page, 200)

	doc := parse(t, page)

	for _, node := range doc.Elements("h2") {
		class, _ := htmltest.Attr(node, "class")
		if class != "sw-card__title" {
			continue
		}
		text := strings.TrimSpace(htmltest.Text(node))
		// The heading text is trimmed_title + " — note" (from the visually-hidden span).
		wantFullText := wantTitle + " \u2014 note"
		if text != wantFullText {
			t.Errorf("short title should be unchanged in search card h2: got %q, want %q", text, wantFullText)
		}

		// The visible link text (before the visually-hidden span) must match the original.
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			if c.Type == html.ElementNode && c.Data == "a" {
				linkText := strings.TrimSpace(htmltest.VisibleText(c))
				if linkText != wantTitle {
					t.Errorf("search card h2 link text should be %q, got %q", wantTitle, linkText)
				}
			}
		}
	}
}

// TestSearchCardVisuallyHiddenTypePreserved verifies that the visually-hidden
// span (e.g. "— note") is preserved after trimming the visible heading text.
// (Acceptance 3.)
func TestSearchCardVisuallyHiddenTypePreserved(t *testing.T) {
	a, h := newApp(t)

	wantFull := "This is a very long title that definitely exceeds six"
	_, err := a.Store.Create("note", map[string]any{
		"title": wantFull,
	})
	if err != nil {
		t.Fatal(err)
	}

	page := get(t, h, "/search?q=long+title")
	wantStatus(t, page, 200)
	body := page.Body.String()

	doc := parse(t, page)

	for _, node := range doc.Elements("h2") {
		class, _ := htmltest.Attr(node, "class")
		if class != "sw-card__title" {
			continue
		}
		// The visible heading text should be trimmed (at most 6 words).
		if headingWords(node) > 6 {
			t.Errorf("search card h2 should be trimmed to at most 6 words, got %d: %q", headingWords(node), strings.TrimSpace(htmltest.VisibleText(node)))
		}

		// The visually-hidden span must still contain the content type name.
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			if c.Type == html.ElementNode && c.Data == "span" {
				class, ok := htmltest.Attr(c, "class")
				if ok && strings.Contains(class, "sw-visually-hidden") {
					spinnerText := strings.TrimSpace(htmltest.Text(c))
					if !strings.Contains(spinnerText, "— note") {
						t.Errorf("visually-hidden span should contain '— note', got %q", spinnerText)
					}
				}
			}
		}
	}

	// Verify the link to the note exists.
	if !strings.Contains(body, "/t/note/") {
		t.Error("search result should link to the note")
	}

}
