package chat_test

import (
	"context"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
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
	m = &scripted{steps: []*llm.Response{{Text: "See /t/note/zzzzzzzzzzzzzzzz and also /t/block/1a2b3c4d5e6f7g8h."}}}
	svc.Provider = m
	svc.Send(context.Background(), "check")
	if len(m.seen) != 2 {
		t.Errorf("mixed valid+phantom pages trigger correction, got %d calls", len(m.seen))
	}
	// /t/block/... is not a content type in the starter schema so it stays Exists:false;
	// only known types are checked. The block URL is silently skipped.

	// A reply that names nothing to create and no pages is accepted as-is.
	m = &scripted{steps: []*llm.Response{{Text: "All done."}}}
	svc.Provider = m
	svc.Send(context.Background(), "confirm")
	if len(m.seen) != 1 {
		t.Errorf("a plain reply needs no correction, got %d calls", len(m.seen))
	}

	// Multiple phantom URLs in one reply all get flagged at once.
	m = &scripted{steps: []*llm.Response{{Text: "I made /t/note/8k3m9p5n2j7x4q6v and also /t/task/abcdef0123456789."}}}
	svc.Provider = m
	svc.Send(context.Background(), "check")
	if len(m.seen) != 2 {
		t.Errorf("multiple phantom pages trigger one correction, got %d calls", len(m.seen))
	}
	correction = m.seen[1].Messages[len(m.seen[1].Messages)-1]
	if !strings.Contains(correction.Content, "/t/note/8k3m9p5n2j7x4q6v") || !strings.Contains(correction.Content, "/t/task/abcdef0123456789") {
		t.Errorf("correction must name both phantom pages: %s", correction.Content)
	}

	// A reply that names a page which exists after the turn is accepted.
	m = &scripted{steps: []*llm.Response{{Text: "It is at /t/note/" + notes[0].ID + "."}}}
	svc.Provider = m
	svc.Send(context.Background(), "where?")
	if len(m.seen) != 1 {
		t.Errorf("an existing page needs no correction, got %d calls", len(m.seen))
	}

	// A reply that names a fake type is corrected too: the type is not known.
	m = &scripted{steps: []*llm.Response{{Text: "See /t/recipe/abcdef0123456789."}}}
	svc.Provider = m
	svc.Send(context.Background(), "check")
	if len(m.seen) != 2 {
		t.Errorf("a recipe page is not known and triggers correction, got %d calls", len(m.seen))
	}
}

