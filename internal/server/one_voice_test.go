package server_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
)

// A change in the log is said the same way wherever it is read, however
// it was stored: an entry written before settings had names, and the undo
// of it, read on the log, on the entry's own page and over the API. The
// stored words had a setting's description in them ("Pace — how changes
// arrive"), which the log cleaned and the entry's page and the API did
// not (crew findings 0566, 0571, 0572, 0574, 0582, 0583).
func TestAChangeIsSaidOneWayEverywhere(t *testing.T) {
	a, h := newApp(t)
	old, err := a.Store.Create("activity", map[string]any{
		"actor": "assistant", "action": "set", "target": "ui.pace", "detail": "calm",
		"summary": "Assistant changed Pace — how changes arrive to Calmly",
	})
	if err != nil {
		t.Fatal(err)
	}
	undo, _ := a.Store.Create("activity", map[string]any{
		"actor": "human", "action": "set", "target": "ui.pace", "detail": "quick", "undoes": old.ID,
		"summary": "You undid: Assistant changed Pace — how changes arrive to Calmly",
	})
	want := "Assistant changed pace to Calm"
	// Pages work the words out, so they say them before anything else runs.
	surfaces := map[string]string{
		"the log":          get(t, h, "/activity").Body.String(),
		"the entry's page": get(t, h, "/t/activity/"+old.ID).Body.String(),
		"the undo's page":  get(t, h, "/t/activity/"+undo.ID).Body.String(),
	}
	// The stored words are brought up to date when a workspace opens, so
	// what hands out the fields as they are (the API, MCP) agrees too.
	if n := chat.Resay(a.Store); n != 2 {
		t.Errorf("both old entries are said again, got %d", n)
	}
	surfaces["the API's record"] = get(t, h, "/api/activity/"+undo.ID).Body.String()
	surfaces["the API's list"] = get(t, h, "/api/activity").Body.String()
	for where, body := range surfaces {
		if strings.Contains(body, "how changes arrive") || strings.Contains(body, "Calmly") {
			t.Errorf("%s still says the stored words", where)
		}
	}
	for _, where := range []string{"the entry's page", "the API's record"} {
		body := surfaces[where]
		if where == "the API's record" {
			var rec map[string]any
			json.Unmarshal([]byte(body), &rec)
			body, _ = rec["title"].(string)
			want = "You undid: " + want
		}
		if !strings.Contains(body, want) {
			t.Errorf("%s says %q:\n%.600s", where, want, body)
		}
	}
}
