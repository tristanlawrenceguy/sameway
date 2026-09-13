package server_test

import (
	"net/http"
	"strings"
	"testing"
)

// TestDescribeRoutesDoNotListFormPages checks that the routes map returned by
// Describe() no longer includes html_new or html_edit entries.  These keys
// must be absent because the form pages were removed.
func TestDescribeRoutesDoNotListFormPages(t *testing.T) {
	a, _ := newApp(t)

	desc := a.Describe()

	if _, ok := desc.Routes["html_new"]; ok {
		t.Error("routes map must not contain html_new — the form page was removed")
	}
	if _, ok := desc.Routes["html_edit"]; ok {
		t.Error("routes map must not contain html_edit — the form page was removed")
	}

	// But the remaining HTML routes should still be present.
	for _, key := range []string{"html_list", "html_detail"} {
		if val, ok := desc.Routes[key]; !ok {
			t.Errorf("routes map missing %q (should still exist)", key)
		} else if val == "" {
			t.Errorf("routes map has empty value for %q", key)
		}
	}

	// API routes must still be present.
	for _, key := range []string{"describe", "list", "create", "get", "update", "delete"} {
		if val, ok := desc.Routes[key]; !ok {
			t.Errorf("routes map missing %q (should still exist)", key)
		} else if val == "" {
			t.Errorf("routes map has empty value for %q", key)
		}
	}

	// The describe routes should not contain any "/new" or "/edit" paths.
	for k, v := range desc.Routes {
		if strings.Contains(v, "/new") || strings.Contains(v, "/edit") {
			t.Errorf("route %q points to a removed form path: %s", k, v)
		}
	}
}

// TestDeleteStillWorks ensures that after removing the form routes, deletion
// via confirm-delete page and POST delete handler still functions.  This is
// an acceptance criterion — deletion stays intact.
func TestDeleteStillWorks(t *testing.T) {
	a, h := newApp(t)

	// Create a note via API.
	rec, err := a.Store.Create("note", map[string]any{"title": "To delete"})
	if err != nil {
		t.Fatal(err)
	}

	// Confirm-delete page should still return 200.
	conf := get(t, h, "/t/note/"+rec.ID+"/confirm-delete")
	wantStatus(t, conf, http.StatusOK)

	// POST to delete should redirect (303).
	del := postForm(t, h, "/t/note/"+rec.ID+"/delete", nil)
	wantStatus(t, del, http.StatusSeeOther)

	// The record should now be gone.
	gone := get(t, h, "/t/note/"+rec.ID)
	wantStatus(t, gone, http.StatusNotFound)
}
