package chat_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/app"
	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// fakeProvider scripts a model: first it calls add_component, then answers.
type fakeProvider struct {
	calls    int
	requests []llm.Request
}

func (f *fakeProvider) Name() string { return "fake" }

func (f *fakeProvider) Complete(_ context.Context, req llm.Request) (*llm.Response, error) {
	f.calls++
	f.requests = append(f.requests, req)
	switch f.calls {
	case 1:
		args, _ := json.Marshal(map[string]any{"component": "list", "props": map[string]any{"items": []string{"Milk", "Eggs"}, "label": "Groceries"}})
		return &llm.Response{ToolCalls: []llm.ToolCall{{ID: "c1", Name: "add_component", Args: args}}}, nil
	case 2:
		// Deliberately bad props: the loop must report the error and continue.
		args, _ := json.Marshal(map[string]any{"component": "heading", "props": map[string]any{"level": 2}})
		return &llm.Response{ToolCalls: []llm.ToolCall{{ID: "c2", Name: "add_component", Args: args}}}, nil
	default:
		return &llm.Response{Text: "Added your grocery list."}, nil
	}
}

func newService(t *testing.T) (*chat.Service, *fakeProvider) {
	t.Helper()
	dir := t.TempDir()
	msg := "name: message\nfields:\n  role: {type: enum, values: [user, assistant, error], required: true}\n  content: {type: text, required: true}\n"
	blk := "name: block\nfields:\n  component: {type: string, required: true}\n  props: {type: json}\n  position: {type: int, default: 0}\n"
	os.WriteFile(filepath.Join(dir, "message.yaml"), []byte(msg), 0o644)
	os.WriteFile(filepath.Join(dir, "block.yaml"), []byte(blk), 0o644)
	types, err := schema.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	st, err := store.Open(":memory:", types)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	reg, err := app.NewRegistry("")
	if err != nil {
		t.Fatal(err)
	}
	fp := &fakeProvider{}
	return &chat.Service{Store: st, Registry: reg, Provider: fp, HistoryLimit: 10}, fp
}

func TestSendRunsToolLoop(t *testing.T) {
	svc, fp := newService(t)
	reply, err := svc.Send(context.Background(), "add a grocery list")
	if err != nil {
		t.Fatal(err)
	}
	if reply.Fields["content"] != "Added your grocery list." {
		t.Errorf("unexpected reply: %v", reply.Fields["content"])
	}
	if fp.calls != 3 {
		t.Errorf("expected 3 model calls, got %d", fp.calls)
	}
	blocks, _ := svc.Store.List(chat.BlockType, store.ListOptions{})
	if len(blocks) != 1 || blocks[0].Fields["component"] != "list" {
		t.Fatalf("expected one list block, got %+v", blocks)
	}
	// The invalid heading call must have been reported back as an error result.
	third := fp.requests[2]
	last := third.Messages[len(third.Messages)-1]
	if last.Role != llm.RoleTool || len(last.ToolResults) != 1 || !last.ToolResults[0].IsError {
		t.Errorf("expected an error tool result for the bad heading, got %+v", last)
	}
	if third.Messages[0].Role != llm.RoleUser {
		t.Errorf("conversation must start with the user turn")
	}
	msgs, _ := svc.Store.List(chat.MessageType, store.ListOptions{OrderBy: "created_at"})
	if len(msgs) != 2 || msgs[0].Fields["role"] != "user" || msgs[1].Fields["role"] != "assistant" {
		t.Errorf("expected user then assistant messages, got %d", len(msgs))
	}
}

func TestSendWithoutProviderRecordsError(t *testing.T) {
	svc, _ := newService(t)
	svc.Provider = nil
	rec, err := svc.Send(context.Background(), "hello")
	if err == nil || rec == nil || rec.Fields["role"] != "error" {
		t.Fatalf("expected a recorded error message, got rec=%v err=%v", rec, err)
	}
}
