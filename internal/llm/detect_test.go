package llm_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
)

// A server is offered by a model that can talk: one that only turns words
// into numbers for search (nomic-embed-text) is listed, never first.
func TestDetectFindsServersWithModels(t *testing.T) {
	t.Parallel()
	good := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" {
			http.NotFound(w, r)
			return
		}
		w.Write([]byte(`{"object":"list","data":[{"id":"nomic-embed-text:latest"},{"id":"qwen3"},{"id":"llama3.1"}]}`))
	}))
	defer good.Close()
	empty := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"data":[]}`))
	}))
	defer empty.Close()
	broken := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>not an api</html>`))
	}))
	defer broken.Close()

	found := llm.Detect(context.Background(), []llm.Candidate{
		{"dead", "http://127.0.0.1:1/v1"},
		{"empty", empty.URL + "/v1"},
		{"broken", broken.URL + "/v1"},
		{"good", good.URL + "/v1"},
	})
	if len(found) != 1 || found[0].Server != "good" || found[0].Model != "qwen3" || len(found[0].Models) != 3 {
		t.Fatalf("detect: %+v", found)
	}
	if found[0].BaseURL != good.URL+"/v1" {
		t.Errorf("base url: %s", found[0].BaseURL)
	}
}

func TestDetectWithNothingRunning(t *testing.T) {
	t.Parallel()
	if found := llm.Detect(context.Background(), []llm.Candidate{{"dead", "http://127.0.0.1:1/v1"}}); len(found) != 0 {
		t.Errorf("expected nothing, got %+v", found)
	}
}

// A model is on this computer when Ollama or another local server runs
// it; Claude, Claude Code and a service at an address elsewhere are not.
func TestAModelIsOnThisComputerOrElsewhere(t *testing.T) {
	t.Parallel()
	for cfg, want := range map[llm.Config]bool{
		{Provider: "openai", BaseURL: "http://localhost:11434/v1"}: true,
		{Provider: "ollama"}: true,
		{Provider: "openai", BaseURL: "https://openrouter.ai/api/v1"}:    false,
		{Provider: "openai", BaseURL: "http://localhost.example.com/v1"}: false,
		{Provider: "openai", BaseURL: "http://[::1]:1234/v1"}:            true,
		{Provider: "anthropic"}:   false,
		{Provider: "claude-code"}: false,
	} {
		if got := cfg.OnThisComputer(); got != want {
			t.Errorf("%+v: %v", cfg, got)
		}
	}
}
