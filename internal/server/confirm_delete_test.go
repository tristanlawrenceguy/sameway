package server_test

import (
	"net/http"
	"strings"
	"testing"

	"golang.org/x/net/html"

	"github.com/tristanlawrenceguy/sameway/internal/app"
	"github.com/tristanlawrenceguy/sameway/internal/render/htmltest"
)

// TestConfirmDeletePageReturns200 checks that GETting the confirmation page
// for a note returns HTTP 200 with an HTML body. This pins acceptance item 1:
// the confirm-delete route must be reachable and render successfully.
func TestConfirmDeletePageReturns200(t *testing.T) {
	a, h := newApp(t)
	noteID := createNote(t, h, a)

	path := "/t/note/" + noteID + "/confirm-delete"
	r := get(t, h, path)
	wantStatus(t, r, http.StatusOK)
	if !strings.Contains(r.Body.String(), "Delete note") {
		t.Errorf("expected page to mention 'Delete note', got body starting with %q", truncate(r.Body.String()))
	}
}

// TestConfirmDeletePageHasHeadingAndControls verifies the confirmation page
// contains the expected heading, a danger-variant confirm button that POSTs
// to the delete endpoint, and a Cancel link back to the detail page. This
// covers acceptance items 1 and 4: two controls on the screen, one dangerous
// submit, one cancel link.
func TestConfirmDeletePageHasHeadingAndControls(t *testing.T) {
	a, h := newApp(t)
	noteID := createNote(t, h, a)

	path := "/t/note/" + noteID + "/confirm-delete"
	doc := parse(t, get(t, h, path))

	// Heading: the page should mention "Delete note".
	bodyText := htmltest.Text(doc.Root)
	if !strings.Contains(bodyText, "Delete note") {
		t.Errorf("confirmation body text should contain 'Delete note', got %q", truncate(bodyText))
	}

	// Danger button that POSTs to the delete endpoint.
	buttons := doc.WithAttr("type", "submit")
	var dangerBtn *html.Node
	for _, b := range buttons {
		if strings.Contains(htmltest.Text(b), "Confirm deletion") {
			dangerBtn = b
			break
		}
	}
	if dangerBtn == nil {
		t.Errorf("confirmation page must have a 'Confirm deletion' submit button")
	}

	// The form containing the confirm button should POST to /t/note/{id}/delete.
	if dangerBtn != nil {
		form := findParentForm(dangerBtn)
		if form == nil {
			t.Error("confirm button must be inside a <form method=post>")
		} else if action, _ := htmltest.Attr(form, "action"); !strings.Contains(action, "/delete") {
			t.Errorf("confirm form should POST to /t/note/%s/delete, got action=%q", noteID, truncate(action))
		}
	}

	// Cancel link pointing back to the detail page.
	cancelLinks := doc.WithAttr("href", "")
	var foundCancel bool
	for _, a := range cancelLinks {
		if htmltest.Text(a) == "Cancel" {
			href, _ := htmltest.Attr(a, "href")
			if strings.Contains(href, "/t/note/"+noteID) {
				foundCancel = true
			}
		}
	}
	if !foundCancel {
		t.Errorf("confirmation page must have a 'Cancel' link pointing back to the detail page")
	}

	assertAllComponentsKnown(t, doc)
}

// TestConfirmDeleteUnknownTypeNotFound ensures that requesting the confirm
// delete page for an unknown type returns 404. This is important because
// the handler looks up the type and should not leak information or crash.
func TestConfirmDeleteUnknownTypeNotFound(t *testing.T) {
	_, h := newApp(t)
	r := get(t, h, "/t/unknowntype/abc123/confirm-delete")
	wantStatus(t, r, http.StatusNotFound)
}

// TestConfirmDeleteNonExistentRecordNotFound ensures that requesting the
// confirm delete page for a non-existent record returns 404. The handler
// must check both type validity and record existence.
func TestConfirmDeleteNonExistentRecordNotFound(t *testing.T) {
	_, h := newApp(t)
	r := get(t, h, "/t/note/does-not-exist/confirm-delete")
	if r.Code != http.StatusNotFound && r.Code != http.StatusInternalServerError {
		t.Errorf("expected 404 or 500 for non-existent record, got %d: %s", r.Code, truncate(r.Body.String()))
	}
}

