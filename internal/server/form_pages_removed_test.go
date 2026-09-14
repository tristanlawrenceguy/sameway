package server_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/render/htmltest"
)

// TestNewRouteReturns404 asserts that the old form-based create page is gone:
// GET /t/{type}/new must return 404 (no handler registered).
func TestNewRouteReturns404(t *testing.T) {
	_, h := newApp(t)
	rec := get(t, h, "/t/note/new")
	wantStatus(t, rec, http.StatusNotFound)
}

// TestEditRouteReturns404 asserts that the old form-based edit page is gone:
// GET /t/{type}/{id}/edit must return 404 (no handler registered).
func TestEditRouteReturns404(t *testing.T) {
	a, h := newApp(t)
	rec, _ := a.Store.Create("note", map[string]any{"title": "Seed"})
	path := "/t/note/" + rec.ID + "/edit"
	resp := get(t, h, path)
	wantStatus(t, resp, http.StatusNotFound)
}

// TestDescribeRoutesNoHTMLNew asserts that the routes map in Describe() has no
// "html_new" key — the JSON API remains the only surface for creating records.
func TestDescribeRoutesNoHTMLNew(t *testing.T) {
	a, _ := newApp(t)
	d := a.Describe()
	if _, ok := d.Routes["html_new"]; ok {
		t.Errorf("routes map should not contain html_new key")
	}
}

// TestListingPageHasNoNewLink checks that listing pages (/t/note, etc.) contain
// no link to "/t/{type}/new" and no text saying "New {Type}".
func TestListingPageHasNoNewLink(t *testing.T) {
	a, h := newApp(t)
	_ = a
	// Create a record so the list is non-empty.
	a.Store.Create("note", map[string]any{"title": "Exists"})

	doc := parse(t, get(t, h, "/t/note"))

	for _, link := range doc.WithAttr("href", "/t/note/new") {
		t.Errorf("listing should not link to /t/note/new: found %s", htmltest.Text(link))
	}

	rec := get(t, h, "/t/note")
	body := rec.Body.String()
	if strings.Contains(body, "New note") {
		t.Errorf("listing page should not say \"New note\"")
	}
}

// TestDetailPageHasNoEditLink checks that detail pages (/t/{type}/{id}) contain
// no link to "/t/{type}/{id}/edit" and no text saying "Edit {Type}".
func TestDetailPageHasNoEditLink(t *testing.T) {
	a, h := newApp(t)
	rec, _ := a.Store.Create("note", map[string]any{"title": "Seed record"})

	doc := parse(t, get(t, h, "/t/note/"+rec.ID))

	for _, link := range doc.WithAttr("href", "/t/note/"+rec.ID+"/edit") {
		t.Errorf("detail page should not link to /edit: found %s", htmltest.Text(link))
	}

	resp := get(t, h, "/t/note/"+rec.ID)
	body := resp.Body.String()
	if strings.Contains(body, "Edit note") {
		t.Errorf("detail page should not say \"Edit note\"")
	}
}

// TestConfirmDeleteStillWorks asserts that the confirm-delete page still returns
// 200 after removing form-based edit surfaces.
func TestConfirmDeleteStillWorks(t *testing.T) {
	a, h := newApp(t)
	rec, _ := a.Store.Create("note", map[string]any{"title": "To delete"})

	resp := get(t, h, "/t/note/"+rec.ID+"/confirm-delete")
	wantStatus(t, resp, http.StatusOK)

	doc := parse(t, resp)
	if n := len(doc.Elements("h1")); n != 1 {
		t.Errorf("confirm-delete should have one h1, got %d", n)
	}
	// The page contains a danger alert and a confirm button.
	alerts := doc.WithAttr("data-component", "alert")
	if len(alerts) == 0 {
		t.Error("confirm-delete should render a danger alert")
	}
	buttons := doc.WithAttr("data-component", "button")
	if len(buttons) == 0 {
		t.Error("confirm-delete should have a confirm button")
	}
}

// TestDeleteStillWorks asserts that deleting a record via POST to the delete
// endpoint still works after removing form-based edit surfaces.
func TestDeleteStillWorks(t *testing.T) {
	a, h := newApp(t)
	rec, _ := a.Store.Create("note", map[string]any{"title": "Gone"})

	detailPath := "/t/note/" + rec.ID

	// POST to the delete endpoint.
	del := postForm(t, h, detailPath+"/delete", nil)
	wantStatus(t, del, http.StatusSeeOther)

	// Re-fetch should return 404.
	wantStatus(t, get(t, h, detailPath), http.StatusNotFound)

	// List page should no longer show this record.
	listDoc := parse(t, get(t, h, "/t/note"))
	for range listDoc.WithAttr("href", detailPath) {
		t.Errorf("deleted note %q should not appear on the listing page", rec.ID)
	}
}
