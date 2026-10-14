package api_test

import (
	"net/http"
	"strings"
	"testing"
)

// A list read with a word it does not take is refused, naming the word,
// what a list takes and how to ask for what was meant, never answered with
// every record as if filtered; the words it takes still work.
func TestAListRefusesAWordItDoesNotTake(t *testing.T) {
	t.Parallel()
	_, h := newApp(t)
	res := get(t, h, "/api/activity?since=today")
	if res.Code < 400 || !strings.Contains(res.Body.String(), "does not take since") || !strings.Contains(res.Body.String(), `created_at\u003e=today`) { // as JSON writes >
		t.Errorf("since is refused, with what to ask instead: %d %s", res.Code, res.Body)
	}
	for _, ok := range []string{"/api/task?where=done%3Dfalse&order=-due&limit=5&fields=title", "/api/note?page=1", "/api/block?canvas="} {
		if res := get(t, h, ok); res.Code != http.StatusOK {
			t.Errorf("%s still answers: %d %s", ok, res.Code, res.Body)
		}
	}
}
