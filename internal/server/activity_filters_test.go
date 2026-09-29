package server_test

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/app"
	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// seedLog makes a log with a change by each of you (a task), the
// assistant (a card), the system, and Sam (a note), and one change by the
// assistant ten days ago.
func seedLog(t *testing.T, a *app.App, h http.Handler) {
	t.Helper()
	task, err := a.Store.Create("task", map[string]any{"title": "Dig the pond"})
	if err != nil {
		t.Fatal(err)
	}
	chat.Record(a.Store, "human", chat.Change{Action: "created", Component: "task", ID: task.ID, Detail: "Dig the pond"})
	chat.Record(a.Store, "assistant", chat.Change{Action: "added", Component: "card", ID: "c1", Detail: "Plan the beds"})
	chat.Record(a.Store, "system", chat.Change{Action: "failed", Detail: "could not reach the model"})
	chat.Record(a.Store, "human", chat.Change{Action: "added", Component: "note", ID: "n1", Detail: "Fern cuttings", By: "Sam", ByLogin: "sam@example.com"})
	old := time.Now().AddDate(0, 0, -10)
	if _, err := a.Store.Put(chat.ActivityType, "oldentry", map[string]any{
		"summary": "Assistant added card Last spring", "actor": "assistant", "action": "added", "target": "card", "target_id": "c0", "detail": "Last spring",
	}, old, old); err != nil {
		t.Fatal(err)
	}
}

// The log offers who, what and when, made from what is in it, as a form
// with Apply that needs no script, and says how many changes it holds.
func TestActivityOffersWhoWhatAndWhen(t *testing.T) {
	a, h := newApp(t)
	seedLog(t, a, h)
	body := get(t, h, "/activity").Body.String()
	for _, want := range []string{
		`data-component="filters" data-shape="form"`, `<form method="get" action="/activity">`,
		`<legend class="sw-filters__legend">Show</legend>`,
		`name="who"`, `<option value="" selected>Anyone</option>`, `<option value="you">You</option>`,
		`<option value="assistant">Assistant</option>`, `<option value="system">System</option>`, `>Sam</option>`,
		`name="kind"`, `<option value="card">Cards</option>`, `<option value="task">Tasks</option>`, `<option value="note">Notes</option>`,
		`<option value="other">Other changes</option>`,
		`name="when"`, `<option value="today">Today</option>`, `<option value="week">Last 7 days</option>`,
		">Apply</button>", "5 changes.", "Last spring", "Dig the pond", "Fern cuttings",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("the log should offer %s\n%s", want, truncate(body))
		}
	}
	if strings.Contains(body, "Showing:") || strings.Contains(body, ">Reset<") {
		t.Error("the whole log says nothing is chosen")
	}
	if strings.Contains(body, "sam@example.com") {
		t.Error("another person is chosen by a fingerprint, not their login")
	}
	if strings.Index(body, ">You<") > strings.Index(body, ">Assistant<") || strings.Index(body, ">Assistant<") > strings.Index(body, ">Sam<") {
		t.Error("you first, then the assistant, then others by name")
	}
}

// Each choice narrows, several together narrow further, what is shown is
// said in words with Reset, and a value the log does not offer is ignored.
func TestActivityNarrowedTogether(t *testing.T) {
	a, h := newApp(t)
	seedLog(t, a, h)
	cases := []struct {
		query, showing string
		has, not       []string
	}{
		{"who=assistant", "by the assistant", []string{"Plan the beds", "Last spring"}, []string{"Dig the pond", "Fern cuttings", "could not reach"}},
		{"who=assistant&when=week", "by the assistant, in the last 7 days", []string{"Plan the beds"}, []string{"Last spring", "Dig the pond"}},
		{"kind=task&when=today", "to tasks, today", []string{"Dig the pond"}, []string{"Plan the beds", "Fern cuttings"}},
		{"who=you&kind=task&when=today", "by you, to tasks, today", []string{"Dig the pond"}, []string{"Plan the beds"}},
		{"when=week", "in the last 7 days", []string{"Dig the pond", "Plan the beds", "Fern cuttings"}, []string{"Last spring"}},
	}
	for _, c := range cases {
		body := get(t, h, "/activity?"+c.query).Body.String()
		if !strings.Contains(body, "Showing: "+c.showing+".") || !strings.Contains(body, `href="/activity">Reset</a>`) {
			t.Errorf("%s: says Showing: %s. with Reset\n%s", c.query, c.showing, truncate(body))
		}
		for _, want := range c.has {
			if !strings.Contains(body, want) {
				t.Errorf("%s: should show %s", c.query, want)
			}
		}
		for _, not := range c.not {
			if strings.Contains(body, not) {
				t.Errorf("%s: should not show %s", c.query, not)
			}
		}
	}
	sam := get(t, h, "/activity").Body.String()
	i := strings.Index(sam, ">Sam</option>")
	v := sam[strings.LastIndex(sam[:i], `value="`)+7:]
	v = v[:strings.Index(v, `"`)]
	if body := get(t, h, "/activity?who="+v).Body.String(); !strings.Contains(body, "Fern cuttings") || strings.Contains(body, "Dig the pond") || !strings.Contains(body, "Showing: by Sam.") {
		t.Errorf("another person is chosen by name: %s", v)
	}
	if body := get(t, h, "/activity?who=nobody&kind=spaceship").Body.String(); strings.Contains(body, "Showing:") || !strings.Contains(body, "5 changes.") {
		t.Error("the address cannot pick what the log does not offer")
	}
	none := get(t, h, "/activity?who=system&kind=task").Body.String()
	if !strings.Contains(none, "No changes match: by the system, to tasks.") || !strings.Contains(none, `href="/activity"`) {
		t.Errorf("nothing matching says the choices, with the way back\n%s", truncate(none))
	}
	if !strings.Contains(none, "<title>Activity: by the system, to tasks, no changes") {
		t.Error("the window title says what is shown")
	}
}

