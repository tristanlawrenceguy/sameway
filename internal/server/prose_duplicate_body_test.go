package server_test

import (
	"net/http"
	"strings"
	"testing"
)

// TestNoteDetailPageBodyDataProp verifies acceptance items 1 and 2 of task 0424.
// The rendered note detail page must have exactly one element with data-prop="body"
// so that swProseField creates only one editor wrapper. This test checks the server-
// side rendering is correct; the client-side fix (hiding the source textarea) ensures
// only one of the two created editors is visible at startup.
func TestNoteDetailPageBodyDataProp(t *testing.T) {
	a, h := newApp(t)

	rec, err := a.Store.Create("note", map[string]any{
		"title": "Duplicate body test",
		"body":  "Some body content for testing.",
	})
	if err != nil {
		t.Fatal(err)
	}

	r := get(t, h, "/t/note/"+rec.ID)
	wantStatus(t, r, http.StatusOK)

	body := r.Body.String()

	// The page must contain data-prop="body" with the body content in both
	// the source attribute and the rendered HTML.
	if !strings.Contains(body, `data-prop="body"`) {
		t.Error("note detail page should have data-prop=\"body\" for inline editing\n" +
			"body starts with: " + truncate(body))
	}

	if !strings.Contains(body, "Some body content for testing.") {
		t.Error("note detail page should display the body content in the rendered HTML")
	}

	if !strings.Contains(body, `data-source="Some body content for testing."`) {
		t.Error("note detail page should have data-source=\"...\" on the body element\n" +
			"so swProseField can populate the Markdown source textarea")
	}

	doc := parse(t, r)
	bodies := doc.WithAttr("data-prop", "body")
	if len(bodies) != 1 {
		t.Errorf("note detail page should have exactly one element with data-prop=\"body\", got %d\n"+
			"this causes swProseField to create %d editor wrappers (one per data-prop element)",
			len(bodies), len(bodies))
	}
}
