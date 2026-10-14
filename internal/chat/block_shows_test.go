package chat_test

import (
	"context"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/blocks"
	"github.com/tristanlawrenceguy/sameway/internal/llm"
)

// The canvas the model is given says what each block shows now, as the
// person sees it, beside its props: a list of two tasks says so, an empty
// one says it shows nothing yet.
func TestTheCanvasSaysWhatEachBlockShows(t *testing.T) {
	t.Parallel()
	svc := newFullService(t)
	svc.Blocks = &blocks.Workspace{Store: svc.Store} // as the app has it
	svc.Store.Create("task", map[string]any{"title": "Pay rent"})
	svc.Store.Create("task", map[string]any{"title": "Call the bank"})
	svc.Call("add_component", []byte(`{"component":"collection","props":{"type":"task","label":"Tasks"}}`))
	svc.Call("add_component", []byte(`{"component":"collection","props":{"type":"note","label":"Notes"}}`))
	m := &scripted{steps: []*llm.Response{{Text: "ok"}}}
	svc.Provider = m
	if _, err := svc.Send(context.Background(), "hello"); err != nil {
		t.Fatal(err)
	}
	listing := m.seen[0].System[strings.Index(m.seen[0].System, "Current canvas"):]
	if !strings.Contains(listing, "shows: 2 tasks") {
		t.Errorf("the tasks list says it shows 2 tasks:\n%s", listing)
	}
	if !strings.Contains(listing, "shows: nothing") {
		t.Errorf("the empty notes list says it shows nothing:\n%s", listing)
	}
}
