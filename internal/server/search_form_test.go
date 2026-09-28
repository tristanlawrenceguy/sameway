package server_test

import (
	"net/http"
	"strings"
	"testing"
)

// The search page wraps its input and button in a <form> element.
func TestSearchPageHasFormElement(t *testing.T) {
	_, h := newApp(t)
	postJSON(t, h, http.MethodPost, "/api/note", map[string]any{"title": "Call the plumber", "body": "About the kitchen tap."})

	page := get(t, h, "/search?q=plumber")
	if page.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", page.Code)
	}
	body := page.Body.String()
	if !strings.Contains(body, `<form method="get" action="/search"`) {
		t.Errorf("the search page should contain a <form> element wrapping the input and button; body:\n%s", body)
	}
	if !strings.Contains(body, `</form>`) {
		t.Error("the search page should close the </form>")
	}
}
