package llm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

// Anthropic calls the Claude API through the official SDK.
type Anthropic struct {
	Model     string
	MaxTokens int
	client    anthropic.Client
}

// NewAnthropic builds a provider. An empty key falls back to the SDK's own
// credential lookup (ANTHROPIC_API_KEY or an `ant auth login` profile).
func NewAnthropic(model, apiKey string, maxTokens int) *Anthropic {
	var opts []option.RequestOption
	if apiKey != "" {
		opts = append(opts, option.WithAPIKey(apiKey))
	}
	return &Anthropic{Model: model, MaxTokens: maxTokens, client: anthropic.NewClient(opts...)}
}

// Name identifies the provider in logs and errors.
func (a *Anthropic) Name() string { return "anthropic (" + a.Model + ")" }

// Complete sends one Messages API request. Thinking is left at the model's
// default (adaptive on current models).
func (a *Anthropic) Complete(ctx context.Context, req Request) (*Response, error) {
	params := anthropic.MessageNewParams{
		Model:     anthropic.Model(a.Model),
		MaxTokens: int64(a.MaxTokens),
	}
	if req.System != "" {
		params.System = []anthropic.TextBlockParam{{Text: req.System}}
	}
	for _, m := range req.Messages {
		params.Messages = append(params.Messages, toAnthropic(m))
	}
	for _, t := range req.Tools {
		tool := anthropic.ToolParam{
			Name:        t.Name,
			Description: anthropic.String(t.Description),
			InputSchema: toolSchema(t.Schema),
		}
		params.Tools = append(params.Tools, anthropic.ToolUnionParam{OfTool: &tool})
	}
	resp, err := a.client.Messages.New(ctx, params)
	if err != nil {
		var apiErr *anthropic.Error
		if errors.As(err, &apiErr) {
			return nil, fmt.Errorf("anthropic returned HTTP %d: %s", apiErr.StatusCode, apiErr.Error())
		}
		return nil, fmt.Errorf("could not reach the Claude API: %w", err)
	}
	if resp.StopReason == anthropic.StopReasonRefusal {
		return nil, fmt.Errorf("the model declined this request (%s)", resp.StopDetails.Explanation)
	}
	out := &Response{StopReason: string(resp.StopReason)}
	for _, block := range resp.Content {
		switch b := block.AsAny().(type) {
		case anthropic.TextBlock:
			out.Text += b.Text
		case anthropic.ToolUseBlock:
			out.ToolCalls = append(out.ToolCalls, ToolCall{ID: b.ID, Name: b.Name, Args: json.RawMessage(b.JSON.Input.Raw())})
		}
	}
	return out, nil
}

func toAnthropic(m Message) anthropic.MessageParam {
	switch m.Role {
	case RoleAssistant:
		var blocks []anthropic.ContentBlockParamUnion
		if m.Content != "" {
			blocks = append(blocks, anthropic.NewTextBlock(m.Content))
		}
		for _, tc := range m.ToolCalls {
			blocks = append(blocks, anthropic.NewToolUseBlock(tc.ID, tc.Args, tc.Name))
		}
		return anthropic.NewAssistantMessage(blocks...)
	case RoleTool:
		var blocks []anthropic.ContentBlockParamUnion
		for _, r := range m.ToolResults {
			blocks = append(blocks, anthropic.NewToolResultBlock(r.CallID, r.Content, r.IsError))
		}
		return anthropic.NewUserMessage(blocks...)
	default:
		return anthropic.NewUserMessage(anthropic.NewTextBlock(m.Content))
	}
}

func toolSchema(schema map[string]any) anthropic.ToolInputSchemaParam {
	out := anthropic.ToolInputSchemaParam{}
	if props, ok := schema["properties"]; ok {
		out.Properties = props
	}
	if req, ok := schema["required"].([]string); ok {
		out.Required = req
	}
	return out
}
