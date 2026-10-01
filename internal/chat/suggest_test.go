package chat_test

import (
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// Asked to improve someone's words, the assistant suggests: each change
// waits until accepted, which is an ordinary change undone like any other;
// a passage that has changed since is not guessed at.
func TestSuggestedEditsWaitForTheWriter(t *testing.T) {
	svc := newFullService(t)
	note, err := svc.Store.Create("note", map[string]any{"title": "Garden", "body": "At the meeting it was decided by everyone that the garden opens in May. We recieve seeds in April."})
	if err != nil {
		t.Fatal(err)
	}
	out := run(t, svc, "suggest_edits", map[string]any{"type": "note", "id": note.ID, "edits": []any{
		map[string]any{"passage": "it was decided by everyone that", "replacement": "we decided", "why": "Shorter.", "kind": "clarity"},
		map[string]any{"passage": "recieve", "replacement": "receive", "why": "Spelling.", "kind": "fix"},
	}})
	if !strings.Contains(out, "2 changes") || !strings.Contains(out, "/t/note/"+note.ID) {
		t.Errorf("it says what waits and where: %s", out)
	}
	if now, _ := svc.Store.Get("note", note.ID); now.Fields["body"] != note.Fields["body"] {
		t.Fatal("suggesting changes nothing")
	}
	waiting := chat.Suggestions(svc.Store, "note", note.ID)
	if len(waiting) != 2 {
		t.Fatalf("both wait on the note, got %d", len(waiting))
	}

	if _, _, _, _, err := chat.AcceptSuggestions(svc.Store, chat.Who{Actor: "human"}, []string{waiting[1].ID}); err != nil {
		t.Fatal(err)
	}
	now, _ := svc.Store.Get("note", note.ID)
	if b := now.Fields["body"].(string); !strings.Contains(b, "We receive seeds") || !strings.Contains(b, "decided by everyone") {
		t.Errorf("accepting one makes that one alone: %q", b)
	}
	if len(entries(t, svc, "updated")) != 1 {
		t.Error("accepting is logged as the person's change, to be undone like any other")
	}

	// The writer changes the sentence; the other suggestion no longer fits.
	svc.Store.Update("note", note.ID, map[string]any{"body": "We all agreed: the garden opens in May. We receive seeds in April."})
	if _, _, _, _, err := chat.AcceptSuggestions(svc.Store, chat.Who{Actor: "human"}, []string{waiting[0].ID}); err != chat.ErrOutdated {
		t.Errorf("an outdated suggestion is not guessed at: %v", err)
	}
	if left := chat.Suggestions(svc.Store, "note", note.ID); len(left) != 0 {
		t.Errorf("answered, nothing waits: %d", len(left))
	}
	all, _ := svc.Store.List(chat.SuggestionType, store.ListOptions{})
	if len(all) != 2 {
		t.Errorf("they are kept, answered: %d", len(all))
	}
}

// A passage not there, there twice, or overlapping another is refused
// with what to do, and nothing is suggested.
func TestASuggestionMustFindItsWords(t *testing.T) {
	svc := newFullService(t)
	note, _ := svc.Store.Create("note", map[string]any{"title": "N", "body": "the cat sat on the mat"})
	refused(t, svc, "suggest_edits", map[string]any{"type": "note", "id": note.ID, "edits": []any{
		map[string]any{"passage": "dog", "replacement": "cat", "why": "x", "kind": "fix"}}}, "copy it exactly")
	refused(t, svc, "suggest_edits", map[string]any{"type": "note", "id": note.ID, "edits": []any{
		map[string]any{"passage": "the", "replacement": "a", "why": "x", "kind": "fix"}}}, "occurs 2 times")
	refused(t, svc, "suggest_edits", map[string]any{"type": "note", "id": note.ID, "edits": []any{
		map[string]any{"passage": "cat sat", "replacement": "dog sat", "why": "x", "kind": "fix"},
		map[string]any{"passage": "sat on", "replacement": "lay on", "why": "y", "kind": "fix"}}}, "overlaps")
	if n := len(chat.Suggestions(svc.Store, "note", note.ID)); n != 0 {
		t.Errorf("nothing suggested when refused: %d", n)
	}
}

// A change to the formatting alone is found as one, whatever the model
// called it; several accepted together are one change and one Undo; and
// more than a writer can weigh in one go is refused.
func TestSuggestionsAreKindedAndTakenTogether(t *testing.T) {
	svc := newFullService(t)
	note, _ := svc.Store.Create("note", map[string]any{"title": "N", "body": "Plans\n\nWe recieve seeds and teh pond liner."})
	run(t, svc, "suggest_edits", map[string]any{"type": "note", "id": note.ID, "edits": []any{
		map[string]any{"passage": "Plans", "replacement": "## Plans", "why": "A heading.", "kind": "style"},
		map[string]any{"passage": "recieve", "replacement": "receive", "why": "Spelling.", "kind": "fix"},
		map[string]any{"passage": "teh", "replacement": "the", "why": "Typo.", "kind": "fix"},
	}})
	waiting := chat.Suggestions(svc.Store, "note", note.ID)
	if waiting[0].Fields["kind"] != "format" || waiting[1].Fields["kind"] != "fix" {
		t.Errorf("formatting is found by Sameway: %v, %v", waiting[0].Fields["kind"], waiting[1].Fields["kind"])
	}
	_, _, made, _, err := chat.AcceptSuggestions(svc.Store, chat.Who{Actor: "human"}, []string{waiting[1].ID, waiting[2].ID})
	if err != nil || made != 2 {
		t.Fatalf("both fixes go in: %d %v", made, err)
	}
	now, _ := svc.Store.Get("note", note.ID)
	if now.Fields["body"] != "Plans\n\nWe receive seeds and the pond liner." {
		t.Errorf("both fixes in one change: %q", now.Fields["body"])
	}
	updated := entries(t, svc, "updated")
	if len(updated) != 1 {
		t.Fatalf("one change, one entry: %d", len(updated))
	}
	run(t, svc, "undo_change", map[string]any{"id": updated[0].ID})
	if back, _ := svc.Store.Get("note", note.ID); back.Fields["body"] != note.Fields["body"] {
		t.Errorf("one Undo takes both back: %q", back.Fields["body"])
	}
	many := []any{}
	for i := 0; i < 16; i++ {
		many = append(many, map[string]any{"passage": "Plans", "replacement": "Plan", "why": "x", "kind": "fix"})
	}
	refused(t, svc, "suggest_edits", map[string]any{"type": "note", "id": note.ID, "edits": many}, "matter most")
}

// A suggestion card is Sameway's to place on the page it belongs to: it
// is refused as a block (and left out of the catalogue: prompt_budget_test).
func TestASuggestionIsNotABlock(t *testing.T) {
	svc := newFullService(t)
	refused(t, svc, "add_component", map[string]any{"component": "suggestion", "props": map[string]any{
		"label": "1 of 1", "why": "x", "now": "a", "nowMark": "a", "accept": "/a", "decline": "/d"}}, "not a block")
}
