package chat

import (
	"encoding/json"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
)

// run executes one tool call and records the change it made, which is the
// whole of what a tool call is from outside: the conversation and the MCP
// server both go through here, so a check or a log entry is never done in
// one place and forgotten in the other.
func (s *Service) run(call llm.ToolCall) toolResult {
	r := s.runTool(call)
	if r.change != nil {
		Record(s.Store, "assistant", *r.change)
	}
	return r
}

// Call runs a tool by name for a caller that is not the conversation, and
// returns what the model would have been told and whether it was an error.
func (s *Service) Call(name string, args json.RawMessage) (text string, isError bool) {
	r := s.run(llm.ToolCall{ID: "call", Name: name, Args: args})
	return r.text, r.isErr
}
