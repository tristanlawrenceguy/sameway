package chat_test

import (
	"context"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// TestComponentValidationErrorsAreHumanised checks that when a component's
// props fail JSON Schema validation, the tool result text does not expose
// raw schema syntax ("additional properties", field names like 'tone'). The
// model then repeats this into chat; we assert on the tool result because
// that is what flows through. This covers acceptance item 1: error messages
// in chat must not contain internal field names or schema syntax.

func TestComponentValidationErrorsAreHumanised(t *testing.T) {
	svc := newFullService(t)
	m := &scripted{steps: []*llm.Response{
		call("add_component", map[string]any{"component": "button", "props": map[string]any{"label": "x", "bogus_prop": 1}}),
		call("update_component", map[string]any{"id": "does-not-exist"}),
	}}
	svc.Provider = m

	if _, err := svc.Send(context.Background(), "try bad props"); err != nil {
		t.Fatal(err)
	}

	res := lastToolResult(m.seen[1])
	if !res.IsError {
		t.Fatalf("expected an error for add_component with invalid props, got %+v", res)
	}

	errText := res.Content
	// The message must not contain schema-specific syntax.
	for _, bad := range []string{"additional properties", "not allowed"} {
		if strings.Contains(errText, bad) {
			t.Errorf("error should not expose schema syntax %q: %q", bad, errText)
		}
	}
	// The message must tell the user their edit was not saved.
	if !strings.Contains(strings.ToLower(errText), "save") &&
		!strings.Contains(strings.ToLower(errText), "change") &&
		!strings.Contains(strings.ToLower(errText), "recognised") {
		t.Errorf("error should inform the user their change did not save: %q", errText)
	}
}

func TestUpdateComponentValidationErrorsAreHumanised(t *testing.T) {
	svc := newFullService(t)
	m := &scripted{steps: []*llm.Response{
		call("add_component", map[string]any{"component": "heading", "props": map[string]any{"text": "Hello"}}),
	}}
	svc.Provider = m

	if _, err := svc.Send(context.Background(), "add heading"); err != nil {
		t.Fatal(err)
	}

	blocks, _ := svc.Store.List(chat.BlockType, store.ListOptions{})
	if len(blocks) == 0 {
		t.Fatalf("expected a block to exist for update test")
	}
	id := blocks[0].ID

	m = &scripted{steps: []*llm.Response{
		call("update_component", map[string]any{"id": id, "props": map[string]any{"bogus_field": true}}),
	}}
	svc.Provider = m
	if _, err := svc.Send(context.Background(), "update with bad props"); err != nil {
		t.Fatal(err)
	}

	res := lastToolResult(m.seen[1])
	if !res.IsError {
		t.Fatalf("expected an error for update_component with invalid props, got %+v", res)
	}

	errText := res.Content
	for _, bad := range []string{"additional properties", "not allowed"} {
		if strings.Contains(errText, bad) {
			t.Errorf("error should not expose schema syntax %q: %q", bad, errText)
		}
	}
	if !strings.Contains(strings.ToLower(errText), "save") &&
		!strings.Contains(strings.ToLower(errText), "change") &&
		!strings.Contains(strings.ToLower(errText), "recognised") {
		t.Errorf("error should inform the user their change did not save: %q", errText)
	}

	// The original block must be unchanged.
	rec, _ := svc.Store.Get(chat.BlockType, id)
	if props, ok := rec.Fields["props"].(map[string]any); !ok || props["bogus_field"] != nil {
		t.Errorf("block should not have been updated with invalid props: %+v", rec.Fields)
	}
}

func TestRecordValidationErrorsAreHumanised(t *testing.T) {
	svc := newFullService(t)
	m := &scripted{steps: []*llm.Response{
		call("create_record", map[string]any{"type": "note", "fields": map[string]any{"body": "no title"}}),
	}}
	svc.Provider = m

	if _, err := svc.Send(context.Background(), "make a note with missing fields"); err != nil {
		t.Fatal(err)
	}

	res := lastToolResult(m.seen[1])
	if !res.IsError {
		t.Fatalf("expected an error for create_record with invalid fields, got %+v", res)
	}

	errText := res.Content
	for _, bad := range []string{"additional properties", "not allowed"} {
		if strings.Contains(errText, bad) {
			t.Errorf("error should not expose schema syntax %q: %q", bad, errText)
		}
	}
	if !strings.Contains(strings.ToLower(errText), "save") &&
		!strings.Contains(strings.ToLower(errText), "change") &&
		!strings.Contains(strings.ToLower(errText), "recognised") {
		t.Errorf("error should inform the user their change did not save: %q", errText)
	}

	if n, _ := svc.Store.Count("note"); n != 0 {
		t.Errorf("nothing should have been saved, got %d notes", n)
	}
}
