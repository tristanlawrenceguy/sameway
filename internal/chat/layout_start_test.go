package chat_test

import (
	"context"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/records"
)

// How the tab reads is given at the start of a turn, not only after a
// write, in Layout now's own words: its rows by name, and what is to look
// at with when to act on it. A tab with nothing wrong says it reads well
// and asks nothing.
func TestTheTabsTroublesAreGivenAtTheStart(t *testing.T) {
	t.Parallel()
	svc := newFullService(t)
	m := &scripted{steps: []*llm.Response{{Text: "ok"}, {Text: "ok"}}}
	svc.Provider = m
	svc.Call("add_component", []byte(`{"component":"text","props":{"content":"Hello."},"span":12}`))
	if _, err := svc.Send(context.Background(), "hello"); err != nil {
		t.Fatal(err)
	}
	if system := m.seen[0].System; strings.Contains(system, "To look at") || strings.Contains(system, "do not tidy unasked") || !strings.Contains(system, "Layout now: row 1:") {
		t.Errorf("a tab with nothing wrong gives its rows and nothing to look at:\n%s", system[strings.Index(system, "Current canvas"):])
	}
	svc.Store.Create(records.BlockType, map[string]any{"component": "button", "props": map[string]any{}, "position": 5, "canvas": ""})
	if _, err := svc.Send(context.Background(), "what is due today?"); err != nil {
		t.Fatal(err)
	}
	system := m.seen[len(m.seen)-1].System
	if !strings.Contains(system, "To look at: (1)") || !strings.Contains(system, "cannot be shown") || !strings.Contains(system, "do not tidy unasked") || strings.Contains(system, "Rows in the main region") {
		t.Errorf("the tab's trouble is given at the start, in Layout now's words:\n%s", system[strings.Index(system, "Current canvas"):])
	}
}
