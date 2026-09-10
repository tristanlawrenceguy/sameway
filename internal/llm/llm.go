// Package llm is a small provider-neutral chat interface with tool calling.
//
// Two providers ship: "openai" speaks the OpenAI-compatible chat completions
// API used by Ollama, LM Studio, llama.cpp, OpenRouter, and OpenAI itself;
// "anthropic" uses the official Anthropic SDK. The chat package drives the
// tool loop, so providers only translate one request and one response.
package llm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
)

// Role of a message in the conversation.
type Role string

const (
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleTool      Role = "tool"
)

// Message is one turn. An assistant turn may carry tool calls; a tool turn
// carries the results for them.
type Message struct {
	Role        Role
	Content     string
	ToolCalls   []ToolCall
	ToolResults []ToolResult
}

// ToolCall is a request from the model to run a tool.
type ToolCall struct {
	ID   string
	Name string
	Args json.RawMessage
}

// ToolResult is what a tool returned for one call.
type ToolResult struct {
	CallID  string
	Content string
	IsError bool
}

// Tool describes a callable tool with a JSON Schema for its input.
type Tool struct {
	Name        string
	Description string
	Schema      map[string]any
}

// Request is one model call.
type Request struct {
	System    string
	Messages  []Message
	Tools     []Tool
	MaxTokens int
}

// Response is the model's reply: text, tool calls, or both.
type Response struct {
	Text       string
	ToolCalls  []ToolCall
	StopReason string
}

// Provider is implemented by each backend.
type Provider interface {
	Name() string
	Complete(ctx context.Context, req Request) (*Response, error)
}

// Config is the llm section of workspace.yaml.
type Config struct {
	// Provider is "openai" (any OpenAI-compatible server), "anthropic", or "none".
	Provider string `yaml:"provider" json:"provider"`
	// BaseURL for openai providers, for example http://localhost:11434/v1.
	BaseURL string `yaml:"base_url,omitempty" json:"base_url,omitempty"`
	Model   string `yaml:"model" json:"model"`
	// APIKeyEnv names the environment variable holding the key. Keys never
	// live in workspace.yaml because that file is shared.
	APIKeyEnv string `yaml:"api_key_env,omitempty" json:"api_key_env,omitempty"`
	MaxTokens int    `yaml:"max_tokens,omitempty" json:"max_tokens,omitempty"`
}

// ErrNotConfigured is returned by New when provider is "none" or empty.
var ErrNotConfigured = errors.New("no model configured: set llm.provider in workspace.yaml")

// New builds a provider from config.
func New(cfg Config) (Provider, error) {
	if cfg.MaxTokens == 0 {
		cfg.MaxTokens = 4096
	}
	key := ""
	if cfg.APIKeyEnv != "" {
		key = os.Getenv(cfg.APIKeyEnv)
	}
	switch strings.ToLower(cfg.Provider) {
	case "", "none":
		return nil, ErrNotConfigured
	case "openai", "openai-compatible", "ollama", "lmstudio", "openrouter":
		if cfg.Model == "" {
			return nil, errors.New("llm.model is required")
		}
		base := cfg.BaseURL
		if base == "" {
			base = "http://localhost:11434/v1"
		}
		return &OpenAI{BaseURL: strings.TrimRight(base, "/"), Model: cfg.Model, APIKey: key, MaxTokens: cfg.MaxTokens}, nil
	case "anthropic":
		model := cfg.Model
		if model == "" {
			model = "claude-opus-5"
		}
		return NewAnthropic(model, key, cfg.MaxTokens), nil
	}
	return nil, fmt.Errorf("unknown llm.provider %q (use openai, anthropic, or none)", cfg.Provider)
}
