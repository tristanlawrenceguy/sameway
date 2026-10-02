package chat_test

import (
	"context"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// The shapes Claude Haiku reached for in a real workspace: a record's
// fields beside type, inside a JSON string, under set; details asked by
// type. Each is taken as meant.
func TestLooseArgumentsAreTakenAsMeant(t *testing.T) {
	svc := newFullService(t)
	m := &scripted{steps: []*llm.Response{
		call("create_record", map[string]any{"type": "note", "title": "Beside"}),
		call("create_record", map[string]any{"type": "note", "set": `{"title":"In a string under set"}`}),
		call("create_record", map[string]any{"type": "note", "fields": `{"title":"Fields as a string"}`}),
		call("details", map[string]any{"type": "note"}),
	}}
	svc.Provider = m
	if _, err := svc.Send(context.Background(), "make three notes"); err != nil {
		t.Fatal(err)
	}
	notes, _ := svc.Store.List("note", store.ListOptions{})
	titles := map[string]bool{}
	for _, n := range notes {
		titles[n.Fields["title"].(string)] = true
	}
	for _, want := range []string{"Beside", "In a string under set", "Fields as a string"} {
		if !titles[want] {
			t.Errorf("no note %q; made %v", want, titles)
		}
	}
	if got := lastToolResult(m.seen[4]); got.IsError || !strings.Contains(got.Content, "content type note") {
		t.Errorf("details by type reads the type: %+v", got)
	}
}

// A model is told of recording a meeting when it makes an event, not in
// every prompt.
func TestAnEventSaysHowToRecordIt(t *testing.T) {
	svc := newFullService(t)
	m := &scripted{steps: []*llm.Response{
		call("create_record", map[string]any{"type": "event", "fields": map[string]any{"title": "Launch", "starts": "2026-10-05 10:00"}}),
	}}
	svc.Provider = m
	if _, err := svc.Send(context.Background(), "a meeting on Monday, record it"); err != nil {
		t.Fatal(err)
	}
	if got := lastToolResult(m.seen[1]); got.IsError || !strings.Contains(got.Content, "call the record_meeting tool yourself now with event") {
		t.Errorf("making an event says how to record it: %+v", got)
	}
}
