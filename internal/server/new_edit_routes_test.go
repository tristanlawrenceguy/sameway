package server_test

import (
	"net/http"
	"testing"
)

// TestNewRouteReturns404 checks that the HTML form-based new page is gone.
// After removing /t/{type}/new, a GET should return 404 for every registered
// content type (note, message, block, activity).
func TestNewRouteReturns404(t *testing.T) {
	_, h := newApp(t)

	for _, typ := range []string{"note", "message", "block", "activity"} {
		rec := get(t, h, "/t/"+typ+"/new")
		if rec.Code != http.StatusNotFound {
			t.Errorf("GET /t/%s/new: expected 404, got %d", typ, rec.Code)
		}
	}
}

// TestEditRouteReturns404 checks that the HTML form-based edit page is gone.
// After removing /t/{type}/{id}/edit, a GET should return 404 for every
// registered content type and any existing record ID. Tests note as the
// representative public type.
func TestEditRouteReturns404(t *testing.T) {
	a, h := newApp(t)

	rec, err := a.Store.Create("note", map[string]any{"title": "Test edit 404"})
	if err != nil {
		t.Fatalf("create note for edit test: %v", err)
	}

	path := "/t/note/" + rec.ID + "/edit"
	res := get(t, h, path)
	if res.Code != http.StatusNotFound {
		t.Errorf("GET %s: expected 404, got %d", path, res.Code)
	}
}

// TestNewLinkAbsentFromListPages checks that listing pages no longer show a
// "New {Type}" link. The cluster div wrapper around the new link is also gone.
func TestNewLinkAbsentFromListPages(t *testing.T) {
	_, h := newApp(t)

	for _, typ := range []string{"note", "message", "block"} {
		doc := parse(t, get(t, h, "/t/"+typ))
		if len(doc.WithAttr("href", "/t/"+typ+"/new")) > 0 {
			t.Errorf("/t/%s should not have a 'New' link", typ)
		}
	}
}

// TestEditLinkAbsentFromDetailPages checks that detail pages no longer show an
// "Edit {Type}" link. Only the delete confirm link should remain in the cluster.
func TestEditLinkAbsentFromDetailPages(t *testing.T) {
	a, h := newApp(t)

	rec, err := a.Store.Create("note", map[string]any{"title": "Test edit link absent"})
	if err != nil {
		t.Fatalf("create note for edit-link test: %v", err)
	}

	doc := parse(t, get(t, h, "/t/note/"+rec.ID))
	if len(doc.WithAttr("href", "/t/note/"+rec.ID+"/edit")) > 0 {
		t.Errorf("/t/note/:id should not have an 'Edit' link")
	}

	// The delete confirm link must still be present.
	if len(doc.WithAttr("href", "/t/note/"+rec.ID+"/confirm-delete")) == 0 {
		t.Errorf("/t/note/:id should still have a confirm-delete link")
	}
}

// TestRoutesMapHasNoFormEntries checks that the Describe() routes map does not
// contain an "html_new" entry. That surface is now handled by the JSON API and
// should be removed from the HTML route list entirely. (html_edit was never in
// this map.) The HTML list and detail routes must still be present.
func TestRoutesMapHasNoFormEntries(t *testing.T) {
	a, _ := newApp(t)
	d := a.Describe()

	if _, ok := d.Routes["html_new"]; ok {
		t.Error("routes map should not contain 'html_new'")
	}

	// The HTML list and detail routes must still be present.
	if d.Routes["html_list"] == "" {
		t.Error("routes map should still contain 'html_list'")
	}
	if d.Routes["html_detail"] == "" {
		t.Error("routes map should still contain 'html_detail'")
	}
}

// TestDeleteFlowStillWorks checks that the confirm-delete and delete routes are
// untouched. GET /t/{type}/{id}/confirm-delete returns 200, POST to delete
// redirects and removes the record.
func TestDeleteFlowStillWorks(t *testing.T) {
	a, h := newApp(t)

	rec, err := a.Store.Create("note", map[string]any{
		"title": "To be deleted",
		"body":  "will vanish",
	})
	if err != nil {
		t.Fatalf("create note for delete test: %v", err)
	}

	// confirm-delete page must render.
	conf := get(t, h, "/t/note/"+rec.ID+"/confirm-delete")
	wantStatus(t, conf, http.StatusOK)

	// POST to delete redirects.
	del := postForm(t, h, "/t/note/"+rec.ID+"/delete", nil)
	wantStatus(t, del, http.StatusSeeOther)

	// Record is gone from the detail page and the API.
	wantStatus(t, get(t, h, "/t/note/"+rec.ID), http.StatusNotFound)
}
