package server_test

import (
	"net/http"
	"strings"
	"testing"
)

// TestListPageHasNoNewLink verifies the list page no longer renders a "New {Type}"
// link to /t/{type}/new. After removing the HTML form-based create surface, there
// should be no link on the listing page pointing to the new-page route. This covers
// acceptance item 7.
func TestListPageHasNoNewLink(t *testing.T) {
	_, h := newApp(t)

	rec := get(t, h, "/t/note")
	wantStatus(t, rec, http.StatusOK)

	body := rec.Body.String()
	if strings.Contains(body, "/t/note/new") {
		t.Errorf("list page should not link to /t/note/new; body contains it at:\n%s", truncate(rec.Body.String()))
	}
	// The label "New Note" (or "New notes") must also be absent since the only UI that
	// shows it was the create-link in listPage.
	if strings.Contains(body, "New note") || strings.Contains(body, "New Note") {
		t.Errorf("list page should not contain a 'New' link label; body:\n%s", truncate(rec.Body.String()))
	}
}

// TestListPageStillRendersCards verifies the list page still works normally — it
// shows records and does not regress in other ways when the new-link is removed.
func TestListPageStillRendersCards(t *testing.T) {
	a, h := newApp(t)

	// Create a record so there's something to list.
	_, err := a.Store.Create("note", map[string]any{"title": "Visible card"})
	if err != nil {
		t.Fatal(err)
	}

	rec := get(t, h, "/t/note")
	wantStatus(t, rec, http.StatusOK)

	body := rec.Body.String()
	if !strings.Contains(body, "Visible card") {
		t.Errorf("list page should show the created record; body:\n%s", truncate(rec.Body.String()))
	}
	// The list page heading should still be present.
	if !strings.Contains(body, "<h1>Notes</h1>") && !strings.Contains(body, "<h1>notes") {
		t.Errorf("list page should still have a Notes heading; body:\n%s", truncate(rec.Body.String()))
	}
}

// TestDetailPageHasNoEditLink verifies the detail page no longer renders an "Edit"
// link to /t/{type}/{id}/edit. After removing the HTML form-based edit surface,
// there should be no edit link on the detail page — only the delete-related controls.
func TestDetailPageHasNoEditLink(t *testing.T) {
	a, h := newApp(t)

	rec, err := a.Store.Create("note", map[string]any{"title": "Test detail edit removal"})
	if err != nil {
		t.Fatal(err)
	}

	detailRec := get(t, h, "/t/note/"+rec.ID)
	wantStatus(t, detailRec, http.StatusOK)

	body := detailRec.Body.String()
	if strings.Contains(body, "/edit") {
		t.Errorf("detail page should not link to /edit; body contains it at:\n%s", truncate(detailRec.Body.String()))
	}
	if strings.Contains(body, "Edit note") || strings.Contains(body, "Edit Note") {
		t.Errorf("detail page should not contain an 'Edit' label; body:\n%s", truncate(detailRec.Body.String()))
	}
}

// TestDetailPageStillHasDeleteLink verifies the detail page still shows a link to
// the confirm-delete endpoint. Delete remains as an action surface for humans — only
// create/edit forms are being removed.
func TestDetailPageStillHasDeleteLink(t *testing.T) {
	a, h := newApp(t)

	rec, err := a.Store.Create("note", map[string]any{"title": "Test detail delete stays"})
	if err != nil {
		t.Fatal(err)
	}

	detailRec := get(t, h, "/t/note/"+rec.ID)
	wantStatus(t, detailRec, http.StatusOK)

	body := detailRec.Body.String()
	if !strings.Contains(body, "/confirm-delete") {
		t.Errorf("detail page should still link to confirm-delete; body:\n%s", truncate(detailRec.Body.String()))
	}
	if !strings.Contains(body, "Delete note") && !strings.Contains(body, "Delete Note") {
		t.Errorf("detail page should still show a 'Delete' label; body:\n%s", truncate(detailRec.Body.String()))
	}
}
