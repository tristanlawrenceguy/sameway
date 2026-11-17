package chat

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// Any agent that says what it can be asked in an A2A agent card, at its
// well-known address, is asked the way the Agent2Agent protocol asks:
// message/send, to the address its card names, which must be on the host
// the owner said yes to sending to. A Sameway has one too (server
// agents_front.go), and gives back the requests it put to its owner, which
// are followed here as any (asks.go). What comes back is data.

// agentAt is where host's agent takes messages, from its agent card, or
// "" when it has none, or its card points off the host.
func agentAt(ctx context.Context, client *http.Client, host string) string {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://"+host+"/.well-known/agent-card.json", nil)
	if err != nil {
		return ""
	}
	res, err := client.Do(req)
	if err != nil {
		return ""
	}
	defer res.Body.Close()
	var card struct {
		URL string `json:"url"`
	}
	if res.StatusCode != http.StatusOK || json.NewDecoder(io.LimitReader(res.Body, 64<<10)).Decode(&card) != nil {
		return ""
	}
	u, err := url.Parse(card.URL)
	if err != nil || u.Scheme != "https" || u.Host != host {
		return ""
	}
	return u.String()
}

// askA2A sends a question as message/send, and gives the words that came
// back and the requests it carried, as {"requests": [...]}.
func askA2A(ctx context.Context, client *http.Client, at, question, who string) (string, []byte, error) {
	id := make([]byte, 8)
	rand.Read(id)
	text := question
	if who != "" {
		text += "\n\n(From " + who + ".)"
	}
	body, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 1, "method": "message/send", "params": map[string]any{"message": map[string]any{
		"role": "user", "messageId": hex.EncodeToString(id), "parts": []any{map[string]any{"kind": "text", "text": text}}, "metadata": map[string]any{"who": who}}}})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, at, bytes.NewReader(body))
	if err != nil {
		return "", nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := client.Do(req)
	if err != nil {
		return "", nil, err
	}
	defer res.Body.Close()
	type part struct {
		Kind string          `json:"kind"`
		Text string          `json:"text"`
		Data json.RawMessage `json:"data"`
	}
	var out struct {
		Result struct {
			Parts  []part `json:"parts"`
			Status struct {
				Message struct {
					Parts []part `json:"parts"`
				} `json:"message"`
			} `json:"status"`
			Artifacts []struct {
				Parts []part `json:"parts"`
			} `json:"artifacts"`
		} `json:"result"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if json.NewDecoder(io.LimitReader(res.Body, 256<<10)).Decode(&out) != nil {
		return "", nil, fmt.Errorf("it answered %s, not as an agent does", res.Status)
	}
	if out.Error != nil {
		return "", nil, fmt.Errorf("it said: %s", out.Error.Message)
	}
	parts := append(append([]part{}, out.Result.Parts...), out.Result.Status.Message.Parts...)
	for _, a := range out.Result.Artifacts {
		parts = append(parts, a.Parts...)
	}
	var words []string
	var requests []byte
	for _, p := range parts {
		switch {
		case p.Kind == "text" && strings.TrimSpace(p.Text) != "":
			words = append(words, p.Text)
		case p.Kind == "data" && requests == nil:
			requests = p.Data
		}
	}
	return strings.Join(words, "\n\n"), requests, nil
}
