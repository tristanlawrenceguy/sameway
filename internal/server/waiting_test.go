package server_test

import (
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/server"
)

// What waits for the person is one list (chat/waiting.go): Today shows
// every thing on it, each kind with its own answers, and only those; the
// assistant's questions are the owner's, so someone else is shown the
// rest.
func TestTodayShowsWhatWaits(t *testing.T) {
	t.Parallel()
	a, _ := newApp(t)
	srv := server.New(a)
	srv.SetUpSorting() // the starter tags
	mail := written(t, a.Store, "created", "email", "", map[string]any{"subject": "Gas bill", "from": "Gas Co", "body": "Please pay.", "tags": []any{"to sort"}})
	asked := written(t, a.Store, "created", "email", "", map[string]any{"subject": "Lunch", "from": "Ana", "body": "Tuesday works. 3pm?", "turn": "yours"})
	if _, err := records.ApplyOps(a.Store,
		records.Op{Type: records.ProposalType, After: map[string]any{"summary": "Remove the old tab?", "state": "pending", "action": map[string]any{"tool": "remove_canvas", "id": "x"}}},
		records.Op{Type: records.ProposalType, After: map[string]any{"summary": "Make a task: call Ana?", "state": "pending", "action": map[string]any{"tool": "create_record", "type": "task", "fields": map[string]any{"title": "Call Ana"}}, "by_action": "a1", "from": "email/" + asked.ID, "yes": "Make it", "no": "No"}},
		records.Op{Type: chat.ClassificationType, After: map[string]any{"record": "email/" + asked.ID, "tag": "important", "why": "someone waits", "state": "suggested"}},
		records.Op{Type: chat.ClassificationType, After: map[string]any{"record": "email/" + mail.ID, "tag": "nothing to do", "why": "no other tag fits", "state": "suggested"}},
	); err != nil {
		t.Fatal(err)
	}

	list := a.Chat.Waiting()
	var kinds []string
	for _, w := range list {
		kinds = append(kinds, w.Kind)
	}
	if got := strings.Join(kinds, " "); got != "question sort reply suggestion tag" {
		t.Fatalf("one of each, in order, and no question about a tag given when nothing else fits: %s", got)
	}
	today := get(t, srv, "/today").Body.String()
	for _, w := range list {
		for _, p := range w.Presses {
			if !strings.Contains(today, `action="`+p.Post+`"`) || !strings.Contains(today, ">"+p.Label+"<") {
				t.Errorf("Today answers the %s %q with %s at %s: %s", w.Kind, w.Words, p.Label, p.Post, truncate(today))
			}
		}
	}
	for _, h := range []string{"The assistant asks", "To sort", "Your turn to reply", "Suggested for you", "Tags to check"} {
		if !strings.Contains(today, "<h2>"+h+"</h2>") {
			t.Errorf("Today has %s", h)
		}
	}

	visitor := as(t, srv, hana, "GET", "/today", "", "").Body.String()
	if strings.Contains(visitor, "The assistant asks") || !strings.Contains(visitor, "Your turn to reply") {
		t.Errorf("the assistant's questions are the owner's; the rest is anyone's: %s", truncate(visitor))
	}
}
