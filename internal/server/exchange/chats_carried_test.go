package exchange_test

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// findsToDo suggests the first thing still to do in a conversation that
// it was not told it suggested before.
type findsToDo struct {
	mu    sync.Mutex
	asked int
}

func (m *findsToDo) Name() string { return "finds to do" }
func (m *findsToDo) Complete(_ context.Context, req llm.Request) (*llm.Response, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.asked++
	record, before, _ := strings.Cut(req.Messages[0].Content, "What you suggested from this record before")
	for _, todo := range []string{"Book the van", "Call the plumber"} {
		if strings.Contains(record, todo) && !strings.Contains(before, todo) {
			return &llm.Response{Text: `{"make": true, "fields": {"title": "` + todo + `"}, "why": "said in the chat"}`}, nil
		}
	}
	return &llm.Response{Text: `{"make": false}`}, nil
}

// A conversation carried on since it was brought is looked through again
// when conversations are, once: what its new words ask is suggested, and
// what its start asked is not suggested again. Never asked, nothing is.
func TestAConversationCarriedOnIsLookedThroughAgain(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)
	model := &findsToDo{}
	a.Chat.Provider = model
	a.Automate.StartWorker()
	suggested := func(n int) []string {
		var out []string
		for end := time.Now().Add(10 * time.Second); len(out) < n && time.Now().Before(end); time.Sleep(20 * time.Millisecond) {
			ps, _ := a.Store.List(records.ProposalType, store.ListOptions{OrderBy: "created_at"})
			out = out[:0]
			for _, p := range ps {
				out = append(out, p.Fields["summary"].(string))
			}
		}
		return out
	}
	bringChats(t, h, true, "Book the van")
	if got := suggested(1); len(got) != 1 || !strings.Contains(got[0], "Book the van") {
		t.Fatalf("brought, it is looked through: %q", got)
	}
	page := bringExport(t, h, false, `[{"title":"Moving house","create_time":1760000000,"current_node":"b","mapping":{
		"a":{"parent":null,"message":{"author":{"role":"user"},"content":{"parts":["Book the van"]}}},
		"b":{"parent":"a","message":{"author":{"role":"user"},"content":{"parts":["Call the plumber too."]}}}}}]`)
	if !strings.Contains(page, "1 carried on since") {
		t.Fatalf("carried on: %s", truncate(page))
	}
	got := suggested(2)
	time.Sleep(200 * time.Millisecond) // a third, if any, has time to come
	if got = suggested(2); len(got) != 2 || !strings.Contains(got[1], "Call the plumber") {
		t.Errorf("one suggestion for the new words, the van not again: %q", got)
	}

	// Never asked to look through them, a conversation carried on is not.
	b, hb := newApp(t)
	other := &findsToDo{}
	b.Chat.Provider = other
	b.Automate.StartWorker()
	bringChats(t, hb, false, "Book the van")
	bringExport(t, hb, false, `[{"title":"Moving house","create_time":1760000000,"current_node":"b","mapping":{
		"a":{"parent":null,"message":{"author":{"role":"user"},"content":{"parts":["Book the van"]}}},
		"b":{"parent":"a","message":{"author":{"role":"user"},"content":{"parts":["Call the plumber too."]}}}}}]`)
	time.Sleep(200 * time.Millisecond)
	other.mu.Lock()
	defer other.mu.Unlock()
	if ps, _ := b.Store.List(records.ProposalType, store.ListOptions{}); len(ps) != 0 || other.asked != 0 {
		t.Errorf("never asked, nothing is looked through: %d suggested, %d asked", len(ps), other.asked)
	}
}

// A very long conversation is kept as its start and its end: carried on,
// its start is the same, and it still updates its note.
func TestALongConversationCarriedOnUpdatesItsNote(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)
	long := strings.Repeat("We talked a long while. ", 10000)
	export := func(last string) string {
		return `[{"title":"Moving house","create_time":1760000000,"current_node":"b","mapping":{
		"a":{"parent":null,"message":{"author":{"role":"user"},"content":{"parts":["` + long + `"]}}},
		"b":{"parent":"a","message":{"author":{"role":"user"},"content":{"parts":["` + last + `"]}}}}}]`
	}
	bringExport(t, h, false, export("Book the van"))
	if page := bringExport(t, h, false, export("Book the van for the 14th")); !strings.Contains(page, "1 carried on since") {
		t.Errorf("a long conversation carried on updates its note: %s", truncate(page))
	}
	notes, _ := a.Store.List("note", store.ListOptions{})
	if len(notes) != 1 || !strings.HasSuffix(notes[0].Fields["body"].(string), "for the 14th") {
		t.Errorf("one note, with its new end: %d", len(notes))
	}
}
