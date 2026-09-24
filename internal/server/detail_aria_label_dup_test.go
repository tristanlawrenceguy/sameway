package server_test

// Tests for no duplicate accessible names from aria-labels on record detail pages
// (task 0186). Acceptance item 3: aria-labels do not duplicate visible labels
// already announced.

import (
	"net/http"
	"strings"
	"testing"

	"golang.org/x/net/html"

	"github.com/tristanlawrenceguy/sameway/internal/render/htmltest"
)

// TestDetailPageAriaLabelsDoNotDuplicateVisibleLabels checks that on record
// detail pages, no element has an aria-label that duplicates visible text
// already announced by its label or content. This covers acceptance item 3 for
// all types with a title field.
func TestDetailPageAriaLabelsDoNotDuplicateVisibleLabels(t *testing.T) {
	a, h := newApp(t)

	records := []struct{ typ, id string }{}

	note, err := a.Store.Create("note", map[string]any{"title": "Test note"})
	if err == nil {
		records = append(records, struct{ typ, id string }{"note", note.ID})
	}
	task, err := a.Store.Create("task", map[string]any{"title": "Test task"})
	if err == nil {
		records = append(records, struct{ typ, id string }{"task", task.ID})
	}
	action, err := a.Store.Create("action", map[string]any{"title": "Test action"})
	if err == nil {
		records = append(records, struct{ typ, id string }{"action", action.ID})
	}

	for _, r := range records {
		rec := get(t, h, "/t/"+r.typ+"/"+r.id)
		wantStatus(t, rec, http.StatusOK)

		doc, err := htmltest.Parse(rec.Body.String())
		if err != nil {
			t.Fatalf("%s/%s: parse error: %v", r.typ, r.id, err)
		}

		doc.Walk(func(el *html.Node) {
			checkAriaLabelRedundancy(t, el, "/t/"+r.typ+"/"+r.id)
		})
	}
}

// checkAriaLabelRedundancy checks if a parsed element has an aria-label that
// duplicates visible text already announced by the element's content.
func checkAriaLabelRedundancy(t *testing.T, n *html.Node, path string) {
	t.Helper()

	ariaLabel, ok := htmltest.Attr(n, "aria-label")
	if !ok || ariaLabel == "" {
		return // no aria-label on this element — fine
	}

	visibleText := strings.TrimSpace(htmltest.Text(n))
	if visibleText == "" {
		return // icon-only or empty element — fine for now
	}

	// Check if any word in the visible text also appears in the aria-label.
	for _, w := range words(visibleText) {
		lowerW := strings.ToLower(w)
		if strings.Contains(strings.ToLower(ariaLabel), lowerW) {
			t.Errorf("%s: element has redundant aria-label %q that duplicates visible text %q — screen readers announce it twice (Acceptance 3)", path, ariaLabel, visibleText)
		}
	}
}
