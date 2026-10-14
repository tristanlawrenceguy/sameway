package chat_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
)

// What a model is given before the person has said anything that needs
// more: the instructions and the tools of a turn on a new workspace. It
// was 80 KB, about 20,000 tokens a turn, when every rule and tool went
// every time; it is about 42 KB with the rest found by name (more_tools).
// The budget is well above that, not just above it: what grows past it is
// detail to give when it is needed, not every turn.
func TestATurnFitsItsBudget(t *testing.T) {
	t.Parallel()
	svc := newFullService(t)
	m := &scripted{steps: []*llm.Response{{Text: "ok"}}}
	svc.Provider = m
	if _, err := svc.Send(context.Background(), "hello"); err != nil {
		t.Fatal(err)
	}
	tools, _ := json.Marshal(m.seen[0].Tools)
	size := len(m.seen[0].System) + len(tools)
	t.Logf("a turn: %d bytes (instructions %d, tools %d)", size, len(m.seen[0].System), len(tools))
	if size > 56*1024 {
		t.Errorf("a turn is %d bytes, over 56 KB: say a rule with the turn that needs it (rules.go) or name a tool for more_tools, rather than every turn", size)
	}
}
