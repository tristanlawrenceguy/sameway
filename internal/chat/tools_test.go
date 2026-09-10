package chat_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// scripted returns canned responses in order, then plain text.
type scripted struct {
	steps []*llm.Response
	seen  []llm.Request
}

func (s *scripted) Name() string { return "scripted" }

func (s *scripted) Complete(_ context.Context, req llm.Request) (*llm.Response, error) {
	s.seen = append(s.seen, req)
	if len(s.steps) == 0 {
		return &llm.Response{Text: "done"}, nil
	}
	next := s.steps[0]
	s.steps = s.steps[1:]
	return next, nil
}

func call(name string, args map[string]any) *llm.Response {
	raw, _ := json.Marshal(args)
	return &llm.Response{ToolCalls: []llm.ToolCall{{ID: "c", Name: name, Args: raw}}}
}

func lastToolResult(req llm.Request) llm.ToolResult {
	last := req.Messages[len(req.Messages)-1]
	return last.ToolResults[len(last.ToolResults)-1]
}

func withModel(t *testing.T, steps ...*llm.Response) (*chat.Service, *scripted) {
	t.Helper()
	svc, _ := newService(t)
	m := &scripted{steps: steps}
	svc.Provider = m
	return svc, m
}

func TestUpdateRemoveAndClear(t *testing.T) {
	svc, _ := withModel(t, call("add_component", map[string]any{"component": "text", "props": map[string]any{"content": "v1"}}))
	if _, err := svc.Send(context.Background(), "add text"); err != nil {
		t.Fatal(err)
	}
	blocks, _ := svc.Store.List(chat.BlockType, store.ListOptions{})
	id := blocks[0].ID

	svc.Provider = &scripted{steps: []*llm.Response{call("update_component", map[string]any{"id": id, "props": map[string]any{"content": "v2", "muted": true}})}}
	if _, err := svc.Send(context.Background(), "make it muted"); err != nil {
		t.Fatal(err)
	}
	rec, _ := svc.Store.Get(chat.BlockType, id)
	props := rec.Fields["props"].(map[string]any)
	if props["content"] != "v2" || props["muted"] != true {
		t.Errorf("update did not replace props: %v", props)
	}

	m := &scripted{steps: []*llm.Response{call("update_component", map[string]any{"id": id, "props": map[string]any{"bogus": 1}})}}
	svc.Provider = m
	svc.Send(context.Background(), "break it")
	if res := lastToolResult(m.seen[1]); !res.IsError || !strings.Contains(res.Content, "invalid props") {
		t.Errorf("bad update should return an error result: %+v", res)
	}
	rec, _ = svc.Store.Get(chat.BlockType, id)
	if rec.Fields["props"].(map[string]any)["content"] != "v2" {
		t.Errorf("bad update must leave the block unchanged")
	}

	m = &scripted{steps: []*llm.Response{call("remove_component", map[string]any{"id": "missing"})}}
	svc.Provider = m
	svc.Send(context.Background(), "remove")
	if res := lastToolResult(m.seen[1]); !res.IsError {
		t.Errorf("removing a missing block should be an error result")
	}
	svc.Provider = &scripted{steps: []*llm.Response{call("remove_component", map[string]any{"id": id})}}
	svc.Send(context.Background(), "remove")
	if n, _ := svc.Store.Count(chat.BlockType); n != 0 {
		t.Errorf("remove left %d blocks", n)
	}

	svc.Provider = &scripted{steps: []*llm.Response{
		call("add_component", map[string]any{"component": "text", "props": map[string]any{"content": "a"}}),
		call("add_component", map[string]any{"component": "text", "props": map[string]any{"content": "b"}}),
		call("clear_canvas", nil),
	}}
	svc.Send(context.Background(), "two then clear")
	if n, _ := svc.Store.Count(chat.BlockType); n != 0 {
		t.Errorf("clear_canvas left %d blocks", n)
	}

	// Starting over keeps the conversation the person is typing into.
	svc.Provider = &scripted{steps: []*llm.Response{
		call("add_component", map[string]any{"component": "chat", "props": map[string]any{}}),
		call("add_component", map[string]any{"component": "text", "props": map[string]any{"content": "c"}}),
		call("clear_canvas", nil),
	}}
	svc.Send(context.Background(), "chat, text, then clear")
	left, _ := svc.Store.List(chat.BlockType, store.ListOptions{})
	if len(left) != 1 || left[0].Fields["component"] != chat.ComponentName {
		t.Errorf("clear_canvas should leave only the chat block, got %+v", left)
	}
}

// TestComponentNamesAreCaseInsensitive covers what local models actually
// send: "List" or " Table " instead of the catalogue name.
func TestComponentNamesAreCaseInsensitive(t *testing.T) {
	svc, m := withModel(t,
		call("add_component", map[string]any{"component": "List", "props": map[string]any{"items": []string{"milk", "eggs"}}}),
		call("add_component", map[string]any{"component": " TABLE ", "props": map[string]any{"caption": "c", "columns": []string{"a"}, "rows": [][]string{{"1"}}}}),
	)
	svc.Send(context.Background(), "add things")
	for i := 1; i <= 2; i++ {
		if res := lastToolResult(m.seen[i]); res.IsError {
			t.Errorf("call %d should succeed despite casing: %s", i, res.Content)
		}
	}
	blocks, _ := svc.Store.List(chat.BlockType, store.ListOptions{OrderBy: "position"})
	if len(blocks) != 2 || blocks[0].Fields["component"] != "list" || blocks[1].Fields["component"] != "table" {
		t.Errorf("stored names should be canonical: %+v", blocks)
	}
}

