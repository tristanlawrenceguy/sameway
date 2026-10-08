package chat

import (
	"encoding/json"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/records"
)

// run executes one tool call and records the change it made, which is the
// whole of what a tool call is from outside: the conversation and the MCP
// server both go through here, so a check or a log entry is never done in
// one place and forgotten in the other.
func (s *Service) run(call llm.ToolCall) toolResult {
	if r, no := s.refuseFor(call); no {
		logCall(call, r)
		return r
	}
	r := s.runOp(call, false)
	r = s.layoutAfter(call.Name, r) // see arrange.go
	logCall(call, r)                // tool_log.go
	if r.change != nil {
		// The receipt keeps the entry id, so the change can be undone from
		// under the reply.
		r.change.Activity = records.Record(s.Store, s.actor(), s.byWho(*r.change))
	}
	for i := range r.changes {
		r.changes[i].Activity = records.Record(s.Store, s.actor(), s.byWho(r.changes[i]))
	}
	return r
}

// Call runs a tool by name for a caller that is not the conversation, and
// returns what the model would have been told and whether it was an error.
func (s *Service) Call(name string, args json.RawMessage) (text string, isError bool) {
	r := s.run(llm.ToolCall{ID: "call", Name: name, Args: args})
	return r.text, r.isErr
}
