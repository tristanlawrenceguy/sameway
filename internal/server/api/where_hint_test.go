package api_test

import (
	"net/http"
	"strings"
	"testing"
)

// A field asked for by its own name is answered with the condition that
// asks it: ?actor=assistant is where=actor=assistant.
func TestAFieldByItsNameIsPointedAtWhere(t *testing.T) {
	t.Parallel()
	_, h := newApp(t)
	res := get(t, h, "/api/activity?actor=assistant")
	if res.Code != http.StatusBadRequest || !strings.Contains(res.Body.String(), "?where=actor=assistant") {
		t.Errorf("the condition that asks it: %d %s", res.Code, res.Body.String())
	}
	if res := get(t, h, "/api/activity?where=actor%3Dassistant"); res.Code != http.StatusOK {
		t.Errorf("which works: %d", res.Code)
	}
}
