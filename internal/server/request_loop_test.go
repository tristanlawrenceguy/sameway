package server_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/server"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// A request is answered back to whoever asked, in their own conversation,
// and Always answers the same kind from them for good: the next one is
// made without asking, logged as theirs; Always is the owner's press.
func TestARequestIsAnsweredBackAndAlwaysAnswersForGood(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)
	vi, _ := a.Store.Create("person", map[string]any{"name": "Vi", "email": "vi@example.com", "access": "view"})
	bob, _ := a.Store.Create("person", map[string]any{"name": "Bob", "email": "bob@example.com", "access": "edit"})
	viewer := records.Visitor{Person: vi.ID, Name: "Vi", Login: "vi@example.com", Access: records.View}
	look := a.Chat.For(viewer)
	if text, isErr := look.Call("create_record", json.RawMessage(`{"type":"event","fields":{"title":"Visit","body":"x"}}`)); !isErr || len(a.Chat.Suggestions()) != 0 {
		t.Fatalf("a request a yes could not make goes back to the assistant to fix: %v %s", isErr, text)
	}
	if text, isErr := look.Call("create_record", json.RawMessage(`{"type":"event","fields":{"title":"Visit","people":["Nobody Here"]}}`)); !isErr || len(a.Chat.Suggestions()) != 0 {
		t.Fatalf("so does one whose link names nobody here: %v %s", isErr, text)
	}
	if text, _ := look.Call("create_record", json.RawMessage(`{"type":"note","fields":{"title":"Milk for Saturday"}}`)); !strings.Contains(text, "/request/") {
		t.Fatalf("they are told where to see the answer: %s", text)
	}
	today := get(t, h, "/today").Body.String()
	if !strings.Contains(today, "Always for Vi&#39;s notes") {
		t.Fatalf("a request from someone with a page offers Always\n%s", truncate(today))
	}
	asks := a.Chat.Suggestions()
	editor := records.Visitor{Person: bob.ID, Name: "Bob", Login: "bob@example.com", Access: records.Edit}
	if r := as(t, h, editor, http.MethodPost, "/suggested/always", url.Values{"id": {asks[0].ID}}.Encode(), "application/x-www-form-urlencoded"); r.Code != http.StatusForbidden {
		t.Errorf("Always is the owner's press alone: %d", r.Code)
	}
	wantStatus(t, postForm(t, h, "/suggested/always", url.Values{"id": {asks[0].ID}}), http.StatusSeeOther)

	msgs, _ := look.Messages()
	heard := false
	for _, m := range msgs {
		heard = heard || strings.Contains(m.Fields["content"].(string), "said yes to your request to make a note")
	}
	if !heard {
		t.Errorf("they hear the answer in their own conversation: %+v", msgs)
	}
	if p, _ := a.Store.Get("person", vi.ID); !strings.Contains(strings.Join(anyStrings(p.Fields["always"]), ","), "create_record note") {
		t.Fatalf("the standing yes is on their page: %v", p.Fields["always"])
	}

	if text, isErr := look.Call("create_record", json.RawMessage(`{"type":"note","fields":{"title":"Eggs too"}}`)); isErr || !strings.Contains(text, "without asking") {
		t.Errorf("the same kind from them is made without asking: %v %s", isErr, text)
	}
	if text, _ := look.Call("create_record", json.RawMessage(`{"type":"task","fields":{"title":"Mow"}}`)); !strings.Contains(text, "Nothing has changed") {
		t.Errorf("another kind is still asked: %s", text)
	}
	notes, _ := a.Store.List("note", store.ListOptions{})
	if len(notes) != 2 {
		t.Fatalf("both notes are made: %d", len(notes))
	}
	log, _ := a.Store.List(records.ActivityType, store.ListOptions{})
	byVi := false
	for _, e := range log {
		if s, _ := e.Fields["summary"].(string); strings.HasPrefix(s, "Vi ") && strings.Contains(s, "Eggs too") {
			t.Logf("LOG: %s", s)
			byVi = true
		}
	}
	if !byVi {
		t.Error("what is made without asking is logged as theirs")
		for _, e := range log {
			t.Log(e.Fields["summary"])
		}
	}
}

func anyStrings(v any) []string {
	list, _ := v.([]any)
	var out []string
	for _, x := range list {
		out = append(out, x.(string))
	}
	return out
}

// served reaches another workspace in this process as the internet does,
// at its published address, and nothing listens on the network.
type served struct{ h http.Handler }

func (s served) RoundTrip(r *http.Request) (*http.Response, error) {
	rec := httptest.NewRecorder()
	s.h.ServeHTTP(rec, r)
	return rec.Result(), nil
}

