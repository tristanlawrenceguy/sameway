package server_test

import (
	"net/http"
	"strings"
	"testing"
)

// What differed by way in, now one rule each.
func TestOneRuleWhereThereWereSeveral(t *testing.T) {
	a, h := newApp(t)

	// A field Sameway keeps is refused to the owner over the API as on
	// the page; it once went through.
	carol, _ := a.Store.Create("person", map[string]any{"name": "Carol", "email": "carol@example.com"})
	if res := postJSON(t, h, http.MethodPatch, "/api/person/"+carol.ID, map[string]any{"access": "edit"}); res.Code != http.StatusUnprocessableEntity {
		t.Errorf("the owner sets access by hand over the API: %d", res.Code)
	}

	// A list is ordered the one way: an order the type cannot take is
	// refused over the API as on the page.
	if res := get(t, h, "/api/note?order=colour"); res.Code != http.StatusBadRequest {
		t.Errorf("an unknown order over the API: %d", res.Code)
	}

	// A record on the canvas says what its page says: not a habit's goal of 0.
	var made map[string]any
	decode(t, postJSON(t, h, http.MethodPost, "/api/habit", map[string]any{"name": "Stretch", "goal": 0}), &made)
	postJSON(t, h, http.MethodPost, "/api/block", map[string]any{"component": "record", "props": map[string]any{"type": "habit", "record": made["id"]}})
	canvas := get(t, h, "/").Body.String()
	if !strings.Contains(canvas, "Stretch") || strings.Contains(canvas, ">Goal<") {
		t.Errorf("the canvas shows the habit without a goal of 0")
	}
}
