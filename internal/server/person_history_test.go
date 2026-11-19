package server_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/records"
)

// Everything with one person is one question: the emails with them,
// what they asked for and the answer, and the events they are in with
// their answer to its invitation, oldest first.
func TestTheHistoryWithSomeoneIsOneQuestion(t *testing.T) {
	t.Parallel()
	a, _ := newAppAt(t, tonight)
	sam, _ := a.Store.Create("person", map[string]any{"name": "Sam", "email": "sam@example.com", "access": "view"})
	first, _ := a.Store.Create("email", map[string]any{"subject": "Kitchen", "from": "Sam <sam@example.com>", "body": "When does phase 2 start?", "message_id": "<s1@example.com>", "from_me": false})
	a.Store.Create("email", map[string]any{"subject": "Re: Kitchen", "from": "tris@example.com", "body": "Monday.", "message_id": "<t1@example.com>", "thread": first.ID, "from_me": true, "turn": "theirs"})
	a.Store.Create("email", map[string]any{"subject": "Something else", "from": "Ana <ana@example.com>", "body": "Hi", "message_id": "<a1@example.com>", "from_me": false})
	a.Chat.For(records.Visitor{Person: sam.ID, Name: "Sam", Access: "view"}).Call("create_record", json.RawMessage(`{"type":"note","fields":{"title":"Tiles for the kitchen"}}`))
	a.Store.Create("event", map[string]any{"title": "Kitchen walk-through", "starts": "2026-03-17T15:00:00Z", "answers": []any{"Sam <sam@example.com>: accepted"}})

	text, isErr := a.Chat.Call("history", json.RawMessage(`{"who":"Sam"}`))
	if isErr {
		t.Fatal(text)
	}
	for _, want := range []string{`They wrote: "Kitchen"`, `You wrote: "Re: Kitchen"`, "theirs", `They asked to make a note: "Tiles for the kitchen": it waits for you`, `Event: "Kitchen walk-through"`, "their answer: accepted"} {
		if !strings.Contains(text, want) {
			t.Errorf("the history has %q:\n%s", want, text)
		}
	}
	if strings.Contains(text, "Something else") {
		t.Errorf("and nothing with anyone else:\n%s", text)
	}
	if text, isErr := a.Chat.For(records.Visitor{Person: sam.ID, Name: "Sam", Access: "view"}).Call("history", json.RawMessage(`{"who":"Sam"}`)); !isErr {
		t.Errorf("it is the owner's to ask: %s", text)
	}
}
