package chat_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/llm"
)

// There is no count of rounds: a model that makes one call per round
// may take as many rounds as the work takes.
func TestALongTurnMayGoOn(t *testing.T) {
	var steps []*llm.Response
	for i := 0; i < 40; i++ {
		steps = append(steps, call("add_component", map[string]any{"component": "text", "props": map[string]any{"content": fmt.Sprintf("item %d", i)}}))
	}
	svc := newFullService(t)
	m := &scripted{steps: steps}
	svc.Provider = m
	rec, err := svc.Send(context.Background(), "add forty texts")
	if err != nil {
		t.Fatal(err)
	}
	changes, _ := rec.Fields["changes"].([]any)
	if len(changes) != 40 || rec.Fields["content"] != "done" {
		t.Errorf("all forty land and the model answers in its own time, got %d changes and %v", len(changes), rec.Fields["content"])
	}
	if last := m.seen[len(m.seen)-1]; last.Tools == nil {
		t.Error("a turn that is getting somewhere keeps its tools")
	}
}

// A call made again with the same arguments is a loop. The turn ends
// there: the repeat does not run, the model is asked without tools to
// say where things stand, and its words are the reply, with what the
// turn did.
func TestARepeatedCallEndsTheTurnInWords(t *testing.T) {
	same := func() *llm.Response {
		return call("add_component", map[string]any{"component": "text", "props": map[string]any{"content": "again"}})
	}
	svc := newFullService(t)
	m := &scripted{steps: []*llm.Response{same(), same(), same()}}
	svc.Provider = m
	rec, err := svc.Send(context.Background(), "add a text")
	if err != nil {
		t.Fatalf("a loop ends in words, not an error: %v", err)
	}
	changes, _ := rec.Fields["changes"].([]any)
	if len(changes) != 1 || rec.Fields["content"] != "done" {
		t.Errorf("the first call ran, the repeat did not, and the reply is the model's words; got %d changes and %v", len(changes), rec.Fields["content"])
	}
	last := m.seen[len(m.seen)-1]
	ask := last.Messages[len(last.Messages)-1]
	if last.Tools != nil || ask.Role != llm.RoleUser || !strings.Contains(ask.Content, "same call") {
		t.Errorf("the model is told why and asked, without tools, to say where things stand; got tools %v and %q", last.Tools != nil, ask.Content)
	}
}

// Tools that fail the same way three rounds in a row are a wall; the
// turn ends the same way. Different failures are not: each error tells
// the model something, and it may keep trying.
func TestTheSameFailureThreeTimesEndsTheTurn(t *testing.T) {
	bad := func(n int) *llm.Response { return call("frobnicate", map[string]any{"try": n}) }
	svc := newFullService(t)
	m := &scripted{steps: []*llm.Response{bad(1), bad(2), bad(3), bad(4), bad(5)}}
	svc.Provider = m
	rec, err := svc.Send(context.Background(), "fix it")
	if err != nil {
		t.Fatal(err)
	}
	if rec.Fields["content"] != "done" || len(m.seen) != 4 {
		t.Errorf("after the same failure three times the model is asked to wrap up: %d requests, reply %v", len(m.seen), rec.Fields["content"])
	}

	svc = newFullService(t)
	m = &scripted{steps: []*llm.Response{
		call("frobnicate", nil),
		call("update_component", map[string]any{"id": "nothing", "props": map[string]any{}}),
		call("add_component", map[string]any{"component": "carousel", "props": map[string]any{}}),
		call("add_component", map[string]any{"component": "text", "props": map[string]any{"content": "got there"}}),
	}}
	svc.Provider = m
	rec, err = svc.Send(context.Background(), "keep trying")
	if err != nil {
		t.Fatal(err)
	}
	if changes, _ := rec.Fields["changes"].([]any); len(changes) != 1 || len(m.seen) != 5 {
		t.Errorf("different failures let the model keep trying until it gets there: %d requests, %d changes", len(m.seen), len(changes))
	}
}

// stopping is a model that waits until the turn is stopped.
type stopping struct{ started chan struct{} }

func (s *stopping) Name() string { return "stopping" }
func (s *stopping) Complete(ctx context.Context, _ llm.Request) (*llm.Response, error) {
	close(s.started)
	<-ctx.Done()
	return nil, ctx.Err()
}

// The person can stop a turn. What it did stays, and the reply says it
// was stopped rather than reporting an error.
func TestAStoppedTurnKeepsWhatItDid(t *testing.T) {
	svc := newFullService(t)
	chat.Record(svc.Store, "assistant", chat.Change{Action: "created", Component: "note", Detail: "earlier"})
	p := &stopping{started: make(chan struct{})}
	svc.Provider = p
	ctx, cancel := context.WithCancel(context.Background())
	go func() { <-p.started; cancel() }()
	rec, err := svc.Send(ctx, "do something long")
	if err != nil || rec == nil {
		t.Fatalf("a stopped turn is not an error: %v", err)
	}
	if rec.Fields["role"] != "assistant" || rec.Fields["content"] != "Stopped, as you asked." {
		t.Errorf("the reply says the turn was stopped, got %v: %v", rec.Fields["role"], rec.Fields["content"])
	}
	if errors.Is(err, context.Canceled) {
		t.Error("the cancellation is not passed on as an error")
	}
}
