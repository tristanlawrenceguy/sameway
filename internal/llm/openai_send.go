package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/trim"
)

// retryWaits are the pauses before sending again when a provider is busy
// or limiting: what a person would do, by hand, and did, after reading the
// error. Anthropic's own client does the same for Claude.
var retryWaits = []time.Duration{2 * time.Second, 6 * time.Second}

// send posts a chat completion and answers the response when it succeeded,
// sending again after a pause while the provider is busy or limiting, and
// otherwise a Failure, sorted (failure.go).
func (o *OpenAI) send(ctx context.Context, payload []byte, stream bool) (*http.Response, error) {
	client := o.Client
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Minute}
	}
	for attempt := 0; ; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, o.BaseURL+"/chat/completions", bytes.NewReader(payload))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
		if stream {
			req.Header.Set("Accept", "text/event-stream")
		}
		if o.APIKey != "" {
			req.Header.Set("Authorization", "Bearer "+o.APIKey)
		}
		resp, err := client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("the AI model at %s isn't answering; it may not be running (%w)", o.BaseURL, err)
		}
		if resp.StatusCode < 400 {
			return resp, nil
		}
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		resp.Body.Close()
		var parsed oaResponse
		msg := strings.TrimSpace(string(raw))
		if json.Unmarshal(raw, &parsed) == nil && parsed.Error != nil {
			msg = parsed.Error.Message
		}
		msg = trim.Clip(msg, 400)
		f := Classify(resp.StatusCode, msg, whoAt(o.BaseURL), o.Model)
		if !f.Retry() || attempt >= len(retryWaits) {
			return nil, f
		}
		wait := retryWaits[attempt]
		if s, err := strconv.Atoi(resp.Header.Get("Retry-After")); err == nil && s > 0 && s <= 20 {
			wait = time.Duration(s) * time.Second
		}
		select {
		case <-ctx.Done():
			return nil, f
		case <-time.After(wait):
		}
	}
}
