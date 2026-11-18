package server_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/records"
)

// What the owner says yes to is said as their pages say it: a choice by
// its label, a link by the name of what it names, how often in words, a
// list as said aloud, and words the asker wrote in quotes, never as a
// record stores them.
func TestARequestIsSaidAsThePagesSayIt(t *testing.T) {
	t.Parallel()
	a, _ := newApp(t)
	sam, _ := a.Store.Create("person", map[string]any{"name": "Sam", "email": "sam@example.com", "access": "view"})
	garden, _ := a.Store.Create("project", map[string]any{"title": "Garden"})
	them := a.Chat.For(records.Visitor{Person: sam.ID, Name: "Sam", Access: records.View})
	them.Call("create_record", json.RawMessage(`{"type":"task","fields":{"title":"Mend the gate","status":"doing","project":"`+garden.ID+`","tags":["outside","wood"],"repeat":"FREQ=WEEKLY","notes":"Use the long screws."}}`))
	asks := a.Chat.Suggestions()
	if len(asks) != 1 {
		t.Fatalf("one request: %d", len(asks))
	}
	got := asks[0].Fields["summary"].(string)
	want := `Sam asks you to make a task: "Mend the gate", notes "Use the long screws", project Garden, repeat every week, status Doing, tags outside and wood. Make it?`
	if got != want {
		t.Errorf("got\n%s\nwant\n%s", got, want)
	}
	if strings.Contains(got, garden.ID) {
		t.Error("a link is never an id")
	}
}
