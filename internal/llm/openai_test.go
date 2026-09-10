package llm_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
)

// TestOpenAIRequestShape checks the wire format an Ollama or OpenAI server
// receives: system first, tools declared, tool calls and results mapped.
func TestOpenAIRequestShape(t *testing.T) {
	var got map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" || r.Header.Get("Authorization") != "Bearer secret" {
			t.Errorf("unexpected request %s auth=%q", r.URL.Path, r.Header.Get("Authorization"))
		}
		json.NewDecoder(r.Body).Decode(&got)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"choices":[{"finish_reason":"tool_calls","message":{"role":"assistant","content":"","tool_calls":[{"id":"c1","type":"function","function":{"name":"add_component","arguments":"{\"component\":\"list\"}"}}]}}]}`))
	}))
	defer srv.Close()

	p := &llm.OpenAI{BaseURL: srv.URL + "/v1", Model: "m", APIKey: "secret", MaxTokens: 99}
	resp, err := p.Complete(context.Background(), llm.Request{
		System: "sys",
		Messages: []llm.Message{
			{Role: llm.RoleUser, Content: "hi"},
			{Role: llm.RoleAssistant, Content: "", ToolCalls: []llm.ToolCall{{ID: "prev", Name: "add_component", Args: json.RawMessage(`{"a":1}`)}}},
			{Role: llm.RoleTool, ToolResults: []llm.ToolResult{{CallID: "prev", Content: "ok"}}},
		},
		Tools: []llm.Tool{{Name: "add_component", Description: "d", Schema: map[string]any{"type": "object"}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got["model"] != "m" || got["max_tokens"] != float64(99) {
		t.Errorf("model/max_tokens: %v %v", got["model"], got["max_tokens"])
	}
	msgs := got["messages"].([]any)
	roles := []string{}
	for _, m := range msgs {
		roles = append(roles, m.(map[string]any)["role"].(string))
	}
	if strings.Join(roles, ",") != "system,user,assistant,tool" {
		t.Errorf("message roles: %v", roles)
	}
	toolMsg := msgs[3].(map[string]any)
	if toolMsg["tool_call_id"] != "prev" || toolMsg["content"] != "ok" {
		t.Errorf("tool result mapping: %v", toolMsg)
	}
	tools := got["tools"].([]any)
	fn := tools[0].(map[string]any)["function"].(map[string]any)
	if fn["name"] != "add_component" || fn["parameters"] == nil {
		t.Errorf("tool declaration: %v", tools[0])
	}
	if len(resp.ToolCalls) != 1 || resp.ToolCalls[0].Name != "add_component" || string(resp.ToolCalls[0].Args) != `{"component":"list"}` {
		t.Errorf("tool call parsing: %+v", resp.ToolCalls)
	}
}

func TestOpenAIErrorsAreReadable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error":{"message":"model 'llama9' not found, try pulling it first"}}`))
	}))
	defer srv.Close()
	p := &llm.OpenAI{BaseURL: srv.URL + "/v1", Model: "llama9"}
	_, err := p.Complete(context.Background(), llm.Request{Messages: []llm.Message{{Role: llm.RoleUser, Content: "hi"}}})
	if err == nil || !strings.Contains(err.Error(), "404") || !strings.Contains(err.Error(), "try pulling it first") {
		t.Errorf("error should carry status and server message: %v", err)
	}

	srv.Close()
	_, err = p.Complete(context.Background(), llm.Request{Messages: []llm.Message{{Role: llm.RoleUser, Content: "hi"}}})
	if err == nil || !strings.Contains(err.Error(), "could not reach the model at") {
		t.Errorf("connection failure should name the base URL: %v", err)
	}
}

func TestOpenAIEmptyArgumentsBecomeEmptyObject(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"choices":[{"message":{"role":"assistant","tool_calls":[{"id":"c","type":"function","function":{"name":"clear_canvas","arguments":""}}]}}]}`))
	}))
	defer srv.Close()
	p := &llm.OpenAI{BaseURL: srv.URL, Model: "m"}
	resp, err := p.Complete(context.Background(), llm.Request{Messages: []llm.Message{{Role: llm.RoleUser, Content: "clear"}}})
	if err != nil || string(resp.ToolCalls[0].Args) != "{}" {
		t.Errorf("empty arguments should become {}: %v %+v", err, resp)
	}
}

func TestNewProviderFromConfig(t *testing.T) {
	if _, err := llm.New(llm.Config{}); err != llm.ErrNotConfigured {
		t.Errorf("empty config: %v", err)
	}
	if _, err := llm.New(llm.Config{Provider: "none"}); err != llm.ErrNotConfigured {
		t.Errorf("none: %v", err)
	}
	if _, err := llm.New(llm.Config{Provider: "openai"}); err == nil || !strings.Contains(err.Error(), "llm.model is required") {
		t.Errorf("openai without model: %v", err)
	}
	if _, err := llm.New(llm.Config{Provider: "carrier-pigeon", Model: "x"}); err == nil || !strings.Contains(err.Error(), "unknown llm.provider") {
		t.Errorf("unknown provider: %v", err)
	}
	p, err := llm.New(llm.Config{Provider: "ollama", Model: "llama3.1"})
	if err != nil || !strings.Contains(p.Name(), "http://localhost:11434/v1") {
		t.Errorf("ollama alias should default the base url: %v %v", err, p)
	}
	t.Setenv("TEST_KEY", "k")
	a, err := llm.New(llm.Config{Provider: "anthropic", APIKeyEnv: "TEST_KEY"})
	if err != nil || !strings.Contains(a.Name(), "claude-opus-5") {
		t.Errorf("anthropic should default the model: %v %v", err, a)
	}
}
