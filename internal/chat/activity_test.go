package chat_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/app"
	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// newFullService builds a service with the complete starter schema:
// provenance fields on blocks, changes on messages, and the activity log.
func newFullService(t *testing.T) *chat.Service {
	t.Helper()
	types, err := schema.Load(filepath.Join("..", "..", "examples", "workspaces", "starter", "schema"))
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
	return &chat.Service{Store: st, Registry: reg, HistoryLimit: 10}
}

func TestTurnLeavesReceiptProvenanceAndActivity(t *testing.T) {
	svc := newFullService(t)
	svc.Provider = &scripted{steps: []*llm.Response{
		call("add_component", map[string]any{"component": "heading", "props": map[string]any{"text": "Shopping"}}),
		call("add_component", map[string]any{"component": "list", "props": map[string]any{"items": []string{"milk", "eggs"}}}),
	}}
	reply, err := svc.Send(context.Background(), "make a shopping list")
	if err != nil {
		t.Fatal(err)
	}
	changes, _ := reply.Fields["changes"].([]any)
	if len(changes) != 2 {
		t.Fatalf("assistant message should carry 2 changes, got %v", reply.Fields["changes"])
	}
	first := changes[0].(map[string]any)
	if first["action"] != "added" || first["component"] != "heading" || first["detail"] != "Shopping" || first["id"] == "" {
		t.Errorf("receipt entry wrong: %v", first)
	}
	if second := changes[1].(map[string]any); second["detail"] != "2 items" {
		t.Errorf("list summary should count items: %v", second)
	}

	blocks, _ := svc.Store.List(chat.BlockType, store.ListOptions{})
	for _, b := range blocks {
		if b.Fields["actor"] != "assistant" || b.Fields["created_by"] != "assistant" {
			t.Errorf("block provenance should be assistant: %v", b.Fields)
		}
	}

	log, _ := svc.Store.List(chat.ActivityType, store.ListOptions{OrderBy: "created_at"})
	var got []string
	for _, r := range log {
		got = append(got, r.Fields["actor"].(string)+":"+r.Fields["action"].(string))
	}
	want := "human:said,assistant:added,assistant:added"
	if joined := join(got); joined != want {
		t.Errorf("activity log %q, want %q", joined, want)
	}
}

func TestFailureIsLoggedAsSystem(t *testing.T) {
	svc := newFullService(t)
	svc.Provider = failing{}
	svc.Send(context.Background(), "hi")
	log, _ := svc.Store.List(chat.ActivityType, store.ListOptions{OrderBy: "created_at"})
	if len(log) != 2 || log[1].Fields["actor"] != "system" || log[1].Fields["action"] != "failed" {
		t.Errorf("expected human:said then system:failed, got %d entries", len(log))
	}
}

func TestOlderWorkspacesWithoutActivityStillWork(t *testing.T) {
	// The minimal schema in newService has no activity type and no
	// provenance fields; a turn must still succeed.
	svc, _ := newService(t)
	svc.Provider = &scripted{steps: []*llm.Response{
		call("add_component", map[string]any{"component": "text", "props": map[string]any{"content": "hi"}}),
	}}
	if _, err := svc.Send(context.Background(), "add text"); err != nil {
		t.Fatalf("turn failed on a minimal schema: %v", err)
	}
	if n, _ := svc.Store.Count(chat.BlockType); n != 1 {
		t.Errorf("block should be saved without provenance fields, got %d", n)
	}
}

func join(parts []string) string {
	out := ""
	for i, p := range parts {
		if i > 0 {
			out += ","
		}
		out += p
	}
	return out
}
