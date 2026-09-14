package server_test

import (
	"net/http"
	"testing"
)

// TestNewRouteReturns404 asserts that GET /t/{type}/new no longer exists —
// it must return 404, not render a form.  Covers acceptance item 1.
func TestNewRouteReturns404(t *testing.T) {
	_, h := newApp(t)
	rec := get(t, h, "/t/note/new")
	wantStatus(t, rec, http.StatusNotFound)
}

// TestEditRouteReturns404 asserts that GET /t/{type}/{id}/edit no longer
// exists — it must return 404.  Covers acceptance item 1.
func TestEditRouteReturns404(t *testing.T) {
	a, h := newApp(t)
	rec, _ := a.Store.Create("note", map[string]any{"title": "Seed"})
	resp := get(t, h, "/t/note/"+rec.ID+"/edit")
	wantStatus(t, resp, http.StatusNotFound)
}

// TestListPageHasNoNewLink asserts that the listing page does not contain
// a link with href pattern /t/{type}/new.  Covers acceptance items 3 and 4.
func TestListPageHasNoNewLink(t *testing.T) {
	_, h := newApp(t)
	rec := get(t, h, "/t/note")
	wantStatus(t, rec, http.StatusOK)
	doc := parse(t, rec)
	if len(doc.WithAttr("href", "/t/note/new")) != 0 {
		t.Errorf("listing page should not contain a New note link with href /t/note/new")
	}
}

// TestDetailPageHasNoEditLink asserts that the detail page does not contain
// a link with an edit path.  Covers acceptance item 5.
func TestDetailPageHasNoEditLink(t *testing.T) {
	a, h := newApp(t)
	rec, _ := a.Store.Create("note", map[string]any{"title": "Seed"})
	resp := get(t, h, "/t/note/"+rec.ID)
	wantStatus(t, resp, http.StatusOK)
	doc := parse(t, resp)
	if len(doc.WithAttr("href", "/t/note/"+rec.ID+"/edit")) != 0 {
		t.Errorf("detail page should not contain an Edit link with href /t/note/{id}/edit")
	}
}

// TestDescribeHasNoHtmlNewRoute asserts that the machine-readable routes map
// under Describe() does not include html_new.  Covers acceptance item 3.
func TestDescribeHasNoHtmlNewRoute(t *testing.T) {
	a, _ := newApp(t)
	d := a.Describe()
	if _, ok := d.Routes["html_new"]; ok {
		t.Errorf("describe routes should not contain html_new")
	}
}