// TestToolSchemasAreStrictJSON guards what local servers choke on: llama.cpp
// builds a grammar from each tool schema and rejects null or missing pieces.
func TestToolSchemasAreStrictJSON(t *testing.T) {
	svc, m := withModel(t)
	svc.Send(context.Background(), "hi")
	for _, tool := range m.seen[0].Tools {
		raw, err := json.Marshal(tool.Schema)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(raw), "null") {
			t.Errorf("tool %s schema contains null: %s", tool.Name, raw)
		}
		var s struct {
			Type       string         `json:"type"`
			Properties map[string]any `json:"properties"`
			Required   []string       `json:"required"`
		}
		json.Unmarshal(raw, &s)
		if s.Type != "object" || s.Properties == nil {
			t.Errorf("tool %s schema must be an object with properties: %s", tool.Name, raw)
		}
		for _, r := range s.Required {
			if _, ok := s.Properties[r]; !ok {
				t.Errorf("tool %s requires %q which is not a property", tool.Name, r)
			}
		}
	}
}

func TestBlocksKeepInsertionOrder(t *testing.T) {
	svc, _ := withModel(t,
		call("add_component", map[string]any{"component": "heading", "props": map[string]any{"text": "First"}}),
		call("add_component", map[string]any{"component": "heading", "props": map[string]any{"text": "Second"}}),
		call("add_component", map[string]any{"component": "heading", "props": map[string]any{"text": "Third"}}),
	)
	svc.Send(context.Background(), "three headings")
	blocks, _ := svc.Store.List(chat.BlockType, store.ListOptions{OrderBy: "position"})
	var got []string
	for _, b := range blocks {
		got = append(got, b.Fields["props"].(map[string]any)["text"].(string))
	}
	if strings.Join(got, ",") != "First,Second,Third" {
		t.Errorf("order: %v", got)
	}
}

func TestToolErrorsGuideTheModel(t *testing.T) {
	svc, m := withModel(t,
		call("add_component", map[string]any{"component": "carousel", "props": map[string]any{}}),
		call("add_component", map[string]any{"component": "button", "props": map[string]any{"label": "x", "variant": "huge"}}),
		call("frobnicate", nil),
		&llm.Response{ToolCalls: []llm.ToolCall{{ID: "bad", Name: "add_component", Args: json.RawMessage(`{not json`)}}},
	)
	svc.Send(context.Background(), "try things")
	checks := []struct {
		req  int
		want string
	}{
		{1, "unknown component \"carousel\". Available: alert, badge, button"},
		{2, "invalid props"},
		{3, "unknown tool frobnicate"},
		{4, "not valid JSON"},
	}
	for _, c := range checks {
		res := lastToolResult(m.seen[c.req])
		if !res.IsError || !strings.Contains(res.Content, c.want) {
			t.Errorf("call %d: want error containing %q, got %+v", c.req, c.want, res)
		}
	}
	if n, _ := svc.Store.Count(chat.BlockType); n != 0 {
		t.Errorf("no block should have been saved, got %d", n)
	}
}

func TestSystemPromptCarriesCatalogueAndCanvas(t *testing.T) {
	svc, m := withModel(t, call("add_component", map[string]any{"component": "list", "props": map[string]any{"items": []string{"a"}}}))
	svc.ExtraPrompt = "Always answer in Dutch."
	svc.Send(context.Background(), "hi")
	first, second := m.seen[0].System, m.seen[1].System
	for _, want := range []string{"Component catalogue", "button: ", `"additionalProperties":false`, "Always answer in Dutch.", "(empty)"} {
		if !strings.Contains(first, want) {
			t.Errorf("first system prompt missing %q", want)
		}
	}
	if !strings.Contains(second, "list span=6 frame=card tone=none {") || strings.Contains(second, "(empty)") {
		t.Errorf("second system prompt should list the new block: %s", second[len(second)-200:])
	}
	if len(m.seen[0].Tools) != 4 {
		t.Errorf("expected 4 tools, got %d", len(m.seen[0].Tools))
	}
}

func TestHistoryLimitAndErrorFiltering(t *testing.T) {
	svc, m := withModel(t)
	svc.HistoryLimit = 3
	for _, text := range []string{"one", "two"} {
		svc.Send(context.Background(), text)
	}
	svc.Store.Create(chat.MessageType, map[string]any{"role": "error", "content": "boom"})
	svc.Send(context.Background(), "three")
	last := m.seen[len(m.seen)-1].Messages
	if last[0].Role != llm.RoleUser {
		t.Errorf("history must start with a user turn, got %s", last[0].Role)
	}
	for _, msg := range last {
		if strings.Contains(msg.Content, "boom") {
			t.Errorf("error notices must not be sent to the model")
		}
	}
	if len(last) > 3 {
		t.Errorf("history limit not applied: %d messages", len(last))
	}
	if err := svc.Clear(); err != nil {
		t.Fatal(err)
	}
	if n, _ := svc.Store.Count(chat.MessageType); n != 0 {
		t.Errorf("Clear left %d messages", n)
	}
}

func TestProviderFailureIsRecorded(t *testing.T) {
	svc, _ := newService(t)
	svc.Provider = failing{}
	rec, err := svc.Send(context.Background(), "hi")
	if err == nil || rec == nil || rec.Fields["role"] != "error" || !strings.Contains(rec.Fields["content"].(string), "boom") {
		t.Errorf("provider failure should be recorded as an error message: %v %v", rec, err)
	}
}

type failing struct{}

func (failing) Name() string { return "failing" }
func (failing) Complete(context.Context, llm.Request) (*llm.Response, error) {
	return nil, errors.New("boom")
}
