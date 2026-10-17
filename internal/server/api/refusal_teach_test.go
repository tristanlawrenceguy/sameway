package api_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

// A write refused for its fields says, for each, what goes there with an
// example, and where the type is described; a field the type does not
// have is answered with the nearest it has and every field it takes.
func TestARefusedWriteSaysWhatEachFieldTakes(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)
	habit, _ := a.Store.Create("habit", map[string]any{"name": "Running", "cadence": "day", "unit": "miles"})
	type refusal struct {
		Error struct {
			Fields   map[string]string `json:"fields"`
			Expects  map[string]string `json:"expects"`
			Describe string            `json:"describe"`
		} `json:"error"`
	}
	var r refusal
	res := postJSON(t, h, http.MethodPost, "/api/entry", map[string]any{"habit": habit.ID, "amount": 5})
	json.Unmarshal(res.Body.Bytes(), &r)
	if res.Code != http.StatusUnprocessableEntity || r.Error.Fields["at"] != "is required" || !strings.Contains(r.Error.Expects["at"], "2026-10-14T09:00 ") || r.Error.Describe != "/api/describe/types/entry" {
		t.Fatalf("a missing moment says its shape and where entry is described: %d %s", res.Code, res.Body)
	}
	if d := get(t, h, r.Error.Describe); d.Code != http.StatusOK || !strings.Contains(d.Body.String(), `"at"`) {
		t.Errorf("the describe it points to answers with the type: %d", d.Code)
	}
	var u refusal
	json.Unmarshal(postJSON(t, h, http.MethodPost, "/api/note", map[string]any{"titel": "Seeds"}).Body.Bytes(), &u)
	if said := u.Error.Expects["titel"]; !strings.Contains(said, "did you mean title?") || !strings.Contains(said, "It takes ") {
		t.Errorf("a field it does not have is answered with the nearest and all it takes: %q", said)
	}
}
