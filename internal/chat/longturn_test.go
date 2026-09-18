package chat_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
)

// A model that keeps calling tools is stopped after a bounded number of
// rounds and asked to say where things stand, without tools. The person
// gets a reply in words with every change the turn made, not an error.
func TestALongTurnEndsInWords(t *testing.T) {
	var steps []*llm.Response
	for i := 0; i < 40; i++ {
		steps = append(steps, call("add_component", map[string]any{"component": "text", "props": map[string]any{"content": fmt.Sprintf("item %d", i)}}))
	}
	svc := newFullService(t)
	m := &scripted{steps: steps}
	svc.Provider = m
	rec, err := svc.Send(context.Background(), "add forty texts")
	if err != nil {
		t.Fatalf("a long turn ends in words, not an error: %v", err)
	}
	if rec.Fields["role"] != "assistant" || rec.Fields["content"] != "done" {
		t.Errorf("the reply is the model's last words, got %v: %v", rec.Fields["role"], rec.Fields["content"])
	}
	last := m.seen[len(m.seen)-1]
	if last.Tools != nil {
		t.Error("the last request offers no tools, so the model can only answer")
	}
	if ask := last.Messages[len(last.Messages)-1]; ask.Role != llm.RoleUser || !strings.Contains(ask.Content, "what you did") {
		t.Errorf("the model is asked to say where things stand, got %+v", ask)
	}
	changes, _ := rec.Fields["changes"].([]any)
	if rounds := len(m.seen) - 1; len(changes) != rounds || rounds >= 40 || rounds < 8 {
		t.Errorf("every change made in the %d rounds is kept on the reply, got %d changes", rounds, len(changes))
	}
}
