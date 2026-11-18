package server_test

import (
	"strings"
	"testing"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/server"
)

// What time alone brings to match sets an action off, as a change would:
// a visit coming to start within a day has its reminder made, once; one
// already within a day when first looked at, and one still days away, do
// not.
func TestTimePassingSetsOffWhatComesToMatch(t *testing.T) {
	t.Parallel()
	a, h := newAppAt(t, tonight)
	srv := h.(*server.Server)
	rang := make(chan string, 4)
	srv.OnRing(func(title, text, _ string) { rang <- title + ": " + text })
	at := func(d time.Duration) string { return tonight.Add(d).UTC().Format(time.RFC3339) }
	a.Store.Create("event", map[string]any{"title": "Visit from Pat", "starts": at(30 * time.Hour)})
	a.Store.Create("event", map[string]any{"title": "Lesson", "starts": at(2 * time.Hour)})
	a.Store.Create("event", map[string]any{"title": "Holiday", "starts": at(10 * 24 * time.Hour)})
	a.Store.Create("action", map[string]any{"title": "Remind them the day before", "kind": "message", "message": "{{title}} is tomorrow. Remind them.", "when": "changed", "what": "event", "only": []any{"starts<+1d", "starts>now"}})
	a.Chat.Provider = &scripted{steps: []*llm.Response{{Text: "I have reminded them."}, {Text: "Again."}}}
	a.Automate.StartWorker()

	if err := a.Automate.TimePassed(tonight); err != nil {
		t.Fatal(err)
	}
	a.Automate.TimePassed(tonight.Add(time.Hour))
	select {
	case got := <-rang:
		t.Fatalf("nothing time has not brought sets anything off: %s", got)
	case <-time.After(300 * time.Millisecond):
	}
	a.Automate.TimePassed(tonight.Add(7 * time.Hour))
	select {
	case got := <-rang:
		if !strings.Contains(got, "Remind them the day before") {
			t.Errorf("the visit coming within a day sets it off: %s", got)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the visit coming within a day sets it off")
	}
	a.Automate.TimePassed(tonight.Add(8 * time.Hour))
	select {
	case got := <-rang:
		t.Errorf("once: %s", got)
	case <-time.After(300 * time.Millisecond):
	}
}
