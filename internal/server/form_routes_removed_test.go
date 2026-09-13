package server_test

import (
	"net/http"
	"testing"
)

// TestNoFormRoutesExist verifies that after removing HTML form-based CRUD,
// every route for creating or editing content via the browser returns 404.
// Acceptance item 1: no route lines for new or edit form pages in server.go
func TestNoFormRoutesExist(t *testing.T) {
	_, h := newApp(t)

	for _, path := range []string{"/t/note/new", "/t/activity/new"} {
		rec := get(t, h, path)
		if rec.Code != http.StatusNotFound {
			t.Errorf("GET %s = %d, want 404 (form route should be removed)", path, rec.Code)
		}
	}

	// POST to /t/{type} was the createForm handler — must also be gone.
	// The GET /t/{type} route still exists for listPage, so POST returns 405.
	for _, path := range []string{"/t/note", "/t/activity"} {
		rec := postForm(t, h, path, nil)
		if rec.Code != http.StatusMethodNotAllowed {
			t.Errorf("POST %s = %d, want 405 (create form route should be removed; GET listPage exists)", path, rec.Code)
		}
	}

	// GET /t/{type}/{id}/edit — the editPage handler.
	rec := get(t, h, "/t/note/fake-id/edit")
	if rec.Code != http.StatusNotFound {
		t.Errorf("GET /t/note/fake-id/edit = %d, want 404 (edit route should be removed)", rec.Code)
	}

	// POST to /t/{type}/{id} — the updateForm handler.
	// The GET /t/{type}/{id} route still exists for detailPage, so POST returns 405.
	rec = postForm(t, h, "/t/note/fake-id", nil)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("POST /t/note/fake-id = %d, want 405 (update form route should be removed; GET detailPage exists)", rec.Code)
	}

	// Verify remaining routes still work.
	wantStatus(t, get(t, h, "/t/note"), http.StatusOK)              // listPage
	wantStatus(t, get(t, h, "/activity"), http.StatusOK)            // activityPage
	wantStatus(t, get(t, h, "/canvas/abc123"), http.StatusNotFound) // canvas route exists but block not found
}
