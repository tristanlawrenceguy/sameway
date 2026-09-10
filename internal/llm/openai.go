package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// OpenAI talks to any server implementing the OpenAI chat completions API.
type OpenAI struct {
	BaseURL   string
	Model     string
	APIKey    string
	MaxTokens int
	Client    *http.Client
}

// Name identifies the provider in logs and errors.
func (o *OpenAI) Name() string { return "openai-compatible (" + o.Model + " at " + o.BaseURL + ")" }

type oaMessage struct {
	Role       string       `json:"role"`
	Content    string       `json:"content,omitempty"`
	ToolCalls  []oaToolCall `json:"tool_calls,omitempty"`
	ToolCallID string       `json:"tool_call_id,omitempty"`
}

type oaToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

type oaRequest struct {
	Model     string      `json:"model"`
	Messages  []oaMessage `json:"messages"`
	Tools     []any       `json:"tools,omitempty"`
	MaxTokens int         `json:"max_tokens,omitempty"`
}

type oaResponse struct {
	Choices []struct {
		Message      oaMessage `json:"message"`
		FinishReason string    `json:"finish_reason"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// Complete sends one chat completion request.
func (o *OpenAI) Complete(ctx context.Context, req Request) (*Response, error) {
	body := oaRequest{Model: o.Model, MaxTokens: o.MaxTokens}
	if req.System != "" {
		body.Messages = append(body.Messages, oaMessage{Role: "system", Content: req.System})
	}
	for _, m := range req.Messages {
		body.Messages = append(body.Messages, toOpenAI(m)...)
	}
	for _, t := range req.Tools {
		body.Tools = append(body.Tools, map[string]any{
			"type": "function",
			"function": map[string]any{
				"name":        t.Name,
				"description": t.Description,
				"parameters":  t.Schema,
			},
		})
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, o.BaseURL+"/chat/completions", bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
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
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 32<<20))
	if err != nil {
		return nil, err
	}
	var parsed oaResponse
	if jsonErr := json.Unmarshal(raw, &parsed); jsonErr != nil || resp.StatusCode >= 400 {
		msg := strings.TrimSpace(string(raw))
		if parsed.Error != nil {
			msg = parsed.Error.Message
		}
		if len(msg) > 400 {
			msg = msg[:400] + "…"
		}
		return nil, fmt.Errorf("model server returned %s: %s", resp.Status, msg)
	}
	if len(parsed.Choices) == 0 {
		return nil, fmt.Errorf("model server returned no choices")
	}
	choice := parsed.Choices[0]
	out := &Response{Text: choice.Message.Content, StopReason: choice.FinishReason}
	for _, tc := range choice.Message.ToolCalls {
		args := strings.TrimSpace(tc.Function.Arguments)
		if args == "" {
			args = "{}"
		}
		out.ToolCalls = append(out.ToolCalls, ToolCall{ID: tc.ID, Name: tc.Function.Name, Args: json.RawMessage(args)})
	}
	return out, nil
}

// toOpenAI maps one neutral message onto the wire shape. A tool turn expands
// into one "tool" message per result.
func toOpenAI(m Message) []oaMessage {
	switch m.Role {
	case RoleTool:
		var out []oaMessage
		for _, r := range m.ToolResults {
			out = append(out, oaMessage{Role: "tool", ToolCallID: r.CallID, Content: r.Content})
		}
		return out
	case RoleAssistant:
		msg := oaMessage{Role: "assistant", Content: m.Content}
		for _, tc := range m.ToolCalls {
			var call oaToolCall
			call.ID, call.Type = tc.ID, "function"
			call.Function.Name = tc.Name
			call.Function.Arguments = string(tc.Args)
			msg.ToolCalls = append(msg.ToolCalls, call)
		}
		return []oaMessage{msg}
	default:
		return []oaMessage{{Role: "user", Content: m.Content}}
	}
}