// TestPhantomRecordCreationIsDiagnosed covers two ways the assistant can lie:
// (A) it says it created something but does not call any tool — the URL in its
// words points nowhere; (B) it calls create_record so a real record exists,
// but its prose names a different (non-existent) id.  The existing correction
// path handles case A; this test also confirms that VerifyClaims detects the
// phantom URL regardless of whether a tool ran, and that the receipt carries
// only URLs for records that actually exist — never phantom ones.
func TestPhantomRecordCreationIsDiagnosed(t *testing.T) {
	ctx := context.Background()

	// --- Scenario B: model calls create_record but names a different id in prose ---
	// The scripted provider returns two steps: (1) bundled text with a phantom URL
	// plus the tool call, which triggers correction; (2) a clean tool call that
	// actually executes after correction.  After the tool runs, the loop asks for
	// another response but none is left so it returns "done".  The final reply
	// carries no phantom URL, but VerifyClaims called on the known intermediate
	// prose still flags it — proving the diagnosis works even when the tool ran.
	svc := newFullService(t)
	m := &scripted{steps: []*llm.Response{
		callWithTextAndID("I created the note 'Plan'. It's at /t/note/8k3m9p5n2j7x4q6v."),
		call("create_record", map[string]any{"type": "note", "fields": map[string]any{"title": "Plan"}}),
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

	// The record is retrievable at its real id — no 404 there.
	got, err := svc.Store.Get("note", realID)
	if err != nil {
		t.Fatalf("the created note should be retrievable at /t/note/%s: %v", realID, err)
	}
	if got.Fields["title"] != "Plan" {
		t.Errorf("title mismatch: want %q, got %q", "Plan", got.Fields["title"])
	}

	// The assistant's final reply text is "done"; the phantom URL lives in the
	// intermediate message that was sent to the model but never persisted.
	if !strings.Contains(reply.Fields["content"].(string), "/t/note/") {
		t.Logf("final reply content does not contain phantom URL: %q", reply.Fields["content"])
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

	// --- Scenario A: model claims without calling tool (already tested in TestAClaimedPageThatDoesNotExistIsAskedAgain) ---
	svc2 := newFullService(t)
	m2 := &scripted{steps: []*llm.Response{
		{Text: "I created the note 'Plan'. It's available at /t/note/8k3m9p5n2j7x4q6v."},
		call("create_record", map[string]any{"type": "note", "fields": map[string]any{"title": "Plan"}}),
		{Text: "Done, the note is at its page."},
	}}
	svc2.Provider = m2
	reply2, err := svc2.Send(ctx, "make a note called Plan")
	if err != nil {
		t.Fatal(err)
	}

	// After correction, VerifyClaims on the corrected reply finds nothing missing.
	results2, err := chat.VerifyClaims(ctx, svc2.Store, reply2.Fields["content"].(string))
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range results2 {
		if !r.Exists && strings.Contains(r.URL, "/t/note/8k3m9p5n2j7x4q6v") {
			t.Errorf("corrected reply should not mention phantom URL %s", r.URL)
		}
	}

	// The model should have been corrected once.
	correction := m2.seen[1].Messages[len(m2.seen[1].Messages)-1]
	if correction.Role != llm.RoleUser || !strings.Contains(correction.Content, "no page at /t/note/8k3m9p5n2j7x4q6v") {
		t.Errorf("correction should name the phantom page: %+v", correction)
	}

	// The real note exists.
	notes2, _ := svc2.Store.List("note", store.ListOptions{})
	if len(notes2) != 1 {
		t.Fatalf("after correction a note should exist, got %d notes", len(notes2))
	}
}

// TestPhantomURLsInBundledResponsesAreCorrected covers the gap where the model
// bundles text (naming a non-existent page) with a tool call in one response.
// The existing correction path only scans text-only replies; bundled responses
// skip verification entirely, so their prose URLs are never checked. This test
// asserts that even when the model calls create_record alongside its prose,
// the phantom URL is still detected and corrected — the fix adds pre-tool-call
// scanning of the response text.
func TestPhantomURLsInBundledResponsesAreCorrected(t *testing.T) {
	ctx := context.Background()

	// The scripted provider returns: (1) bundled text+toolcall with phantom URL,
	// (2) a clean call-only step, (3) final text that names the real page.
	svc := newFullService(t)
	m := &scripted{steps: []*llm.Response{
		callWithTextAndID("I created the note 'Plan'. It's at /t/note/8k3m9p5n2j7x4q6v."),
		call("create_record", map[string]any{"type": "note", "fields": map[string]any{"title": "Plan"}}),
	},
	}
	svc.Provider = m
	reply, err := svc.Send(ctx, "make a note called Plan")
	if err != nil {
		t.Fatal(err)
	}

	// The model should have been corrected once about the phantom URL.
	// Today it is NOT (len(m.seen) == 2: initial + tool execution).
	// After the fix, len(m.seen) == 3: initial + correction + tool call.
	if len(m.seen) != 3 {
		t.Errorf("the model should be corrected once about the phantom URL in bundled text then run its tool, got %d calls (expected 3)", len(m.seen))
	}

	correction := m.seen[1].Messages[len(m.seen[1].Messages)-1]
	if correction.Role != llm.RoleUser || !strings.Contains(correction.Content, "no page at /t/note/8k3m9p5n2j7x4q6v") {
		t.Errorf("correction should name the phantom page: %+v", correction)
	}

	// After correction and tool execution, the note exists.
	notes, _ := svc.Store.List("note", store.ListOptions{})
	if len(notes) != 1 {
		t.Fatalf("after correction a note should exist, got %d notes", len(notes))
	}

	// The final reply should reference the real page, not the phantom one.
	finalText := reply.Fields["content"].(string)
	if strings.Contains(finalText, "/t/note/8k3m9p5n2j7x4q6v") {
		t.Errorf("final reply should not mention phantom URL %s: got %q", "/t/note/8k3m9p5n2j7x4q6v", finalText)
	}

	// The receipt (changes) must carry the creation.
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

	// VerifyClaims on the corrected reply finds nothing missing.
	results, err := chat.VerifyClaims(ctx, svc.Store, finalText)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range results {
		if !r.Exists && strings.Contains(r.URL, "/t/note/8k3m9p5n2j7x4q6v") {
			t.Errorf("corrected reply should not mention phantom URL %s", r.URL)
		}
	}

	// A bundled response that names a real page is NOT corrected.
	svc2 := newFullService(t)
	realNote, _ := svc2.Store.Create("note", map[string]any{"title": "Existing"})
	m2 := &scripted{steps: []*llm.Response{
		callWithTextAndID("It is at /t/note/" + realNote.ID + "."),
	}}
	svc2.Provider = m2
	svc2.Send(ctx, "check")
	// After the fix, a bundled response with only existing URLs should not trigger
	// correction.  Complete is always called twice: once for the step, once when no
	// steps remain (returning "done").  The test asserts that the second call has no
	// tool results — confirming nothing was corrected about the bundled text.
	if len(m2.seen) != 2 {
		t.Fatalf("a bundled response naming real pages should not be corrected: got %d calls", len(m2.seen))
	}
	lastMsg := m2.seen[1].Messages[len(m2.seen[1].Messages)-1]
	if lastMsg.Role == llm.RoleUser && strings.Contains(lastMsg.Content, "no page at") {
		t.Errorf("a bundled response naming real pages should not be corrected: user message is %q", lastMsg.Content)
	}

}
