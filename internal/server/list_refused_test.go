package server_test

import (
	"net/http"
	"strings"
	"testing"
)

// A list page asked to narrow or order by what its kind does not have
// says so in a person's words, with the labels its pages use, and offers
// every record; no field as it is stored reaches the page.
func TestAListPageSaysARefusedFilterInWords(t *testing.T) {
	t.Parallel()
	_, h := newApp(t)
	for path, want := range map[string]string{
		"/t/event?where=mood%3Dx":   "Events have nothing called “mood”, so this list cannot be narrowed by it. They can be narrowed by ",
		"/t/event?order=-colour":    "Events have nothing called “colour” to put them in order by.",
		"/t/event?where=just+words": "“just words” is not a way to narrow events.",
	} {
		res := get(t, h, path)
		body := res.Body.String()
		if res.Code != http.StatusBadRequest || !strings.Contains(body, want) || !strings.Contains(body, "See all events") {
			t.Errorf("%s says %q: %d", path, want, res.Code)
		}
		for _, stored := range []string{"created_at", "updated_at", "has no field"} {
			if strings.Contains(body, stored) {
				t.Errorf("%s shows the stored word %q", path, stored)
			}
		}
	}
}
