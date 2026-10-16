package api_test

import (
	"net/http"
	"strings"
	"testing"
)

// What changed while the person was away is one ask: since takes a day in
// words as well as a cursor, and words it cannot read are refused with the
// ways that work.
func TestChangesSinceADay(t *testing.T) {
	t.Parallel()
	_, h := newApp(t)
	postJSON(t, h, http.MethodPost, "/api/note", map[string]any{"title": "Back from the trip"})
	for _, since := range []string{"yesterday", "-2w", "2020-01-01"} {
		if body := get(t, h, "/api/changes?since="+since).Body.String(); !strings.Contains(body, "Back from the trip") {
			t.Errorf("since=%s has the note made today: %s", since, body)
		}
	}
	if res := get(t, h, "/api/changes?since=whenever"); res.Code != http.StatusBadRequest || !strings.Contains(res.Body.String(), "yesterday, -2w") {
		t.Errorf("words it cannot read say the ways that work: %d %s", res.Code, res.Body)
	}
}
