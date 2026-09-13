package server_test

import (
	"net/http"
	"net/url"
	"testing"
)

// TestFormRoutesAreGone verifies the four HTML form CRUD routes are removed:
// GET /t/{type}/new, POST /t/{type}, GET /t/{type}/{id}/edit, and
// POST /t/{type}/{id}.  Each should return 404.
func TestFormRoutesAreGone(t *testing.T) {
	_, h := newApp(t)

	t.Run("new page returns 404", func(t *testing.T) {
		rec := get(t, h, "/t/note/new")
		wantStatus(t, rec, http.StatusNotFound)
	})

	t.Run("create form post returns 404", func(t *testing.T) {
		rec := postForm(t, h, "/t/note", url.Values{"title": {"Hello"}})
		if rec.Code != http.StatusMethodNotAllowed && rec.Code != http.StatusNotFound {
			t.Errorf("POST /t/note = %d, want 405 or 404", rec.Code)
		}
	})

	t.Run("edit page returns 404 for note", func(t *testing.T) {
		rec := get(t, h, "/t/note/abc123/edit")
		wantStatus(t, rec, http.StatusNotFound)
	})

	t.Run("update form post returns 404 for note", func(t *testing.T) {
		rec := postForm(t, h, "/t/note/abc123", url.Values{"title": {"Hello"}})
		if rec.Code != http.StatusMethodNotAllowed && rec.Code != http.StatusNotFound {
			t.Errorf("POST /t/note/abc123 = %d, want 405 or 404", rec.Code)
		}
	})

	t.Run("edit page returns 404 for block", func(t *testing.T) {
		rec := get(t, h, "/t/block/some-id/edit")
		wantStatus(t, rec, http.StatusNotFound)
	})

	t.Run("update form post returns 404 for block", func(t *testing.T) {
		rec := postForm(t, h, "/t/block/some-id", url.Values{
			"component": {"text"},
			"props":     {`{"content":"hi"}`},
			"position":  {"0"},
		})
		if rec.Code != http.StatusMethodNotAllowed && rec.Code != http.StatusNotFound {
			t.Errorf("POST /t/block/some-id = %d, want 405 or 404", rec.Code)
		}
	})
}
