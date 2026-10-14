package server_test

import (
	"html"
	"net/http"
	"strings"
	"testing"
)

// People with a comma in their name ("Silva, Ana") are set apart by
// semicolons where several are named, on the page and in the words it is
// edited in, and those words sent back are the same people, not three.
func TestSeveralNamesWithCommasReadAsThemselves(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)
	ana, _ := a.Store.Create("person", map[string]any{"name": "Silva, Ana"})
	bob, _ := a.Store.Create("person", map[string]any{"name": "Bob Lee"})
	ev, _ := a.Store.Create("event", map[string]any{"title": "Pricing", "starts": "2026-10-06 14:00", "people": []any{ana.ID, bob.ID}})

	if page := html.UnescapeString(get(t, h, "/t/event/"+ev.ID).Body.String()); !strings.Contains(page, "Silva, Ana; Bob Lee") {
		t.Errorf("the people should be set apart by a semicolon on the event's page")
	}
	res := postJSON(t, h, http.MethodPatch, "/api/event/"+ev.ID, map[string]any{"people": "Bob Lee; Silva, Ana"})
	if res.Code >= 300 {
		t.Fatalf("names as the page writes them should be taken back: %d %s", res.Code, res.Body.String())
	}
	got, _ := a.Store.Get("event", ev.ID)
	if people, _ := got.Fields["people"].([]any); len(people) != 2 || people[0] != bob.ID || people[1] != ana.ID {
		t.Errorf("the two names should be the two people, got %v", got.Fields["people"])
	}
}
