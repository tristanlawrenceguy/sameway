package server_test

import (
	"context"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
)

// promptBudget is what the in-app prompt may weigh on the starter
// workspace. It carried every component's props and every type's schema
// until it reached 115 KB and had no room left; now it names each in a
// line and a refusal gives the whole of one when it is needed, about 32 KB.
// grows past this is paid for on every turn, so what is new goes behind
// a refusal too.
const promptBudget = 40 * 1024

// The assistant is told the rules and the catalogue on every turn, so the
// prompt is held to a budget, and among the rules is the one the haiku
// runs broke: it never makes up records to have something to show.
func TestPromptStaysInBudgetAndForbidsInventedData(t *testing.T) {
	a, _ := newApp(t)
	model := &scripted{steps: []*llm.Response{{Text: "ok"}}}
	a.Chat.Provider, a.Chat.ProviderErr = model, nil
	if _, err := a.Chat.Send(context.Background(), "hello"); err != nil {
		t.Fatal(err)
	}
	if len(model.seen) == 0 {
		t.Fatal("the model was not asked")
	}
	system := model.seen[0].System
	t.Logf("in-app prompt: %d bytes", len(system))
	if len(system) > promptBudget {
		t.Errorf("the prompt is %d bytes, over the budget of %d", len(system), promptBudget)
	}
	for _, want := range []string{"Never invent records", "never log entries to test"} {
		if !strings.Contains(system, want) {
			t.Errorf("the prompt should say %q", want)
		}
	}
}
