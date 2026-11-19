package server_test

import (
	"strings"
	"testing"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/server"
)

// An event that has just happened is a change an action can follow, such
// as asking the assistant to thank them; one long past, from a calendar
// brought in, or one still to come, is not.
func TestAnEventThatHappenedSetsOffWhatFollows(t *testing.T) {
	t.Parallel()
	a, h := newAppAt(t, tonight)
	srv := h.(*server.Server)
	rang := make(chan string, 4)
	srv.OnRing(func(title, text, _ string) { rang <- title + ": " + text })
	just, _ := a.Store.Create("event", map[string]any{"title": "Visit to Sam", "starts": tonight.Add(-3 * time.Hour).UTC().Format(time.RFC3339), "ends": tonight.Add(-2 * time.Hour).UTC().Format(time.RFC3339)})
	old, _ := a.Store.Create("event", map[string]any{"title": "Dentist", "starts": tonight.AddDate(-1, 0, 0).UTC().Format(time.RFC3339)})
	soon, _ := a.Store.Create("event", map[string]any{"title": "Lesson", "starts": tonight.Add(time.Hour).UTC().Format(time.RFC3339)})
	a.Store.Create("action", map[string]any{"title": "Thank them after a visit", "kind": "message", "message": "{{title}} has happened. Thank them.", "when": "changed", "what": "event", "only": []any{"over=true"}})
	a.Chat.Provider = &scripted{steps: []*llm.Response{{Text: "I have drafted a thank-you."}}}
	a.Automate.StartWorker()

	if err := a.Chat.MarkOver(tonight); err != nil {
		t.Fatal(err)
	}
	for id, want := range map[string]bool{just.ID: true, old.ID: false, soon.ID: false} {
		if e, _ := a.Store.Get("event", id); e.Fields["over"] != want {
			t.Errorf("%v over: %v", e.Fields["title"], e.Fields["over"])
		}
	}
	select {
	case got := <-rang:
		if !strings.Contains(got, "Thank them after a visit") {
			t.Errorf("what follows is set off: %s", got)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("what follows is set off")
	}
	a.Chat.MarkOver(tonight.Add(time.Hour))
	if len(rang) != 0 {
		t.Error("once")
	}
}
