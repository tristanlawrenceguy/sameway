package server_test

import (
	"net/http"
	"strings"
	"testing"
)

// Deleting a note returns the person to the list, where a success alert
// says the note was deleted and offers to put it back.
func TestNoteDeletionShowsConfirmationAlert(t *testing.T) {
	_, h := newApp(t)
	created := postJSON(t, h, http.MethodPost, "/api/note", map[string]any{"title": "Water the plants", "body": "Sunday."})
	var note struct{ ID string }
	decode(t, created, &note)

	rec := postForm(t, h, "/t/note/"+note.ID+"/delete", nil)
	body := after(t, h, rec).Body.String()
	if !strings.Contains(body, `action="/activity/`) || !strings.Contains(body, "sw-outcome__undo") {
		t.Fatalf("the message should carry Undo; body: %s", truncate(body))
	}
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

// Deleting from the note's own page, which is gone, lands on the list;
// deleting from anywhere else stays there.
func TestNoteDeletionConfirmationRedirectsToListing(t *testing.T) {
	_, h := newApp(t)
	created := postJSON(t, h, http.MethodPost, "/api/note", map[string]any{"title": "Buy milk", "body": "Check the list."})
	var note struct{ ID string }
	decode(t, created, &note)

	rec := withReferer(t, h, http.MethodPost, "/t/note/"+note.ID+"/delete", "/t/note/"+note.ID, "", "application/x-www-form-urlencoded")
	if _, at := landed(t, h, rec); at != "/t/note" {
		t.Errorf("deleting from the note's own page lands on the list, got %q", at)
	}
	created = postJSON(t, h, http.MethodPost, "/api/note", map[string]any{"title": "Buy bread"})
	decode(t, created, &note)
	rec = withReferer(t, h, http.MethodPost, "/t/note/"+note.ID+"/delete", "/c/kitchen", "", "application/x-www-form-urlencoded")
	if _, at := landed(t, h, rec); at != "/c/kitchen" {
		t.Errorf("deleting from elsewhere stays there, got %q", at)
	}
}
