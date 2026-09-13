package server_test

import (
	"net/http"
	"strings"
	"testing"
)

// TestNewRouteReturns404 verifies that GET /t/{type}/new no longer exists.
// After removing the HTML form-based create page, any request to this path
// should return 404 Not Found rather than a form. This covers acceptance item 1.
func TestNewRouteReturns404(t *testing.T) {
	_, h := newApp(t)

	r := get(t, h, "/t/note/new")
	wantStatus(t, r, http.StatusNotFound)

	body := r.Body.String()
	if strings.Contains(body, "New note") || strings.Contains(body, `<form`) {
		t.Errorf("GET /t/note/new should return 404, not a form page; body starts with %q", truncate(body))
	}
}

// TestEditRouteReturns404 verifies that GET /t/{type}/{id}/edit no longer exists.
// After removing the HTML form-based edit page, any request to this path should
// return 404 Not Found rather than an edit form. This covers acceptance item 1.
func TestEditRouteReturns404(t *testing.T) {
	a, h := newApp(t)

	// Create a note so we have a valid ID to try editing.
	rec, err := a.Store.Create("note", map[string]any{"title": "Test edit removal"})
	if err != nil {
		t.Fatal(err)
	}

	r := get(t, h, "/t/note/"+rec.ID+"/edit")
	wantStatus(t, r, http.StatusNotFound)

	body := r.Body.String()
	if strings.Contains(body, "Edit note") || strings.Contains(body, `<form`) {
		t.Errorf("GET /t/note/{id}/edit should return 404, not an edit form; body starts with %q", truncate(body))
	}
}

// TestCreatePostRouteReturns405 verifies that POST /t/{type} (the create form
// handler) no longer exists. After removing the HTML form-based create surface,
// this path returns 405 Method Not Allowed because Go's default mux matches on
// path first: GET /t/{type} still serves listPage for GET requests, so a POST to
// the same path gets "method not allowed" rather than 404. The absence of a form
// handler is confirmed by the body not containing any form markup. This covers acceptance item 1.
func TestCreatePostRouteReturns405(t *testing.T) {
	_, h := newApp(t)

	r := postForm(t, h, "/t/note", nil)
	wantStatus(t, r, http.StatusMethodNotAllowed)

	body := r.Body.String()
	if strings.Contains(body, `method="post"`) || strings.Contains(body, `<form`) {
		t.Errorf("POST /t/note should not render a form; body starts with %q", truncate(body))
	}
}

// TestUpdatePostRouteReturns405 verifies that POST /t/{type}/{id} (the update
// form handler) no longer exists. After removing the HTML form-based edit surface,
// this path returns 405 Method Not Allowed because Go's default mux matches on
// path first: GET /t/{type}/{id} still serves detailPage for GET requests, so a
// POST to the same path gets "method not allowed" rather than 404. The absence of
// a form handler is confirmed by the body not containing any form markup. This
// covers acceptance item 1.
func TestUpdatePostRouteReturns405(t *testing.T) {
	a, h := newApp(t)

	rec, err := a.Store.Create("note", map[string]any{"title": "Test update removal"})
	if err != nil {
		t.Fatal(err)
	}

	r := postForm(t, h, "/t/note/"+rec.ID, nil)
	wantStatus(t, r, http.StatusMethodNotAllowed)

	body := r.Body.String()
	if strings.Contains(body, `method="post"`) || strings.Contains(body, `<form`) {
		t.Errorf("POST /t/note/{id} should not render a form; body starts with %q", truncate(body))
	}
}
