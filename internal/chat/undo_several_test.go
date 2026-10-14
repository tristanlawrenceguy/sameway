package chat_test

import (
	"context"
	"regexp"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// The next turn's prompt names every change of the last reply, and
// undo_change with those ids takes them all back in one call.
func TestALastReplyIsUndoneInOneCall(t *testing.T) {
	t.Parallel()
	svc := newFullService(t)
	svc.Provider = &scripted{steps: []*llm.Response{
		call("create_record", map[string]any{"type": "note", "fields": map[string]any{"title": "Seeds"}}),
		call("create_record", map[string]any{"type": "note", "fields": map[string]any{"title": "Beds"}}),
		call("create_record", map[string]any{"type": "note", "fields": map[string]any{"title": "Tools"}}),
		{Text: "Made three notes."},
	}}
	if _, err := svc.Send(context.Background(), "three garden notes"); err != nil {
		t.Fatal(err)
	}
	m := &scripted{steps: []*llm.Response{{Text: "ok"}}}
	svc.Provider = m
	if _, err := svc.Send(context.Background(), "undo all that"); err != nil {
		t.Fatal(err)
	}
	line := regexp.MustCompile(`Your last reply made 3 changes; undo_change with ids ([a-z0-9, ]+) takes them all back`).FindStringSubmatch(m.seen[0].System)
	if line == nil {
		t.Fatalf("the prompt names the last reply's three changes:\n%s", tail(m.seen[0].System))
	}
	ids := strings.Split(line[1], ", ")
	text := run(t, svc, "undo_change", map[string]any{"ids": ids})
	if !strings.HasPrefix(text, "undid all 3") {
		t.Errorf("all three undone in one call: %s", text)
	}
	if notes, _ := svc.Store.List("note", store.ListOptions{}); len(notes) != 0 {
		t.Errorf("no note is left, %d are", len(notes))
	}
}

// Several undone in one call are all or none, as the page's Undo all is:
// one that cannot be leaves every one done, and is named.
func TestUndoChangeWithIdsIsAllOrNone(t *testing.T) {
	t.Parallel()
	svc := newFullService(t)
	run(t, svc, "create_record", map[string]any{"type": "note", "fields": map[string]any{"title": "Seeds"}})
	run(t, svc, "create_record", map[string]any{"type": "note", "fields": map[string]any{"title": "Beds"}})
	made := entries(t, svc, "created")
	notes, _ := svc.Store.List("note", store.ListOptions{OrderBy: "created_at"})
	svc.Store.Delete("note", notes[0].ID) // the older one went since
	text, isErr := svc.Call("undo_change", []byte(`{"ids":["`+made[1].ID+`","`+made[0].ID+`"]}`))
	if !isErr || !strings.Contains(text, "nothing was undone: the second of 2") {
		t.Errorf("refused, naming the one: %v %s", isErr, text)
	}
	if left, _ := svc.Store.List("note", store.ListOptions{}); len(left) != 1 {
		t.Errorf("the newer note is put back, %d left", len(left))
	}
}
