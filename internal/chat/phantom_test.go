package chat_test

import (
	"context"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// TestPhantomRecordCreation verifies that when the assistant claims to have
// created a record at a specific URL, the record is actually retrievable from
// the store — i.e. it does not return 404. This covers the scenario where a
// model bundles text naming a phantom URL with a create_record tool call:
// after correction, the real record exists and VerifyClaims confirms no
// phantom URLs remain in the final reply.
func TestPhantomRecordCreation(t *testing.T) {
	ctx := context.Background()

	// Step 1: bundled response with phantom URL triggers correction.
	svc := newFullService(t)
	m := &scripted{steps: []*llm.Response{
		callWithTextAndID("I created the note 'Plan'. It's at /t/note/8k3m9p5n2j7x4q6v."),
		call("create_record", map[string]any{"type": "note", "fields": map[string]any{
			"title": "Plan",
		}}),
	}}
	svc.Provider = m

	reply, err := svc.Send(ctx, "make a note called Plan")
	if err != nil {
		t.Fatal(err)
	}

	// The tool actually ran: one note exists in the store.
	notes, _ := svc.Store.List("note", store.ListOptions{})
	if len(notes) != 1 {
		t.Fatalf("create_record should have persisted a note, got %d notes", len(notes))
	}

	realID := notes[0].ID

	// The record is retrievable at its real id — no 404.
	got, err := svc.Store.Get("note", realID)
	if err != nil {
		t.Fatalf("the created note should be retrievable at /t/note/%s: %v", realID, err)
	}
	if got.Fields["title"] != "Plan" {
		t.Errorf("title mismatch: want %q, got %q", "Plan", got.Fields["title"])
	}

	// VerifyClaims on the final reply finds no phantom URLs.
	finalText := reply.Fields["content"].(string)
	results, err := chat.VerifyClaims(ctx, svc.Store, finalText)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range results {
		if !r.Exists {
			t.Errorf("final reply should not name non-existent URL %s", r.URL)
		}
	}

	// The receipt (changes) must carry the creation with its actual id.
	changes, ok := reply.Fields["changes"].([]any)
	if !ok || len(changes) != 1 {
		t.Fatalf("expected one change in receipt, got %v", reply.Fields["changes"])
	}
	changeMap, ok := changes[0].(map[string]any)
	if !ok {
		t.Fatalf("change is not a map: %T", changes[0])
	}
	detail, _ := changeMap["detail"].(string)
	if detail == "" {
		t.Error("receipt entry missing 'detail' field")
	}
}
