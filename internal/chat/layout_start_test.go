package chat_test

import (
	"context"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/records"
)

// What is wrong with the tab is given at the start of a turn, not only
// after a write, to say when it bears on what is asked; a tab with nothing
// wrong gives nothing.
func TestTheTabsTroublesAreGivenAtTheStart(t *testing.T) {
	t.Parallel()
	svc := newFullService(t)
	m := &scripted{steps: []*llm.Response{{Text: "ok"}, {Text: "ok"}}}
	svc.Provider = m
	if _, err := svc.Send(context.Background(), "hello"); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(m.seen[0].System, "On this tab, to look at") {
		t.Error("a tab with nothing wrong gives nothing")
	}
	svc.Store.Create(records.BlockType, map[string]any{"component": "button", "props": map[string]any{}, "position": 5, "canvas": ""})
	if _, err := svc.Send(context.Background(), "what is due today?"); err != nil {
		t.Fatal(err)
	}
	system := m.seen[len(m.seen)-1].System
	if !strings.Contains(system, "On this tab, to look at") || !strings.Contains(system, "cannot be shown") || !strings.Contains(system, "do not tidy unasked") {
		t.Errorf("the tab's trouble is given at the start:\n%s", system[strings.Index(system, "Current canvas"):])
	}
}
