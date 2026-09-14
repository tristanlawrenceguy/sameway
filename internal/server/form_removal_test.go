package server_test

import (
	"net/http"
	"strings"
	"testing"
)

// TestFormRoutesReturn404 asserts that the form-based create/edit surfaces
// have been removed: GET /t/{type}/new, GET /t/{type}/{id}/edit and their
// POST counterparts all return 404.
func TestFormRoutesReturn404(t *testing.T) {
	a, h := newApp(t)

	// Create a record so the edit path at least has an ID to target;
	// we only care about whether the route handler exists, not the data.
	rec, err := a.Store.Create("note", map[string]any{"title": "For form removal"})
	if err != nil {
		t.Fatal(err)
	}

	for _, tc := range []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/t/note/new"},
		{http.MethodPost, "/t/note"},
		{http.MethodGet, "/t/note/" + rec.ID + "/edit"},
		{http.MethodPost, "/t/note/" + rec.ID},
	} {
		rec := do(t, h, tc.method, tc.path, nil, "")
		// GET returns 404 (no route at all). POST returns 405 because Go's
		// ServeMux matches the path pattern from GET /t/{type} and rejects
		// unmatched methods with 405 — not a bug, just how net/http works.
		if rec.Code != http.StatusNotFound && rec.Code != http.StatusMethodNotAllowed {
			t.Errorf("%s %s: expected 404 or 405 (route removed), got %d", tc.method, tc.path, rec.Code)
		}
	}

	// Also check a non-existent type still returns 404 (not a panic).
	r := do(t, h, http.MethodGet, "/t/nope/new", nil, "")
	if r.Code != http.StatusNotFound {
		t.Errorf("GET /t/nope/new: expected 404, got %d", r.Code)
	}

	r = do(t, h, http.MethodGet, "/t/nope/someid/edit", nil, "")
	if r.Code != http.StatusNotFound {
		t.Errorf("GET /t/nope/someid/edit: expected 404, got %d", r.Code)
	}
}

// TestDescribeHasNoHtmlNew asserts that the routes map in Describe() no longer
// contains an "html_new" entry. This is how agents learn which surfaces exist;
// a stale "new" route would mislead them into trying to compose forms.
func TestDescribeHasNoHtmlNew(t *testing.T) {
	_, h := newApp(t)

	rec := get(t, h, "/api/describe")
	wantStatus(t, rec, http.StatusOK)

	var d struct {
		Routes map[string]string
	}
	decode(t, rec, &d)

	if _, ok := d.Routes["html_new"]; ok {
		t.Errorf("routes should not contain 'html_new' — form-based creation is removed")
	}
}

// TestDeleteFlowStillWorks asserts that the delete confirmation page and the
// confirm-delete handler are untouched by this change. They belong to actions,
// not composition, and must stay.
func TestDeleteFlowStillWorks(t *testing.T) {
	a, h := newApp(t)

	rec, err := a.Store.Create("note", map[string]any{"title": "To delete"})
	if err != nil {
		t.Fatal(err)
	}

	// The confirm-delete page must render.
	delGet := get(t, h, "/t/note/"+rec.ID+"/confirm-delete")
	wantStatus(t, delGet, http.StatusOK)

	body := delGet.Body.String()
	if !strings.Contains(body, "Delete") {
		t.Errorf("confirm-delete page should mention Delete: %s", truncate(body))
	}
}

// TestAPIDeleteStillWorks asserts that the DELETE API endpoint for notes is
// still operational after removing form surfaces. The agent path must not break.
func TestAPIDeleteStillWorks(t *testing.T) {
	a, h := newApp(t)

	rec, err := a.Store.Create("note", map[string]any{"title": "Delete me"})
	if err != nil {
		t.Fatal(err)
	}

	delRec := do(t, h, http.MethodDelete, "/api/note/"+rec.ID, nil, "")
	wantStatus(t, delRec, http.StatusOK)

	// The record should be gone.
	getRec := get(t, h, "/api/note/"+rec.ID)
	if getRec.Code != http.StatusNotFound {
		t.Errorf("after delete GET /api/note/%s: expected 404, got %d", rec.ID, getRec.Code)
	}
}

// TestListPageHasNoNewLink asserts that the list page for content types no
// longer renders a "New {type}" link, since form-based creation is removed.
func TestListPageHasNoNewLink(t *testing.T) {
	_, h := newApp(t)

	rec := get(t, h, "/t/note")
	wantStatus(t, rec, http.StatusOK)

	body := rec.Body.String()
	if strings.Contains(body, `href="/t/note/new"`) {
		t.Errorf("list page should not contain href to /t/note/new; form creation is removed")
	}
}

// TestDetailPageHasNoEditLink asserts that the detail page for a record no
// longer renders an "Edit {type}" link, since form-based editing is removed.
func TestDetailPageHasNoEditLink(t *testing.T) {
	a, h := newApp(t)

	rec, err := a.Store.Create("note", map[string]any{"title": "No edit for me"})
	if err != nil {
		t.Fatal(err)
	}

	getRec := get(t, h, "/t/note/"+rec.ID)
	wantStatus(t, getRec, http.StatusOK)

	body := getRec.Body.String()
	if strings.Contains(body, `/t/note/`+rec.ID+`/edit"`) {
		t.Errorf("detail page should not contain href to /t/note/:id/edit; form editing is removed")
	}
}
