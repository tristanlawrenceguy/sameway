package chat_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/records"
)

// typeLine is the index's line for one content type in a system prompt.
func typeLine(system, name string) string {
	for _, line := range strings.Split(system, "\n") {
		if strings.HasPrefix(line, name+": ") {
			return line
		}
	}
	return ""
}

// The model is told how much each type holds, and who made an empty
// one, so it can notice the kinds nobody uses and offer to hide them.
func TestTheTypeIndexSaysWhatEachTypeHolds(t *testing.T) {
	t.Parallel()
	svc := newFullService(t)
	for _, title := range []string{"Water the plants", "Call the bank", "Book a dentist"} {
		raw, _ := json.Marshal(map[string]any{"type": "note", "fields": map[string]any{"title": title}})
		if text, isErr := svc.Call("create_record", raw); isErr {
			t.Fatal(text)
		}
	}
	raw, _ := json.Marshal(map[string]any{"type": "task", "fields": map[string]any{"title": "Pay rent"}})
	if text, isErr := svc.Call("create_record", raw); isErr {
		t.Fatal(text)
	}
	records.Record(svc.Store, records.ActorAgent, records.Change{Action: "added", Component: "type", Detail: "habit", By: "crew-tester"})

	m := &scripted{}
	svc.Provider = m
	svc.Send(context.Background(), "hi")
	system := m.seen[0].System
	if line := typeLine(system, "note"); !strings.HasSuffix(line, " 3 records.") {
		t.Errorf("a type in use says its count: %q", line)
	}
	if line := typeLine(system, "task"); !strings.Contains(line, " 1 record, last changed ") {
		t.Errorf("a type with nearly nothing says when it was last written: %q", line)
	}
	if line := typeLine(system, "project"); !strings.HasSuffix(line, " No records.") {
		t.Errorf("an empty type says so: %q", line)
	}
	if line := typeLine(system, "habit"); !strings.HasSuffix(line, " No records, made by an agent, crew-tester.") {
		t.Errorf("an empty type an agent made says who: %q", line)
	}
	if !strings.Contains(system, "never hide one without their yes") {
		t.Error("the index says an empty type is offered for hiding, never hidden unasked")
	}
}
