package server_test

import (
	"encoding/json"
	"net/http"
	"testing"
)

// TestDescribeNoHTMLNew verifies that /api/describe no longer contains the "html_new"
// route entry. After removing the HTML form-based create page, the routes map in
// app.go Describe() must not list it. This covers acceptance item 3.
func TestDescribeNoHTMLNew(t *testing.T) {
	_, h := newApp(t)

	rec := get(t, h, "/api/describe")
	wantStatus(t, rec, http.StatusOK)

	var d struct {
		Routes map[string]string `json:"routes"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &d); err != nil {
		t.Fatalf("bad JSON: %v", err)
	}

	if entry, ok := d.Routes["html_new"]; ok && entry != "" {
		t.Errorf("/api/describe should not contain 'html_new' route; got: %+v", d.Routes)
	}
}

// TestDescribeHasExpectedHTMLRoutes verifies that the routes map still lists the
// remaining HTML surfaces (list and detail) after removing html_new. This confirms
// we only removed what was intended, not the entire HTML layer.
func TestDescribeHasExpectedHTMLRoutes(t *testing.T) {
	_, h := newApp(t)

	rec := get(t, h, "/api/describe")
	wantStatus(t, rec, http.StatusOK)

	var d struct {
		Routes map[string]string `json:"routes"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &d); err != nil {
		t.Fatalf("bad JSON: %v", err)
	}

	for _, key := range []string{"html_list", "html_detail"} {
		if entry, ok := d.Routes[key]; !ok || entry == "" {
			t.Errorf("/api/describe should still contain route '%s'; got: %+v", key, d.Routes)
		}
	}
}
