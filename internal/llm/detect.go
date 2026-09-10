package llm

import (
	"context"
	"encoding/json"
	"net/http"
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
// returns every server that answered with at least one model.
func Detect(ctx context.Context, candidates []Candidate) []Detected {
	client := &http.Client{Timeout: 1500 * time.Millisecond}
	var found []Detected
	for _, c := range candidates {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+"/models", nil)
		if err != nil {
			continue
		}
		resp, err := client.Do(req)
		if err != nil {
			continue
		}
		var body struct {
			Data []struct {
				ID string `json:"id"`
			} `json:"data"`
		}
		decodeErr := json.NewDecoder(resp.Body).Decode(&body)
		resp.Body.Close()
		if decodeErr != nil || resp.StatusCode != http.StatusOK || len(body.Data) == 0 {
			continue
		}
		d := Detected{Server: c.Server, BaseURL: c.BaseURL}
		for _, m := range body.Data {
			if m.ID != "" {
				d.Models = append(d.Models, m.ID)
			}
		}
		if len(d.Models) == 0 {
			continue
		}
		d.Model = d.Models[0]
		found = append(found, d)
	}
	return found
}
