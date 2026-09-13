package server_test

import (
	"net/http"
	"testing"
)

// TestDescribeHasNoHTMLNewOrHTMLEdit verifies that the agent-facing describe
// endpoint no longer advertises removed HTML form surfaces.
// Acceptance item 2: no "html_new" or "html_edit" entry in app.go Describe() Routes map
func TestDescribeHasNoFormRoutes(t *testing.T) {
	_, h := newApp(t)

	rec := get(t, h, "/api/describe")
	wantStatus(t, rec, http.StatusOK)

	var d struct {
		Routes map[string]string
	}
	decode(t, rec, &d)

	for _, key := range []string{"html_new"} {
		if val, ok := d.Routes[key]; ok && val != "" {
			t.Errorf("routes should not advertise %q = %q (form surface removed)", key, val)
		}
	}

	// Verify remaining routes are still advertised.
	for _, wantKey := range []string{"describe", "list", "create", "get", "update", "delete", "chat", "html_list", "html_detail", "css"} {
		if d.Routes[wantKey] == "" {
			t.Errorf("routes should still advertise %q", wantKey)
		}
	}

	// GET /t/{type}/{id} (detailPage) and DELETE are still registered — verify
	// they remain in the describe output.  html_detail maps to the detail page,
	// not an edit form.
	if d.Routes["html_detail"] == "" {
		t.Error("routes should still list html_detail for GET /t/{type}/{id}")
	}
}
