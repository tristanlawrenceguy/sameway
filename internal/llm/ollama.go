package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"strings"
	"time"
)

// Ollama runs models on a person's own computer, free, and its installer is
// signed, so Windows lets it run where an unsigned model server is blocked
// outright. Once it is installed, Sameway does the rest through its API:
// fetches a model, and makes a copy of it with room for Sameway's prompt.
// Ollama gives a model a few thousand tokens of room by default and cuts
// what does not fit without a word; Sameway's prompt and tools are some
// twenty thousand, and a model given a third of them answered a request it
// never saw.

// OllamaURL is where Ollama listens on this computer, its own API rather
// than /v1.
const OllamaURL = "http://127.0.0.1:11434"

// Ollama is Ollama at an address. The zero value is this computer's; an
// app is given its own through app.Options, so a test points one at a
// fake while others run beside it.
type Ollama struct {
	// URL is its own API, without /v1; "" is OllamaURL.
	URL string
	// Start is how to start it when it is installed and not answering;
	// nil is how it is installed here (ollama_wake.go).
	Start func() *exec.Cmd
}

// Base is the address of its own API.
func (o Ollama) Base() string {
	if o.URL == "" {
		return OllamaURL
	}
	return strings.TrimRight(o.URL, "/")
}

// V1 is its OpenAI-compatible address, the one a workspace's model uses.
func (o Ollama) V1() string { return o.Base() + "/v1" }

// FreeModel is the model Sameway fetches for a person with none: small
// enough for an ordinary computer, and able to use tools.
const FreeModel = "qwen3.5:4b"

// FreeModelWords says FreeModel and its size where a person reads it.
const FreeModelWords = "Qwen 3.5, 3.3 GB"

// roomy is how much a model is given, in tokens.
const roomy = 32768

// Running says whether Ollama answers.
func (o Ollama) Running(ctx context.Context) bool {
	client := &http.Client{Timeout: 1500 * time.Millisecond}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, o.Base()+"/api/version", nil)
	if err != nil {
		return false
	}
	res, err := client.Do(req)
	if err != nil {
		return false
	}
	res.Body.Close()
	return res.StatusCode == http.StatusOK
}

// Is says whether a base URL is this Ollama's.
func (o Ollama) Is(base string) bool {
	return strings.HasPrefix(strings.TrimRight(base, "/"), o.Base())
}

// Pull fetches a model, telling progress how far it has come, in bytes of
// the whole.
func (o Ollama) Pull(ctx context.Context, model string, progress func(done, total int64)) error {
	body, _ := json.Marshal(map[string]any{"model": model, "stream": true})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, o.Base()+"/api/pull", bytes.NewReader(body))
	if err != nil {
		return err
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("Ollama did not answer: %w", err)
	}
	defer res.Body.Close()
	lines := bufio.NewScanner(res.Body)
	lines.Buffer(make([]byte, 64*1024), 1024*1024)
	for lines.Scan() {
		var step struct {
			Status    string `json:"status"`
			Total     int64  `json:"total"`
			Completed int64  `json:"completed"`
			Error     string `json:"error"`
		}
		if json.Unmarshal(lines.Bytes(), &step) != nil {
			continue
		}
		if step.Error != "" {
			return fmt.Errorf("Ollama could not fetch %s: %s", model, step.Error)
		}
		if step.Total > 0 && progress != nil {
			progress(step.Completed, step.Total)
		}
		if step.Status == "success" {
			return nil
		}
	}
	if err := lines.Err(); err != nil {
		return err
	}
	return fmt.Errorf("Ollama stopped before %s was fetched", model)
}

// Roomy is Sameway's copy of an Ollama model, with room for its
// prompt, made the first time it is asked for: "sameway-qwen3.5" for
// qwen3.5:4b. A model already Sameway's is itself.
func (o Ollama) Roomy(ctx context.Context, model string) (string, error) {
	if strings.HasPrefix(model, "sameway-") {
		return model, nil
	}
	name := "sameway-" + strings.SplitN(model, ":", 2)[0]
	body, _ := json.Marshal(map[string]any{"model": name, "from": model, "parameters": map[string]any{"num_ctx": roomy}, "stream": false})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, o.Base()+"/api/create", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("Ollama did not answer: %w", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Ollama could not give %s room for Sameway's prompt (%s)", model, res.Status)
	}
	return name, nil
}

// Keep asks Ollama to keep a model loaded for a while, "30m": it
// unloads one after five minutes unused by default, and loading it again
// costs the next message seconds (internal/chat warm.go).
func (o Ollama) Keep(ctx context.Context, model, keep string) error {
	body, _ := json.Marshal(map[string]any{"model": model, "keep_alive": keep})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, o.Base()+"/api/generate", bytes.NewReader(body))
	if err != nil {
		return err
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	res.Body.Close()
	return nil
}
