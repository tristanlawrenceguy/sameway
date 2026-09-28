package server_test

// Tests for no duplicate accessible name on mark checkboxes in list pages
// (task 0178). On /t/{type} listings, each expand checkbox must have exactly
// one source of its accessible name — the aria-label. No adjacent text node
// (visible or visually-hidden) may repeat it.

import (
	"net/http"
	"strings"
	"testing"

	"golang.org/x/net/html"

	"github.com/tristanlawrenceguy/sameway/internal/render/htmltest"
)

// TestAMarkCheckboxOnTaskListHasNoDuplicateLabel checks that on the /t/task
// listing page, each mark checkbox's aria-label is not duplicated by an
// adjacent text node. The fix suppresses the visually-hidden span when both
// quiet and ariaLabel are set (Acceptance 1).
func TestAMarkCheckboxOnTaskListHasNoDuplicateLabel(t *testing.T) {
	_, h := newApp(t)

	// Create a task so it appears on the /t/task listing.
	var task struct{ ID string }
	decode(t, postJSON(t, h, http.MethodPost, "/api/task",
		map[string]any{"title": "Order compost"}), &task)

	rec := get(t, h, "/t/task")
	wantStatus(t, rec, http.StatusOK)

	doc, err := htmltest.Parse(rec.Body.String())
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	var found bool
	for _, input := range doc.Elements("input") {
		typ, ok := htmltest.Attr(input, "type")
		if !ok || typ != "checkbox" {
			continue
		}
		found = true
		checkCheckboxRedundancy(t, input, "/t/task")
	}
	if !found {
		t.Fatal("expected at least one checkbox on the /t/task listing page")
	}
}

// checkCheckboxRedundancy walks the parsed <input type="checkbox"> node and its
// following text nodes / spans to ensure no duplicate accessible name exists.
func checkCheckboxRedundancy(t *testing.T, n *html.Node, path string) {
	t.Helper()

	ariaLabel, ok := htmltest.Attr(n, "aria-label")
	if !ok || ariaLabel == "" {
		return // no aria-label on this checkbox — fine for non-list pages
	}

	// Look at the next sibling node(s) after the input. If they contain text
	// that duplicates words from the aria-label (and are not visually-hidden),
	// that is a duplicate accessible name.
	next := n.NextSibling
	for next != nil {
		if next.Type == html.TextNode {
			text := strings.TrimSpace(next.Data)
			if text == "" {
				next = next.NextSibling
				continue
			}
			// Any non-empty visible text adjacent to the checkbox is a duplicate.
			t.Errorf("%s: checkbox with aria-label=%q has adjacent visible text %q — screen readers announce it twice (Acceptance 1)", path, ariaLabel, text)
		}
		if next.Type == html.ElementNode && next.Data == "span" {
			// Check if this span is visually-hidden. If so and it contains words
			// from the aria-label, that's a duplicate too (screen readers still read it).
			classVal, _ := htmltest.Attr(next, "class")
			if strings.Contains(classVal, "visually-hidden") {
				childText := textContent(next)
				if childText != "" {
					t.Errorf("%s: checkbox with aria-label=%q has adjacent visually-hidden span %q — screen readers announce it twice (Acceptance 1)", path, ariaLabel, strings.TrimSpace(childText))
				}
			}
		}
		next = next.NextSibling
	}
}

// textContent returns the concatenated text of all descendant text nodes.
func textContent(n *html.Node) string {
	var b strings.Builder
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.TextNode {
			b.WriteString(c.Data)
		} else if c.Type == html.ElementNode {
			b.WriteString(textContent(c))
		}
	}
	return b.String()
}
