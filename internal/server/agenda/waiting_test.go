package agenda_test

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/app"
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
	srv := seedWaiting(t, a)

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

	hana := records.Visitor{Name: "Hana", Login: "hana@example.com", Access: records.Edit}
	visitor := as(t, srv, hana, "GET", "/today", "", "").Body.String()
	if strings.Contains(visitor, "The assistant asks") || !strings.Contains(visitor, "Your turn to reply") {
		t.Errorf("the assistant's questions are the owner's; the rest is anyone's: %s", truncate(visitor))
	}
}

// seedWaiting puts one of each kind of thing that waits in a fresh
// workspace, and one tag that is not asked about.
func seedWaiting(t *testing.T, a *app.App) *server.Server {
	t.Helper()
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
	return srv
}

// Today, the morning brief and an agent count what waits from the same
// list, so they agree: on a day with nothing due, the brief still goes,
// saying how many wait and what.
func TestTodayTheBriefAndAgentsAgreeOnWhatWaits(t *testing.T) {
	t.Parallel()
	a, _ := newApp(t)
	srv := seedWaiting(t, a)
	n := len(a.Chat.Waiting())
	sent := make(chan string, 1)
	srv.OnRing(func(title, text, link string) { sent <- title + " | " + text })
	postForm(t, srv, "/brief", url.Values{"at": {"00:00"}})
	if !srv.BriefIfDue(time.Now()) {
		t.Fatal("what waits is worth a brief")
	}
	if got, want := <-sent, fmt.Sprintf("Today: %d waiting for you | Remove the old tab?; Gas bill; Lunch", n); got != want {
		t.Errorf("the brief counts what waits and names the first: %q, want %q", got, want)
	}
	api := do(t, srv, http.MethodPost, "/api/tools/what_waits", strings.NewReader(`{}`), "application/json").Body.String()
	if !strings.Contains(api, fmt.Sprintf("%d waiting for the person", n)) || !strings.Contains(api, "POST /mail/answered") {
		t.Errorf("an agent reads the same %d, with how to answer each: %s", n, api)
	}
}
