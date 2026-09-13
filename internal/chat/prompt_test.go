package chat_test

import (
	"context"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
)

func TestSystemPromptIncludesA11yNotes(t *testing.T) {
	svc := newFullService(t)
	m := &scripted{steps: []*llm.Response{{Text: "done"}}}
	svc.Provider = m
	svc.Send(context.Background(), "hi")
	prompt := m.seen[0].System
	if !strings.Contains(prompt, "Kind is conveyed by a text prefix") {
		t.Errorf("system prompt should include alert a11y notes; got:\n%s", prompt[len(prompt)-500:])
	}
	if !strings.Contains(prompt, "\nA11y:") {
		t.Errorf("system prompt should have A11y: line after component props")
	}
}
