package chat_test

import (
	"context"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// A model that says it made something, naming a page that does not exist,
// is told so and asked again; the reply the person sees is the one that
// really made it. A reply naming a real page is left alone.
func TestAClaimedPageThatDoesNotExistIsAskedAgain(t *testing.T) {
	svc := newFullService(t)
	m := &scripted{steps: []*llm.Response{
		{Text: "I created the note 'Plan'. It's available at /t/note/8k3m9p5n2j7x4q6v."},
		call("create_record", map[string]any{"type": "note", "fields": map[string]any{"title": "Plan"}}),
		{Text: "Done, the note is at its page."},
	}}
	svc.Provider = m
	reply, err := svc.Send(context.Background(), "make a note called Plan")
	if err != nil {
		t.Fatal(err)
	}
	if len(m.seen) != 3 {
		t.Fatalf("the model should be corrected once and then run its tool, got %d calls", len(m.seen))
	}
	correction := m.seen[1].Messages[len(m.seen[1].Messages)-1]
	if correction.Role != llm.RoleUser || !strings.Contains(correction.Content, "no page at /t/note/8k3m9p5n2j7x4q6v") || !strings.Contains(correction.Content, "tools") {
		t.Errorf("the correction names the page and asks for the tools: %+v", correction)
	}
	notes, _ := svc.Store.List("note", store.ListOptions{})
	if len(notes) != 1 || reply.Fields["content"] != "Done, the note is at its page." {
		t.Errorf("the note is made and the honest reply is the one kept: %d notes, %v", len(notes), reply.Fields["content"])
	}
	if changes, _ := reply.Fields["changes"].([]any); len(changes) != 1 {
		t.Errorf("the receipt carries the creation, got %v", reply.Fields["changes"])
	}

	// A reply naming the page that now exists is not questioned, and a
	// model that insists is asked only once.
	real := "/t/note/" + notes[0].ID
	m = &scripted{steps: []*llm.Response{{Text: "It is at " + real + "."}}}
	svc.Provider = m
	svc.Send(context.Background(), "where is it?")
	if len(m.seen) != 1 {
		t.Errorf("a real page needs no correction, got %d calls", len(m.seen))
	}
	m = &scripted{steps: []*llm.Response{{Text: "See /t/note/zzzzzzzzzzzzzzzz."}, {Text: "See /t/note/zzzzzzzzzzzzzzzz, really."}}}
	svc.Provider = m
	reply, _ = svc.Send(context.Background(), "and the other one?")
	if len(m.seen) != 2 || !strings.Contains(reply.Fields["content"].(string), "really") {
		t.Errorf("one correction, then the reply stands as the model's own: %d calls, %v", len(m.seen), reply.Fields["content"])
	}
}

// A model that names several non-existent pages gets a single correction
// listing all of them at once, not one page per round. VerifyClaims returns
// every URL the reply mentions so the handler can build one correction.
func TestMultipleClaimedPagesAreListedInOneCorrection(t *testing.T) {
	svc := newFullService(t)
	m := &scripted{steps: []*llm.Response{
		{Text: "I created a note at /t/note/8k3m9p5n2j7x4q6v, a card at /t/block/1a2b3c4d5e6f7g8h and a recipe at /t/recipe/abcdef0123456789."},
		call("create_record", map[string]any{"type": "note", "fields": map[string]any{"title": "Three pages"}}),
		{Text: "Done, all three are created."},
	}}
	svc.Provider = m
	reply, err := svc.Send(context.Background(), "make a note, a card and a recipe")
	if err != nil {
		t.Fatal(err)
	}
	if len(m.seen) < 2 {
		t.Fatalf("the model should be corrected once then run its tool, got %d calls", len(m.seen))
	}
	correction := m.seen[1].Messages[len(m.seen[1].Messages)-1]
	if correction.Role != llm.RoleUser {
		t.Errorf("correction must come from the user role: %+v", correction)
	}
	content := correction.Content
	for _, page := range []string{"/t/note/8k3m9p5n2j7x4q6v", "/t/block/1a2b3c4d5e6f7g8h", "/t/recipe/abcdef0123456789"} {
		if !strings.Contains(content, page) {
			t.Errorf("correction should mention %s, got: %s", page, content)
		}
	}
	if reply.Fields["content"] != "Done, all three are created." {
		t.Errorf("the honest final reply is kept: got %q", reply.Fields["content"])
	}
}
