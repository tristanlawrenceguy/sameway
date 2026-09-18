package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Streamer is a provider that can say its reply as it comes. delta gets
// each piece of text in order; the whole reply comes back as usual, tool
// calls included, so a turn runs the same way whether it streamed or not.
type Streamer interface {
	Stream(ctx context.Context, req Request, delta func(text string)) (*Response, error)
}

// oaChunk is one server-sent piece of a streamed completion.
type oaChunk struct {
	Choices []struct {
		Delta struct {
			Content   string `json:"content"`
			ToolCalls []struct {
				Index    int    `json:"index"`
				ID       string `json:"id"`
				Function struct {
					Name      string `json:"name"`
					Arguments string `json:"arguments"`
				} `json:"function"`
			} `json:"tool_calls"`
		} `json:"delta"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// Stream asks for the completion as server-sent events and hands each
// piece of text on as it arrives. Tool calls come in pieces too, by index,
// and are put back together here.
func (o *OpenAI) Stream(ctx context.Context, req Request, delta func(string)) (*Response, error) {
	body := o.body(req)
	body.Stream = true
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, o.BaseURL+"/chat/completions", bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")
	if o.APIKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+o.APIKey)
	}
	client := o.Client
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Minute}
	}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("could not reach the model at %s: %w", o.BaseURL, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		var parsed oaResponse
		msg := strings.TrimSpace(string(raw))
		if json.Unmarshal(raw, &parsed) == nil && parsed.Error != nil {
			msg = parsed.Error.Message
		}
		if len(msg) > 400 {
			msg = msg[:400] + "…"
		}
		return nil, fmt.Errorf("model server returned %s: %s", resp.Status, msg)
	}
	out := &Response{}
	var text strings.Builder
	calls := map[int]*ToolCall{}
	args := map[int]*strings.Builder{}
	order := []int{}
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64<<10), 8<<20)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "[DONE]" {
			break
		}
		var chunk oaChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}
		if chunk.Error != nil {
			return nil, fmt.Errorf("model server: %s", chunk.Error.Message)
		}
		if len(chunk.Choices) == 0 {
			continue
		}
		c := chunk.Choices[0]
		if c.Delta.Content != "" {
			text.WriteString(c.Delta.Content)
			if delta != nil {
				delta(c.Delta.Content)
			}
		}
		for _, tc := range c.Delta.ToolCalls {
			call, ok := calls[tc.Index]
			if !ok {
				call = &ToolCall{}
				calls[tc.Index], args[tc.Index] = call, &strings.Builder{}
				order = append(order, tc.Index)
			}
			if tc.ID != "" {
				call.ID = tc.ID
			}
			if tc.Function.Name != "" {
				call.Name += tc.Function.Name
			}
			args[tc.Index].WriteString(tc.Function.Arguments)
		}
		if c.FinishReason != "" {
			out.StopReason = c.FinishReason
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("the model's reply broke off: %w", err)
	}
	out.Text = text.String()
	for _, i := range order {
		a := strings.TrimSpace(args[i].String())
		if a == "" {
			a = "{}"
		}
		call := *calls[i]
		if call.ID == "" {
			call.ID = fmt.Sprintf("call_%d", i)
		}
		call.Args = json.RawMessage(a)
		out.ToolCalls = append(out.ToolCalls, call)
	}
	return out, nil
}
