package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// Candidate is a local server to probe during detection.
type Candidate struct {
	// Server is the human name printed when found, such as "Ollama".
	Server string
	// BaseURL is the OpenAI-compatible base, ending in /v1.
	BaseURL string
}

// DefaultCandidates lists the local servers Detect probes, in order of
// preference. All speak the OpenAI chat completions API.
var DefaultCandidates = []Candidate{
	{"Ollama", "http://127.0.0.1:11434/v1"},
	{"LM Studio", "http://127.0.0.1:1234/v1"},
	{"llama.cpp", "http://127.0.0.1:8090/v1"},
	{"llama.cpp", "http://127.0.0.1:8080/v1"},
	{"local server", "http://127.0.0.1:8000/v1"},
}

// Detected is a reachable local model server and its first model.
type Detected struct {
	Server  string
	BaseURL string
	Model   string
	Models  []string
}

// Detect probes each candidate's /models endpoint with a short timeout and
// returns every server that answered with at least one model, in the
// candidates' order. The probes run at once, so nothing running costs one
// timeout, not one per candidate: a page can ask while it is being made.
func Detect(ctx context.Context, candidates []Candidate) []Detected {
	found := make([]*Detected, len(candidates))
	var wg sync.WaitGroup
	for i, c := range candidates {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if models := listModels(ctx, c.BaseURL); len(models) > 0 {
				found[i] = &Detected{Server: c.Server, BaseURL: c.BaseURL, Model: models[0], Models: models}
			}
		}()
	}
	wg.Wait()
	var out []Detected
	for _, d := range found {
		if d != nil {
			out = append(out, *d)
		}
	}
	return out
}

// listModels is the models an OpenAI-compatible server says it has, or
// nothing when it does not answer.
func listModels(ctx context.Context, base string) []string {
	client := &http.Client{Timeout: 1500 * time.Millisecond}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(base, "/")+"/models", nil)
	if err != nil {
		return nil
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	var body struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if json.NewDecoder(resp.Body).Decode(&body) != nil || resp.StatusCode != http.StatusOK {
		return nil
	}
	var models []string
	for _, m := range body.Data {
		if m.ID != "" {
			models = append(models, m.ID)
		}
	}
	return models
}

// Answers says whether a configured model can be reached now, and when it
// cannot, why, in words a person reads: the model server is not answering,
// the program is not installed, the key is not saved. It looks without
// sending a conversation anywhere.
func Answers(ctx context.Context, cfg Config) (bool, string) {
	switch strings.ToLower(cfg.Provider) {
	case "", "none":
		return false, "No AI model is set up yet."
	case "openai", "openai-compatible", "ollama", "lmstudio", "openrouter":
		base := cfg.BaseURL
		if base == "" {
			base = "http://localhost:11434/v1"
		}
		// A cloud service lists its models only to a key; a local one to
		// anyone. Only a local one can be asked without the key.
		if !local(base) {
			if cfg.APIKeyEnv != "" && os.Getenv(cfg.APIKeyEnv) == "" {
				return false, "The key for the AI service at " + base + " is not saved on this computer (" + cfg.APIKeyEnv + ")."
			}
			return true, ""
		}
		if models := listModels(ctx, base); len(models) > 0 {
			return true, ""
		}
		return false, "The AI model at " + base + " isn't answering. It may not be running."
	case "anthropic":
		if cfg.APIKeyEnv == "" || os.Getenv(cfg.APIKeyEnv) == "" {
			return false, "The key for Claude is not saved on this computer."
		}
		return true, ""
	case "claude-code":
		if _, err := exec.LookPath("claude"); err != nil {
			return false, "Claude Code is not installed on this computer."
		}
		return true, ""
	}
	return true, ""
}

// local says an address is on this computer.
func local(base string) bool {
	for _, h := range []string{"//localhost", "//127.0.0.1", "//[::1]"} {
		if strings.Contains(base, h) {
			return true
		}
	}
	return false
}
