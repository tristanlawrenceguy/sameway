package server_test

import (
	"net/http"
	"strings"
	"testing"
)

// TestListPageHasNoNewLink verifies that the listing page does not contain a
// "New {type}" link after form-based creation is removed.
func TestListPageHasNoNewLink(t *testing.T) {
	_, h := newApp(t)

	rec := get(t, h, "/t/note")
	wantStatus(t, rec, http.StatusOK)

	body := rec.Body.String()
	if strings.Contains(body, "New Note") || strings.Contains(body, `href="/t/note/new"`) {
		t.Error("list page should not contain a \"New\" link or /t/{type}/new href after form removal")
	}
}

// TestDetailPageHasNoEditLink verifies that the detail page does not contain an
// "Edit" link pointing to /t/{type}/{id}/edit.
func TestDetailPageHasNoEditLink(t *testing.T) {
	a, h := newApp(t)
	rec, err := a.Store.Create("note", map[string]any{"title": "Test"})
	if err != nil {
		t.Fatal(err)
	}

	detailRec := get(t, h, "/t/note/"+rec.ID)
	wantStatus(t, detailRec, http.StatusOK)

	body := detailRec.Body.String()
	if strings.Contains(body, `href="/t/note/`+rec.ID+`/edit"`) {
		t.Error("detail page should not contain an \"Edit\" link or /t/{type}/{id}/edit href after form removal")
	}

	// But the delete confirmation link must still be present.
	if !strings.Contains(body, "/confirm-delete") {
		t.Error("detail page should still show a delete confirmation link")
	}
}
