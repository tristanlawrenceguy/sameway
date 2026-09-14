package server_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/render/htmltest"
)

// TestNewRouteReturns404 verifies that GET /t/{type}/new no longer serves a
// form page. After task 0061 the HTML create surface is gone; creation happens
// through the JSON API. This test ensures the route registration was removed
// and a 404 lands instead of a 200 with a form.
func TestNewRouteReturns404(t *testing.T) {
	_, h := newApp(t)

	rec := get(t, h, "/t/note/new")
	wantStatus(t, rec, http.StatusNotFound)

	doc := parse(t, rec)
	if len(doc.WithAttr("method", "post")) > 0 {
		t.Error("new page should be a 404, not serve an HTML form")
	}
}

// TestEditRouteReturns404 verifies that GET /t/{type}/{id}/edit no longer serves
// an edit form. After task 0061 the HTML create surface is gone; editing happens
// through the JSON API. This test ensures the route registration was removed.
func TestEditRouteReturns404(t *testing.T) {
	a, h := newApp(t)

	rec2, _ := a.Store.Create("note", map[string]any{"title": "Delete me"})

	// Request the edit page for an existing record — it should be 404 because
	// the route handler was removed. Using a real ID ensures the 404 comes from
	// a missing route, not from a missing database row.
	rec := get(t, h, "/t/note/"+rec2.ID+"/edit")
	wantStatus(t, rec, http.StatusNotFound)

	doc := parse(t, rec)
	if len(doc.WithAttr("method", "post")) > 0 {
		t.Error("edit page should be a 404, not serve an HTML form")
	}
}

// TestListPageShowsCreateViaAPI verifies the listing page links to the JSON API
// for creation instead of a removed /new page. The label must say "Create via
// API" and the href must point at /api/{type}.
func TestListPageShowsCreateViaAPI(t *testing.T) {
	_, h := newApp(t)

	rec := get(t, h, "/t/note")
	wantStatus(t, rec, http.StatusOK)
	doc := parse(t, rec)

	hrefs := doc.WithAttr("href", "/api/note")
	if len(hrefs) == 0 {
		t.Fatal("list page should have a link with href=/api/note for API creation")
	}

	texts := make([]string, 0, len(hrefs))
	for _, a := range hrefs {
		texts = append(texts, htmltest.Text(a))
	}
	found := false
	for _, txt := range texts {
		if strings.Contains(txt, "Create via API") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("list page link to /api/note should have label containing 'Create via API', got labels: %v", texts)
	}

	// The old /t/note/new link must be gone.
	oldHrefs := doc.WithAttr("href", "/t/note/new")
	if len(oldHrefs) > 0 {
		t.Error("list page should not link to the removed /new route")
	}
}

// TestDetailPageNoEditLink verifies that the detail page has an Edit link only
// after task 0061. After removal there is no edit form, so the link must be
// absent and a Delete (confirm-delete) link must still exist.
func TestDetailPageNoEditLink(t *testing.T) {
	a, h := newApp(t)

	rec, _ := a.Store.Create("note", map[string]any{"title": "No edit here"})

	page := parse(t, get(t, h, "/t/note/"+rec.ID))

	editLinks := page.WithAttr("href", "/t/note/"+rec.ID+"/edit")
	if len(editLinks) > 0 {
		t.Error("detail page should not have an Edit link; editing is via API only")
	}

	deleteLink := page.WithAttr("href", "/t/note/"+rec.ID+"/confirm-delete")
	if len(deleteLink) == 0 {
		t.Error("detail page should still show a Delete/confirm-delete link")
	}
}

// TestDescribeNoHtmlNew verifies the Routes map in /api/describe has no
// "html_new" entry. That route is gone; creation happens via API only.
func TestDescribeNoHtmlNew(t *testing.T) {
	_, h := newApp(t)

	rec := get(t, h, "/api/describe")
	wantStatus(t, rec, http.StatusOK)

	var d struct {
		Routes map[string]string `json:"routes"`
	}
	decode(t, rec, &d)

	if _, ok := d.Routes["html_new"]; ok {
		t.Errorf("Routes map should not contain 'html_new'; creation is API-only")
	}
}
