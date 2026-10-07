package llm

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
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
	Content    any          `json:"content,omitempty"` // a string, or parts when there are pictures
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
	Stream    bool        `json:"stream,omitempty"`
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
	payload, err := json.Marshal(o.body(req))
	if err != nil {
		return nil, err
	}
	resp, err := o.send(ctx, payload, false) // openai_send.go
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 32<<20))
	if err != nil {
		return nil, err
	}
	var parsed oaResponse
	if jsonErr := json.Unmarshal(raw, &parsed); jsonErr != nil {
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
	text, _ := choice.Message.Content.(string)
	out := &Response{Text: text, StopReason: choice.FinishReason}
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
		if len(m.Images) == 0 {
			return []oaMessage{{Role: "user", Content: m.Content}}
		}
		// Pictures go as parts, which OpenAI and the local servers that
		// follow it (Ollama, LM Studio, llama.cpp) take for a model that sees.
		parts := []map[string]any{}
		for _, img := range m.Images {
			parts = append(parts, map[string]any{"type": "image_url", "image_url": map[string]any{"url": "data:" + img.Type + ";base64," + base64.StdEncoding.EncodeToString(img.Data)}})
		}
		parts = append(parts, map[string]any{"type": "text", "text": m.Content})
		return []oaMessage{{Role: "user", Content: parts}}
	}
}

// body is the request on the wire, the same whether it streams or not.
func (o *OpenAI) body(req Request) oaRequest {
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
	return body
}