// The page's other address fields are kept and Reset keeps them; Undo on
// a narrowed log comes back to the same narrowing.
func TestActivityKeepsItsPlace(t *testing.T) {
	a, h := newApp(t)
	seedLog(t, a, h)
	body := get(t, h, "/activity?who=you&show=links").Body.String()
	if !strings.Contains(body, `<input type="hidden" name="show" value="links">`) || !strings.Contains(body, `href="/activity?show=links">Reset</a>`) {
		t.Errorf("the page's other fields are kept\n%s", truncate(body))
	}
	if !strings.Contains(body, `<input type="hidden" name="from" value="/activity?who=you&amp;show=links">`) {
		t.Fatalf("Undo carries the narrowed address\n%s", truncate(body))
	}
	entries, _ := a.Store.List(chat.ActivityType, store.ListOptions{})
	id := ""
	for _, e := range entries {
		if e.Fields["target"] == "task" {
			id = e.ID
		}
	}
	rec := postForm(t, h, "/activity/"+id+"/undo", url.Values{"from": {"/activity?who=you&show=links"}})
	if loc := rec.Header().Get("Location"); !strings.HasPrefix(loc, "/activity?who=you&show=links") {
		t.Errorf("Undo comes back to the narrowed log, got %q (%d)", loc, rec.Code)
	}
}

// A long log goes a page at a time, and the pages keep the choices.
func TestActivityPagesKeepTheChoices(t *testing.T) {
	a, h := newApp(t)
	for i := range 205 {
		chat.Record(a.Store, "assistant", chat.Change{Action: "added", Component: "card", ID: fmt.Sprint("c", i), Detail: fmt.Sprint("Card ", i)})
	}
	body := get(t, h, "/activity?who=assistant").Body.String()
	if !strings.Contains(body, "205 changes.") || !strings.Contains(body, `aria-label="Pages of activity"`) || !strings.Contains(body, `href="/activity?page=2&amp;who=assistant"`) && !strings.Contains(body, `href="/activity?who=assistant&amp;page=2"`) {
		t.Errorf("the pages keep the choices\n%s", truncate(body[strings.Index(body, "sw-pagination")-200:]))
	}
	if two := get(t, h, "/activity?who=assistant&page=2").Body.String(); !strings.Contains(two, ">Assistant added card Card 0<") && !strings.Contains(two, "Card 0<") {
		t.Error("the second page has the oldest")
	}
}

// An agent outside Sameway is chosen by the name it gave.
func TestActivityNamesAgents(t *testing.T) {
	a, h := newApp(t)
	seedLog(t, a, h)
	chat.Record(a.Store, chat.ActorAgent, chat.Change{Action: "added", Component: "note", ID: "n2", Detail: "Seed order", By: "Claude Code", Via: chat.ThroughAPI})
	body := get(t, h, "/activity").Body.String()
	i := strings.Index(body, ">Claude Code</option>")
	if i < 0 {
		t.Fatalf("an agent is offered by its name\n%s", truncate(body))
	}
	v := body[strings.LastIndex(body[:i], `value="`)+7:]
	v = v[:strings.Index(v, `"`)]
	if got := get(t, h, "/activity?who="+v).Body.String(); !strings.Contains(got, "Seed order") || strings.Contains(got, "Fern cuttings") {
		t.Errorf("choosing the agent shows its changes alone: %s", v)
	}
}
