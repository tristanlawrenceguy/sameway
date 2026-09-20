package server_test

import (
	"net/http"
	"strings"
	"testing"
)

// POSTing to /t/note/{id}/delete after deleting a note returns an HTML page
// with a success alert saying the note was deleted, then redirects.
func TestNoteDeletionShowsConfirmationAlert(t *testing.T) {
	_, h := newApp(t)
	created := postJSON(t, h, http.MethodPost, "/api/note", map[string]any{"title": "Water the plants", "body": "Sunday."})
	var note struct{ ID string }
	decode(t, created, &note)

	rec := postForm(t, h, "/t/note/"+note.ID+"/delete", nil)
	wantStatus(t, rec, http.StatusOK)
	body := rec.Body.String()
	if !strings.Contains(body, "sw-alert") {
		t.Fatalf("the delete response should contain an alert; body: %s", truncate(body))
	}
	if !strings.Contains(body, `data-kind="success"`) {
		t.Fatalf(`the alert should have kind="success"; body: %s`, truncate(body))
	}
	bodyLower := strings.ToLower(body)
	if !strings.Contains(bodyLower, "deleted") && !strings.Contains(bodyLower, "removed") {
		t.Fatalf(`the alert message should contain "deleted" or "removed"; body: %s`, truncate(body))
	}
}

// After the confirmation page shows, there is a way to continue to the
// listing: either a meta refresh or an explicit link.
func TestNoteDeletionConfirmationRedirectsToListing(t *testing.T) {
	_, h := newApp(t)
	created := postJSON(t, h, http.MethodPost, "/api/note", map[string]any{"title": "Buy milk", "body": "Check the list."})
	var note struct{ ID string }
	decode(t, created, &note)

	rec := postForm(t, h, "/t/note/"+note.ID+"/delete", nil)
	wantStatus(t, rec, http.StatusOK)
	body := rec.Body.String()
	if !strings.Contains(body, `<a class="sw-link" href="/t/note">`) && !strings.Contains(body, `http-equiv="refresh"`) {
		t.Fatalf(`the confirmation page should have a link or meta refresh to /t/note; body: %s`, truncate(body))
	}
}