// TestDirectPostDeleteStillWorks verifies that POSTing directly to the delete
// endpoint still deletes the record and redirects. This is acceptance item 2:
// existing form-based delete flows (tests, automation) continue to work
// unchanged even though the UI no longer uses an inline form on the detail page.
func TestDirectPostDeleteStillWorks(t *testing.T) {
	a, h := newApp(t)

	// Create a note and capture its ID.
	rec, _ := a.Store.Create("note", map[string]any{"title": "Confirm backward compat"})
	detailPath := "/t/note/" + rec.ID

	// POST directly to the delete endpoint (not via confirm page).
	del := postForm(t, h, detailPath+"/delete", nil)
	wantStatus(t, del, http.StatusSeeOther)

	// The record should be gone.
	gone := get(t, h, detailPath)
	wantStatus(t, gone, http.StatusNotFound)
}

// TestCanvasDeleteUnaffected ensures that canvas block deletion is NOT gated
// by a confirmation page — it remains a direct destructive action. This covers
// acceptance item 3: /canvas/{id}/delete stays unchanged.
func TestCanvasDeleteUnaffected(t *testing.T) {
	a, h := newApp(t)

	// Create a canvas block directly in the store.
	blockRec, err := a.Store.Create("block", map[string]any{
		"component": "text", "props": map[string]any{},
		"position": 0, "span": 12,
	})
	if err != nil {
		t.Fatal(err)
	}

	// Canvas delete should work directly without a confirm page.
	del := postForm(t, h, "/canvas/"+blockRec.ID+"/delete", nil)
	wantStatus(t, del, http.StatusSeeOther)

	// The block should be gone.
	gone := get(t, h, "/canvas/"+blockRec.ID)
	if gone.Code != http.StatusNotFound {
		t.Errorf("expected 404 for removed canvas block, got %d", gone.Code)
	}
}

// TestDetailPageDeleteIsLink verifies that the detail page no longer contains
// an inline form for deletion; instead it has a link to the confirm-delete
// endpoint. This is acceptance item 4: clicking Delete navigates to the
// confirmation page, not a direct POST.
func TestDetailPageDeleteIsLink(t *testing.T) {
	a, h := newApp(t)
	noteID := createNote(t, h, a)

	doc := parse(t, get(t, h, "/t/note/"+noteID))

	// The detail page should have a link to the confirm-delete endpoint.
	confirmLinks := doc.WithAttr("href", "")
	var foundConfirm bool
	for _, a := range confirmLinks {
		href, _ := htmltest.Attr(a, "href")
		if strings.Contains(href, "/confirm-delete") && strings.Contains(htmltest.Text(a), "Delete") {
			foundConfirm = true
		}
	}
	if !foundConfirm {
		t.Error("detail page must have a link to the confirm-delete endpoint with label 'Delete note'")
	}

	// The detail page should NOT contain an inline delete form.
	deleteForms := doc.WithAttr("action", "")
	for _, f := range deleteForms {
		action, _ := htmltest.Attr(f, "action")
		if strings.Contains(action, "/delete") && !strings.Contains(action, "/confirm-delete") {
			t.Error("detail page must not contain an inline <form> that POSTs to /delete; it should link to confirm-delete instead")
		}
	}

	assertAllComponentsKnown(t, doc)
}

// findParentForm walks up from a node looking for the nearest <form>.
func findParentForm(n *html.Node) *html.Node {
	cur := n.Parent
	for cur != nil {
		if cur.Data == "form" {
			return cur
		}
		cur = cur.Parent
	}
	return nil
}

// createNote creates a note in the store and returns its ID.
func createNote(t *testing.T, h http.Handler, a *app.App) string {
	t.Helper()
	rec, err := a.Store.Create("note", map[string]any{"title": "test"})
	if err != nil {
		t.Fatal(err)
	}
	return rec.ID
}