// The owner's assistant asks someone else's published Sameway, by its
// address or by the name of someone here whose page says where it is, put
// to the owner first; what it asks for there is a request their owner answers,
// and asking with the link it brought back says what became of it.
func TestMyAssistantAsksSomeoneElsesPublishedPage(t *testing.T) {
	t.Parallel()
	plumber, plumberH := newApp(t)
	plumber.Workspace.Config.Publish.Types = "note"
	plumber.Store.Create("note", map[string]any{"title": "Callouts", "body": "Weekdays 8 to 6."})
	plumber.Chat.Provider = &scripted{steps: []*llm.Response{
		toolCall("create_record", map[string]any{"type": "event", "fields": map[string]any{"title": "Visit Tris", "starts": "2026-03-12T10:00:00Z"}}),
		{Text: "I have asked for Thursday at 10; nothing is booked until they say yes."},
	}}
	internet := plumberH.(*server.Server).Public(nil)

	me, myH := newApp(t)
	me.Automate.HTTP = &http.Client{Transport: served{internet}}
	me.Store.Create("person", map[string]any{"name": "Ana the plumber", "sameway": "plumber.example.ts.net"})
	me.Chat.Provider = &scripted{steps: []*llm.Response{
		toolCall("ask_page", map[string]any{"address": "Ana the plumber", "question": "Can you come Thursday at 10?", "who": "Tris, tris@example.com"}),
		{Text: "I have asked you first."},
		toolCall("create_record", map[string]any{"type": "event", "fields": map[string]any{"title": "Ana the plumber visits", "starts": "2026-03-12T10:00:00Z"}}),
		{Text: "It is on your calendar."},
	}}
	if _, err := me.Chat.Send(context.Background(), "ask the plumber's page whether they can come Thursday at 10"); err != nil {
		t.Fatal(err)
	}
	if n, _ := plumber.Store.Count(records.ProposalType); n != 0 {
		t.Fatal("nothing is sent before the owner says yes")
	}
	ps := me.Chat.Proposals()
	if len(ps) != 1 || !strings.Contains(ps[0].Fields["summary"].(string), "Ask plumber.example.ts.net this?") {
		t.Fatalf("asking is put to the owner: %+v", ps)
	}
	wantStatus(t, postForm(t, myH, "/proposal/"+ps[0].ID+"/accept", nil), http.StatusSeeOther)

	theirs := plumber.Chat.Suggestions()
	if len(theirs) != 1 || !strings.Contains(theirs[0].Fields["summary"].(string), "Tris, tris@example.com asks you to make an event") {
		t.Fatalf("it is a request on the plumber's Today: %+v", theirs)
	}
	link := "https://plumber.example.ts.net/request/" + theirs[0].Fields["token"].(string)
	sent, _ := me.Store.List("ask", store.ListOptions{})
	if len(sent) != 1 || sent[0].Fields["at"] != link || sent[0].Fields["state"] != "waiting" {
		t.Fatalf("what it asked for there is kept here, with its link, waiting: %+v", sent)
	}
	// What to do with an answer is the owner's to say, once, as any
	// automation: here, their assistant puts what is agreed on the calendar.
	me.Store.Create("action", map[string]any{"title": "Put what they agree to on my calendar", "kind": "message", "message": "They said yes to: {{title}}. Put it on my calendar.", "when": "changed", "what": "ask", "only": []any{"state=yes"}})
	me.Automate.StartWorker()
	rang := make(chan string, 4)
	myH.(*server.Server).OnRing(func(title, text, _ string) { rang <- title + ": " + text })
	if me.Chat.FollowSent(context.Background()); len(rang) != 0 {
		t.Fatal("nothing is told while they have not answered")
	}
	wantStatus(t, postForm(t, plumberH, "/suggested/yes", url.Values{"id": {theirs[0].ID}}), http.StatusSeeOther)
	if err := me.Chat.FollowSent(context.Background()); err != nil {
		t.Fatal(err)
	}
	if got := <-rang; !strings.Contains(got, "plumber.example.ts.net said yes to your request to \"make an event") {
		t.Errorf("their yes is told, without asking again: %s", got)
	}
	if s, _ := me.Store.Get("ask", sent[0].ID); s.Fields["state"] != "yes" {
		t.Errorf("and kept: %v", s.Fields["state"])
	}
	for deadline := time.Now().Add(5 * time.Second); ; {
		if n, _ := me.Store.Count("event"); n == 1 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("an automation on asks lets the assistant act on the answer")
		}
		time.Sleep(20 * time.Millisecond)
	}

	text, isErr := me.Chat.Call("ask_page", json.RawMessage(`{"address":"`+link+`"}`))
	if isErr || !strings.Contains(strings.ReplaceAll(text, " ", ""), `"state":"yes"`) {
		t.Errorf("reading the link says what became of it: %v %s", isErr, text)
	}
	if strings.Contains(text, "Callouts") {
		t.Errorf("and nothing else: %s", text)
	}
}
