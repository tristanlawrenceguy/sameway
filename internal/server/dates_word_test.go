package server_test

import (
	"strings"
	"testing"
	"time"
)

// A task past its day is called one thing wherever it is said (WCAG
// 3.2.4, NN/g consistency): Overdue on Today, in its list, on its page
// and in the glance an agent reads, never Late on one and Overdue on
// another.
func TestOverdueIsOneWordEverywhere(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)
	now := time.Now()
	rec, _ := a.Store.Create("task", map[string]any{"title": "Pay rent", "due": now.AddDate(0, 0, -2).Format("2006-01-02")})
	for path, want := range map[string]string{
		"/today":                  ">Overdue<",
		"/t/task":                 ">Overdue ",
		"/t/task/" + rec.ID:       "Overdue, due ",
		"/api/task/" + rec.ID:     `"glance": "Overdue, due `,
		"/api/look?path=%2Ftoday": `"Overdue"`,
	} {
		body := get(t, h, path).Body.String()
		if !strings.Contains(body, want) {
			t.Errorf("%s says %s: %s", path, want, truncate(body))
		}
		if strings.Contains(body, ">Late<") || strings.Contains(body, `"Late"`) {
			t.Errorf("%s calls it Late: %s", path, truncate(body))
		}
	}
}

// What a dealing with a person says is named for a person: With Ana
// Silva, not Person Ana Silva; its kind is its Type, the word a reminder
// uses for the same thing.
func TestInteractionSaysWithAndType(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)
	p, _ := a.Store.Create("person", map[string]any{"name": "Ana Silva"})
	rec, _ := a.Store.Create("interaction", map[string]any{"summary": "Talked about the trip", "person": p.ID, "kind": "call", "at": time.Now().AddDate(0, 0, -1).Format(time.RFC3339)})
	page := get(t, h, "/t/interaction/"+rec.ID).Body.String()
	for _, want := range []string{"With <bdi>Ana Silva</bdi>", "<dt>With</dt>", "<dt>Type</dt>"} {
		if !strings.Contains(page, want) {
			t.Errorf("the page says %s: %s", want, truncate(page))
		}
	}
	for _, not := range []string{"Person <bdi>Ana Silva", "<dt>Person</dt>", "<dt>Kind</dt>"} {
		if strings.Contains(page, not) {
			t.Errorf("the page does not say %s", not)
		}
	}
	if api := get(t, h, "/api/interaction/"+rec.ID).Body.String(); !strings.Contains(api, "With Ana Silva") {
		t.Errorf("an agent reads the same words: %s", truncate(api))
	}
}
