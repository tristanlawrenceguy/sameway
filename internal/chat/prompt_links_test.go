package chat_test

import (
	"context"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
)

// The model is asked for a link named by the thing's title, never a bare
// path as words: a path in a reply is machine language on the page
// (backlog 0548).
func TestPromptAsksForNamedLinks(t *testing.T) {
	t.Parallel()
	svc := newFullService(t)
	m := &scripted{steps: []*llm.Response{}}
	svc.Provider = m
	if _, err := svc.Send(context.Background(), "hello"); err != nil {
		t.Fatal(err)
	}
	sys := m.seen[0].System
	if !strings.Contains(sys, "[Seeds to buy](/t/note/<id>)") {
		t.Errorf("the prompt should show a link named by its title")
	}
	if strings.Contains(sys, "as the page path the tool returns") {
		t.Errorf("the prompt still asks for the bare page path")
	}
	if !strings.Contains(sys, "no Markdown, except a link") {
		t.Errorf("the prompt's plain-text rule should allow links")
	}
}
