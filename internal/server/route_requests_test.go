package server_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/app"
	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/records"
)

// routing answers who handles a request as the owner's words say it.
type routing struct{ scripted }

func (r *routing) Complete(ctx context.Context, req llm.Request) (*llm.Response, error) {
	if strings.Contains(req.System, "who handles a request") {
		_, asked, _ := strings.Cut(req.Messages[0].Content, "The request:")
		if strings.Contains(asked, "kitchen") {
			return &llm.Response{Text: "Sam"}, nil
		}
		return &llm.Response{Text: "owner"}, nil
	}
	return r.scripted.Complete(ctx, req)
}

// A request is passed to whoever the owner's words say handles it, among
// those let in to change things: they are told in their own conversation
// and answer it there, as the owner would; nobody else let in can, and
// one the words give to nobody stays the owner's.
func TestARequestGoesToWhoeverHandlesIt(t *testing.T) {
	t.Parallel()
	a, _ := newAppAt(t, tonight)
	if err := a.Workspace.Set("requests.route", "Anything about the kitchen goes to Sam"); err != nil {
		t.Fatal(err)
	}
	a.Workspace.Config.Publish.Book = "on"
	a.Chat.Provider = &routing{}
	sam, _ := a.Store.Create("person", map[string]any{"name": "Sam", "email": "sam@example.com", "access": "edit"})
	ana, _ := a.Store.Create("person", map[string]any{"name": "Ana", "email": "ana@example.com", "access": "edit"})
	samChat := a.Chat.For(records.Visitor{Person: sam.ID, Name: "Sam", Access: records.Edit})
	anaChat := a.Chat.For(records.Visitor{Person: ana.ID, Name: "Ana", Access: records.Edit})

	token, err := a.Chat.Booking("Pat", "pat@example.com", "About fitting the kitchen", thursdayAt(9))
	if err != nil {
		t.Fatal(err)
	}
	p := a.Chat.RequestByToken(token)
	if p.Fields["for"] != sam.ID {
		t.Fatalf("it is passed to Sam: %v", p.Fields["for"])
	}
	msgs, _ := samChat.Messages()
	if len(msgs) == 0 || !strings.Contains(msgs[len(msgs)-1].Fields["content"].(string), "A request for you to answer") {
		t.Fatalf("and Sam is told, in their own conversation: %+v", msgs)
	}
	if text, isErr := anaChat.Call("answer_request", json.RawMessage(`{"id":"`+p.ID+`","yes":true}`)); !isErr || !strings.Contains(text, "not passed to them") {
		t.Errorf("nobody else let in answers it: %s", text)
	}
	if text, isErr := samChat.Call("answer_request", json.RawMessage(`{"id":"`+p.ID+`","yes":true}`)); isErr {
		t.Fatalf("Sam answers it: %s", text)
	}
	if n, _ := a.Store.Count("event"); n != 1 || chatState(a, token) != "yes" {
		t.Errorf("it is made as the owner would, and whoever asked hears so: %d %s", n, chatState(a, token))
	}

	other, _ := a.Chat.Booking("Lee", "lee@example.com", "About the garden", thursdayAt(11).Add(24*time.Hour))
	if q := a.Chat.RequestByToken(other); q.Fields["for"] != nil && q.Fields["for"] != "" {
		t.Errorf("one the words give to nobody stays the owner's: %v", q.Fields["for"])
	}
}

func chatState(a *app.App, token string) string {
	return chat.RequestState(a.Chat.RequestByToken(token))
}
