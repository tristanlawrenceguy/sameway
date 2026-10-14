package chat_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
)

// The canvas the model is given says when each block was added and by
// whom, so a button an agent left days ago reads as left over beside what
// the person asked for today.
func TestTheCanvasSaysWhenAndByWhomEachBlockWasAdded(t *testing.T) {
	t.Parallel()
	svc := newFullService(t)
	svc.Call("add_component", []byte(`{"component":"collection","props":{"type":"task","label":"Tasks"}}`))
	later := time.Now().Add(9*24*time.Hour + time.Hour)
	svc.Book.Clock = func() time.Time { return later }
	m := &scripted{steps: []*llm.Response{{Text: "ok"}}}
	svc.Provider = m
	if _, err := svc.Send(context.Background(), "hello"); err != nil {
		t.Fatal(err)
	}
	listing := m.seen[0].System[strings.Index(m.seen[0].System, "Current canvas"):]
	if !strings.Contains(listing, "added 9 days ago by the assistant") {
		t.Errorf("the block says it was added 9 days ago by the assistant:\n%s", listing)
	}
}
