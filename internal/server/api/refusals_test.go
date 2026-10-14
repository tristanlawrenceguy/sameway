package api_test

import (
	"net/http"
	"strings"
	"testing"
)

// The same mistake gets the same words and the right status on every
// route: a type or a record not there is 404 named as the tools and the
// CLI name it; a query word a read does not take is refused with the one
// likely meant and what it takes; a number that is not one is said in
// words, never as Go's own error; a tool's argument it does not take is
// refused, never ignored.
func TestARefusalSaysWhatWasWrongTheSameEverywhere(t *testing.T) {
	t.Parallel()
	_, h := newApp(t)
	var made struct{ ID string }
	decode(t, postJSON(t, h, http.MethodPost, "/api/note", map[string]any{"title": "Kept"}), &made)
	cases := []struct {
		method, path, body string
		status             int
		want               []string
	}{
		{"GET", "/api/tasks", "", 404, []string{`"not_found"`, `"describe": "/api/describe/types"`, "there is no content type tasks (did you mean task?)", "GET /api/describe/types"}},
		{"POST", "/api/tasks", `{"title":"x"}`, 404, []string{"there is no content type tasks (did you mean task?)"}},
		{"GET", "/api/note/nope", "", 404, []string{"no note with id nope; GET /api/note lists those there are"}},
		{"PATCH", "/api/note/nope", `{"title":"x"}`, 404, []string{"no note with id nope"}},
		{"DELETE", "/api/note/nope", "", 404, []string{"no note with id nope"}},
		{"GET", "/api/note/" + made.ID + "?feilds=title", "", 400, []string{"does not take feilds (did you mean fields?). It takes fields"}},
		{"GET", "/api/note?limt=2", "", 400, []string{"does not take limt (did you mean limit?)"}},
		{"GET", "/api/changes?snce=x", "", 400, []string{"GET /api/changes does not take snce (did you mean since?). It takes since, limit, wait"}},
		{"GET", "/api/changes?wait=soon", "", 400, []string{`wait is how many seconds to wait for a change, a whole number from 0 to 60, not \"soon\"`}}, // as JSON writes it
		{"GET", "/api/describe?ful=1", "", 400, []string{"does not take ful (did you mean full?)"}},
		{"GET", "/api/note?limit=abc", "", 400, []string{`limit is how many to answer with, a whole number from 1, not \"abc\"`}},
		{"POST", "/api/tools/create_record", `{"type":"note","fields":{"title":"a"},"canvas":"x"}`, 422, []string{"create_record does not take canvas. It takes fields, type", "Nothing was done"}},
	}
	for _, c := range cases {
		res := do(t, h, c.method, c.path, strings.NewReader(c.body), "application/json")
		if res.Code != c.status {
			t.Errorf("%s %s: status %d, want %d: %s", c.method, c.path, res.Code, c.status, res.Body)
		}
		for _, w := range c.want {
			if !strings.Contains(res.Body.String(), w) {
				t.Errorf("%s %s: want %q in %s", c.method, c.path, w, res.Body)
			}
		}
		if strings.Contains(res.Body.String(), "strconv") {
			t.Errorf("%s %s: Go's own words reached the agent: %s", c.method, c.path, res.Body)
		}
	}
	var list struct{ Count int }
	decode(t, get(t, h, "/api/note"), &list)
	if list.Count != 1 {
		t.Errorf("a refused tool call made nothing: %d notes", list.Count)
	}
	// The log is read, never written: 405 says what the path does take.
	if res := do(t, h, http.MethodPut, "/api/activity/x", strings.NewReader("{}"), "application/json"); res.Code != http.StatusMethodNotAllowed || res.Header().Get("Allow") != "GET" {
		t.Errorf("a write to the log is 405 with Allow: GET, got %d %q", res.Code, res.Header().Get("Allow"))
	}
}
