package server_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/render/htmltest"
)

// TestNewPageReturns404 verifies that the form-based new-record page has been
// removed and no longer serves HTML. Acceptance item 1: GET /t/{type}/new → 404.
func TestNewPageReturns404(t *testing.T) {
	_, h := newApp(t)

	rec := get(t, h, "/t/note/new")
	wantStatus(t, rec, http.StatusNotFound)

	rec = get(t, h, "/t/activity/new")
	wantStatus(t, rec, http.StatusNotFound)
}

// TestNewPageIsNotInDescribe verifies that app.go Describe() no longer lists
// an html_new route. Acceptance item 1: Routes map has no "html_new" entry.
func TestNewPageIsNotInDescribe(t *testing.T) {
	_, h := newApp(t)

	rec := get(t, h, "/api/describe")
	wantStatus(t, rec, http.StatusOK)
	var d struct {
		Routes map[string]string
	}
	decode(t, rec, &d)
	if _, ok := d.Routes["html_new"]; ok {
		t.Errorf("Routes should not contain html_new — the new page was removed")
	}
}

// TestEditPageReturns404 verifies that the form-based edit page has been removed.
// Acceptance item 1: GET /t/{type}/{id}/edit → 404.
func TestEditPageReturns404(t *testing.T) {
	a, h := newApp(t)

	// Need a record to test the edit route against it.
	rec, _ := a.Store.Create("note", map[string]any{"title": "Seed"})

	rec2 := get(t, h, "/t/note/"+rec.ID+"/edit")
	wantStatus(t, rec2, http.StatusNotFound)

	// Non-existent type should also return 404.
	rec3 := get(t, h, "/t/activity/nonexistent/edit")
	wantStatus(t, rec3, http.StatusNotFound)
}

// TestListingPageShowsCreateViaAPI verifies the listing page links to the API
// instead of a dead new-page form link. Acceptance item 3: "Create via API" →
// /api/{type} on listing pages.
func TestListingPageShowsCreateViaAPI(t *testing.T) {
	_, h := newApp(t)

	rec := get(t, h, "/t/note")
	wantStatus(t, rec, http.StatusOK)
	body := rec.Body.String()

	if !strings.Contains(body, "Create via API") {
		t.Errorf("listing page should show \"Create via API\" link\nbody: %s", truncate(body))
	}

	doc, err := htmltest.Parse(body)
	if err != nil {
		t.Fatal(err)
	}
	link := doc.WithAttr("href", "/api/note")
	if len(link) == 0 {
		t.Errorf("listing page should have a link to /api/note")
	}

	// Also verify the old dead link is gone.
	oldLink := doc.WithAttr("href", "/t/note/new")
	if len(oldLink) > 0 {
		t.Errorf("listing page should not still link to /t/note/new (dead link)")
	}
}

// TestDetailPageHasNoEditLink verifies the detail page removed its edit link.
// Acceptance item 4: detail pages have no Edit link — only Delete.
func TestDetailPageHasNoEditLink(t *testing.T) {
	a, h := newApp(t)

	rec, _ := a.Store.Create("note", map[string]any{"title": "Seed"})

	doc := parse(t, get(t, h, "/t/note/"+rec.ID))

	// The body string should not contain an /edit href.
	body := get(t, h, "/t/note/"+rec.ID).Body.String()
	if strings.Contains(body, `/t/note/`+rec.ID+`/edit`) {
		t.Errorf("detail page should not link to /edit")
	}

	// Confirm the delete link to confirm-delete is still present.
	deleteLinks := doc.WithAttr("href", "/t/note/"+rec.ID+"/confirm-delete")
	if len(deleteLinks) == 0 {
		t.Errorf("detail page should have a Delete → confirm-delete link")
	}
}

// TestPostToRemovedFormRoutesReturns404 verifies POST to old form-based routes
// is no longer accepted. Acceptance item 1 + backlog 0163.
// Go's standard mux returns 405 (Method Not Allowed) when the path matches but
// the method doesn't — both 404 and 405 confirm the old handlers are gone.
func TestPostToRemovedFormRoutesReturns404(t *testing.T) {
	_, h := newApp(t)

	// POST /t/note (old create handler) should be gone.
	postRec := do(t, h, http.MethodPost, "/t/note", strings.NewReader("title=hello"), "application/x-www-form-urlencoded")
	if postRec.Code != http.StatusNotFound && postRec.Code != http.StatusMethodNotAllowed {
		t.Errorf("POST /t/note: expected 404 or 405 (handler removed), got %d", postRec.Code)
	}

	a2, _ := newApp(t)
	rec, _ := a2.Store.Create("note", map[string]any{"title": "Seed"})

	// POST /t/note/{id} (old update handler) should be gone.
	updateRec := do(t, h, http.MethodPost, "/t/note/"+rec.ID, strings.NewReader("title=updated"), "application/x-www-form-urlencoded")
	if updateRec.Code != http.StatusNotFound && updateRec.Code != http.StatusMethodNotAllowed {
		t.Errorf("POST /t/note/{id}: expected 404 or 405 (handler removed), got %d", updateRec.Code)
	}
}
