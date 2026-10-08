package server_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/records"
)

// An agent's key changes things at most PaceChanges a minute; past that it
// is told how long to wait, it can still read, and the owner at this
// computer is not held back.
func TestAnAgentsChangesArePaced(t *testing.T) {
	a, h := newApp(t)
	key, _, err := records.LetAgentIn(a.Store, records.Who{Actor: "human", Via: records.ThroughCLI}, "Busy", "edit")
	if err != nil {
		t.Fatal(err)
	}
	// The budget comes back a change a second, and a slow machine earns
	// some back while it sends, so send until refused.
	made, res := 0, withKey(h, key, http.MethodPost, "/api/note", `{"title":"One more"}`)
	for ; res.Code == http.StatusCreated && made < 3*records.PaceChanges; made++ {
		res = withKey(h, key, http.MethodPost, "/api/note", `{"title":"One more"}`)
	}
	if made < records.PaceChanges {
		t.Errorf("the first %d changes are within the pace, refused after %d: %d %s", records.PaceChanges, made, res.Code, res.Body.String())
	}
	if res.Code != http.StatusTooManyRequests || res.Header().Get("Retry-After") == "" || !strings.Contains(res.Body.String(), `"slow_down"`) {
		t.Errorf("past the pace is 429 slow_down with Retry-After: %d %q %.300s", res.Code, res.Header().Get("Retry-After"), res.Body.String())
	}
	if res := withKey(h, key, http.MethodGet, "/api/note", ""); res.Code != http.StatusOK {
		t.Errorf("reading is not paced: %d", res.Code)
	}
	if res := postJSON(t, h, http.MethodPost, "/api/note", map[string]any{"title": "Mine"}); res.Code != http.StatusCreated {
		t.Errorf("the owner, with no key, is not paced: %d", res.Code)
	}
}
