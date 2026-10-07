package server_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
)

// A workspace on Ollama, with Ollama installed and not running, starts it
// rather than saying an address is not answering and offering to download
// Ollama again; once it answers, the card goes. Started once a minute at
// most.
func TestOllamaInstalledButAsleepIsStarted(t *testing.T) {
	asleep := httptest.NewServer(http.NotFoundHandler())
	asleep.Close() // nothing listens here
	wasURL, wasInstalled, wasCandidates := llm.OllamaURL, llm.OllamaInstalled, llm.DefaultCandidates
	defer func() {
		llm.OllamaURL, llm.OllamaInstalled, llm.DefaultCandidates = wasURL, wasInstalled, wasCandidates
	}()
	llm.OllamaURL, llm.DefaultCandidates = asleep.URL, nil
	started := 0
	llm.OllamaInstalled = func() *exec.Cmd {
		started++
		return exec.Command(os.Args[0], "-test.run=^$") // a program that starts and ends
	}

	a, h := newApp(t)
	yaml := filepath.Join(a.Workspace.Dir, "workspace.yaml")
	os.WriteFile(yaml, []byte("name: T\nllm:\n  provider: openai\n  base_url: "+asleep.URL+"/v1\n  model: qwen\n"), 0o644)
	a.Workspace.Config.LLM = llm.Config{Provider: "openai", BaseURL: asleep.URL + "/v1", Model: "qwen"}
	a.Chat.Provider, a.Chat.ProviderErr = &llm.OpenAI{BaseURL: asleep.URL + "/v1", Model: "qwen"}, nil

	page := get(t, h, "/chat").Body.String()
	if !strings.Contains(page, "Ollama was not running, so Sameway is starting it.") || strings.Contains(page, "Download Ollama") || strings.Contains(page, asleep.URL) {
		t.Errorf("Ollama asleep is started and said so, with no download and no address: %s", truncate(page))
	}
	postForm(t, h, "/model/check", nil)
	get(t, h, "/chat")
	if started != 1 {
		t.Errorf("started once a minute at most, got %d", started)
	}
}
