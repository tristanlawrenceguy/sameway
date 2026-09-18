package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// A streamed completion arrives in pieces: the words are handed on as they
// come, and a tool call split across chunks is put back together.
func TestOpenAIStreamsWordsAndReassemblesToolCalls(t *testing.T) {
	chunks := []string{
		`{"choices":[{"delta":{"content":"Let me "}}]}`,
		`{"choices":[{"delta":{"content":"add that."}}]}`,
		`{"choices":[{"delta":{"tool_calls":[{"index":0,"id":"call_1","function":{"name":"add_component","arguments":"{\"compo"}}]}}]}`,
		`{"choices":[{"delta":{"tool_calls":[{"index":0,"function":{"arguments":"nent\":\"card\"}"}}]}}]}`,
		`{"choices":[{"delta":{},"finish_reason":"tool_calls"}]}`,
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		if body["stream"] != true {
			t.Errorf("a streamed request says stream: true, got %v", body["stream"])
		}
		w.Header().Set("Content-Type", "text/event-stream")
		for _, c := range chunks {
			fmt.Fprintf(w, "data: %s\n\n", c)
		}
		fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer srv.Close()

	var pieces []string
	p := &OpenAI{BaseURL: srv.URL, Model: "m"}
	resp, err := p.Stream(context.Background(), Request{Messages: []Message{{Role: RoleUser, Content: "add a card"}}}, func(s string) { pieces = append(pieces, s) })
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(pieces, "|") != "Let me |add that." || resp.Text != "Let me add that." {
		t.Errorf("the words come as they are said, got %q and %q", pieces, resp.Text)
	}
	if len(resp.ToolCalls) != 1 || resp.ToolCalls[0].Name != "add_component" || string(resp.ToolCalls[0].Args) != `{"component":"card"}` || resp.ToolCalls[0].ID != "call_1" {
		t.Errorf("a tool call split across chunks is whole again, got %+v", resp.ToolCalls)
	}
	if resp.StopReason != "tool_calls" {
		t.Errorf("the stop reason comes through, got %q", resp.StopReason)
	}
}

// A refusal from the model server is said in words, not read as a stream.
func TestOpenAIStreamSaysWhatWentWrong(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, `{"error":{"message":"bad key"}}`)
	}))
	defer srv.Close()
	p := &OpenAI{BaseURL: srv.URL, Model: "m"}
	_, err := p.Stream(context.Background(), Request{}, nil)
	if err == nil || !strings.Contains(err.Error(), "bad key") {
		t.Errorf("the server's words should be in the error, got %v", err)
	}
}
