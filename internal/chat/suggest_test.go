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
		map[string]any{"passage": "it was decided by everyone that", "replacement": "we decided", "why": "Shorter."},
		map[string]any{"passage": "recieve", "replacement": "receive", "why": "Spelling."},
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

	if _, _, err := chat.AcceptSuggestion(svc.Store, chat.Who{Actor: "human"}, waiting[1].ID); err != nil {
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
	if _, _, err := chat.AcceptSuggestion(svc.Store, chat.Who{Actor: "human"}, waiting[0].ID); err != chat.ErrOutdated {
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
		map[string]any{"passage": "dog", "replacement": "cat", "why": "x"}}}, "copy it exactly")
	refused(t, svc, "suggest_edits", map[string]any{"type": "note", "id": note.ID, "edits": []any{
		map[string]any{"passage": "the", "replacement": "a", "why": "x"}}}, "occurs 2 times")
	refused(t, svc, "suggest_edits", map[string]any{"type": "note", "id": note.ID, "edits": []any{
		map[string]any{"passage": "cat sat", "replacement": "dog sat", "why": "x"},
		map[string]any{"passage": "sat on", "replacement": "lay on", "why": "y"}}}, "overlaps")
	if n := len(chat.Suggestions(svc.Store, "note", note.ID)); n != 0 {
		t.Errorf("nothing suggested when refused: %d", n)
	}
}
