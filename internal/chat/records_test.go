package chat_test

import (
	"context"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// A person who asks for a note wants a note: something on /t/note, not a
// card on the canvas. The tools for that are generated from the workspace's
// schema, so the starter's note type is enough to prove all three.
func TestModelMakesAndChangesRecords(t *testing.T) {
	svc := newFullService(t)
	m := &scripted{steps: []*llm.Response{
		call("create_record", map[string]any{"type": "note", "fields": map[string]any{"title": "Call the dentist", "body": "Ask about Thursday.", "tags": []string{"health"}}}),
		call("find_records", map[string]any{"type": "note", "query": "dentist"}),
	}}
	svc.Provider = m
	if _, err := svc.Send(context.Background(), "make a note to call the dentist"); err != nil {
		t.Fatal(err)
	}

	notes, _ := svc.Store.List("note", store.ListOptions{})
	if len(notes) != 1 || notes[0].Fields["title"] != "Call the dentist" {
		t.Fatalf("expected one note record, got %+v", notes)
	}
	if notes[0].Fields["status"] != "draft" {
		t.Errorf("the schema's defaults apply, got status %v", notes[0].Fields["status"])
	}
	if blocks, _ := svc.Store.List(chat.BlockType, store.ListOptions{}); len(blocks) != 0 {
		t.Errorf("a note is not a canvas block, got %d blocks", len(blocks))
	}
	id := notes[0].ID
	created := lastToolResult(m.seen[1])
	if created.IsError || !strings.Contains(created.Content, "/t/note/"+id) {
		t.Errorf("the result should say where the person finds it: %+v", created)
	}
	found := lastToolResult(m.seen[2])
	if found.IsError || !strings.Contains(found.Content, id+"\tCall the dentist") {
		t.Errorf("find_records should list the id with its title: %+v", found)
	}
	activity, _ := svc.Store.List(chat.ActivityType, store.ListOptions{OrderBy: "created_at", Desc: true, Limit: 1})
	if len(activity) != 1 || activity[0].Fields["action"] != "created" || activity[0].Fields["target"] != "note" {
		t.Errorf("creating a record is a change the log records, got %+v", activity)
	}

	// Changing one field leaves the rest alone.
	m = &scripted{steps: []*llm.Response{
		call("update_record", map[string]any{"type": "note", "id": id, "fields": map[string]any{"status": "published"}}),
	}}
	svc.Provider = m
	svc.Send(context.Background(), "publish it")
	rec, _ := svc.Store.Get("note", id)
	if rec.Fields["status"] != "published" || rec.Fields["title"] != "Call the dentist" {
		t.Errorf("update should change only what was passed, got %+v", rec.Fields)
	}
}

// The schema is the contract: a bad field comes back as an error that says
// what to fix, and nothing is saved.
func TestRecordToolsRefuseWhatTheSchemaRefuses(t *testing.T) {
	svc := newFullService(t)
	m := &scripted{steps: []*llm.Response{
		call("create_record", map[string]any{"type": "note", "fields": map[string]any{"body": "no title", "colour": "red"}}),
		call("create_record", map[string]any{"type": "recipe", "fields": map[string]any{"title": "Soup"}}),
		call("update_record", map[string]any{"type": "note", "id": "nope", "fields": map[string]any{"title": "x"}}),
	}}
	svc.Provider = m
	svc.Send(context.Background(), "try some bad ones")
	checks := []struct {
		req  int
		want string
	}{
		{1, "title"},
		{1, "colour"},
		{2, "unknown content type"},
		{3, "find_records"},
	}
	for _, c := range checks {
		res := lastToolResult(m.seen[c.req])
		if !res.IsError || !strings.Contains(res.Content, c.want) {
			t.Errorf("call %d: want an error mentioning %q, got %+v", c.req, c.want, res)
		}
	}
	if n, _ := svc.Store.Count("note"); n != 0 {
		t.Errorf("nothing should have been saved, got %d notes", n)
	}
}

// The model can only make what it is told exists: the prompt carries the
// content types with their fields, and the tools are offered only when
// there is a type to write to.
func TestPromptAndToolsFollowTheSchema(t *testing.T) {
	svc := newFullService(t)
	m := &scripted{}
	svc.Provider = m
	svc.Send(context.Background(), "hi")
	system := m.seen[0].System
	for _, want := range []string{"Content types", "note: A short piece of writing.", `"title"`, "create_record", "a card on the canvas is not a note"} {
		if !strings.Contains(system, want) {
			t.Errorf("the prompt should carry %q", want)
		}
	}
	var names []string
	for _, tool := range m.seen[0].Tools {
		names = append(names, tool.Name)
	}
	joined := strings.Join(names, ",")
	for _, want := range []string{"create_record", "update_record", "find_records"} {
		if !strings.Contains(joined, want) {
			t.Errorf("tools offered should include %s: %v", want, names)
		}
	}

	// A workspace with only the system's own types offers no record tools.
	bare, _ := newService(t)
	m = &scripted{}
	bare.Provider = m
	bare.Send(context.Background(), "hi")
	for _, tool := range m.seen[0].Tools {
		if strings.HasSuffix(tool.Name, "_record") || tool.Name == "find_records" {
			t.Errorf("no content types, no record tools, got %s", tool.Name)
		}
	}
}
