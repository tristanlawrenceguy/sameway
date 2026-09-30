package chat_test

import (
	"context"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
)

// Every tool the model is offered has a handler, and every handler is
// for a tool it is offered or a question can carry: a tool added to the
// list with nothing to run it, or a handler left for a tool taken out, is
// caught here rather than by a model calling it.
func TestEveryToolHasAHandler(t *testing.T) {
	svc := newFullService(t)
	// Every hook the server gives, so every tool is offered.
	svc.Look = func(context.Context, map[string]any) (string, error) { return "", nil }
	handled := map[string]bool{}
	for _, name := range chat.HandledTools() {
		handled[name] = true
	}
	offered := map[string]bool{"accept_action": true} // a question's Yes, not the model's own
	for _, tool := range svc.Tools() {
		offered[tool.Name] = true
		if !handled[tool.Name] {
			t.Errorf("%s is offered but nothing runs it", tool.Name)
		}
	}
	for name := range handled {
		if !offered[name] {
			t.Errorf("%s has a handler but is not a tool the model is offered", name)
		}
	}
}
