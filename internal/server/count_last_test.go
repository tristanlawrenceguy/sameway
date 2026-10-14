package server_test

import (
	"strings"
	"testing"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/when"
)

// Someone keeping up with friends sees on each person when they last
// spoke: the count of interactions says the latest that has happened,
// never one still to come, on the person's page and in the people list.
// What is ticked off, like a project's tasks, says no last.
func TestCountsSayWhenTheLatestWas(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)
	now := time.Now()
	friend, _ := a.Store.Create("person", map[string]any{"name": "Sam"})
	a.Store.Create("interaction", map[string]any{"summary": "Coffee", "person": friend.ID, "at": now.AddDate(0, 0, -20).Format(time.RFC3339)})
	a.Store.Create("interaction", map[string]any{"summary": "Phone call about the move", "person": friend.ID, "at": now.AddDate(0, 0, -3).Format(time.RFC3339)})
	a.Store.Create("interaction", map[string]any{"summary": "Dinner next week", "person": friend.ID, "at": now.AddDate(0, 0, 6).Format(time.RFC3339)})
	want := "3 interactions, last " + when.Day(now.AddDate(0, 0, -3), now)

	if p := get(t, h, "/t/person/"+friend.ID).Body.String(); !strings.Contains(p, want) {
		t.Errorf("the person's page says %q: %s", want, truncate(p))
	}
	if p := get(t, h, "/t/person").Body.String(); !strings.Contains(p, want) {
		t.Errorf("the people list says %q: %s", want, truncate(p))
	}

	// Logged with no day, it happened when it was logged.
	other, _ := a.Store.Create("person", map[string]any{"name": "Jo"})
	a.Store.Create("interaction", map[string]any{"summary": "Quick message", "person": other.ID})
	if p := get(t, h, "/t/person/"+other.ID).Body.String(); !strings.Contains(p, "1 interaction, last today") {
		t.Errorf("one logged with no day is last today: %s", truncate(p))
	}

	project, _ := a.Store.Create("project", map[string]any{"title": "Move house"})
	a.Store.Create("task", map[string]any{"title": "Book the van", "project": project.ID, "due": now.AddDate(0, 0, -1).Format("2006-01-02")})
	if p := get(t, h, "/t/project/"+project.ID).Body.String(); strings.Contains(p, "1 task, last") {
		t.Error("tasks are things to do, so their count says no last")
	}
}
