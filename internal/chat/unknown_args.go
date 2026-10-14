package chat

import (
	"encoding/json"
	"maps"
	"slices"

	"github.com/tristanlawrenceguy/sameway/internal/records"
)

// An argument a tool does not take is refused, never ignored: a call to
// create_record with fields and a stray canvas made the record and said
// nothing of the canvas, so the agent took it as placed. What the tool
// does take is said, with the one likely meant, as the API says a query
// word it does not read (records.NotTaken). The model, MCP and
// POST /api/tools all read it, since all of them run through here.
// Anthropic's tool guidance: an error says what to change, not only that
// something failed.

// unknownArgs is why a call cannot run as sent, "" when it can: the
// arguments its tool, as this workspace offers it, does not take. A tool
// whose schema names no properties, or allows more, takes anything. A
// call that names a tool (a question's action, or propose_change carrying
// one) takes that tool's arguments too: checked here, since what the
// person agrees to runs as it was asked.
func (s *Service) unknownArgs(op Op, raw json.RawMessage) string {
	var args map[string]any
	if json.Unmarshal(raw, &args) != nil {
		return ""
	}
	props, ok := s.props(op)
	if !ok {
		return ""
	}
	if name, _ := args["tool"].(string); name != "" {
		if other, ok := OpFor(name); ok {
			more, _ := s.props(other)
			props = maps.Clone(props)
			maps.Copy(props, more)
			props["tool"] = true
		}
	}
	return NotTaken(op.Name, props, args)
}

// NotTaken is why a call to name with args cannot run, "" when it can:
// the arguments not among props, with the one likely meant, and every
// one it takes. MCP's own tools say it too (mcp/not_taken.go).
func NotTaken(name string, props, args map[string]any) string {
	var unknown, takes []string
	for k := range args {
		if _, ok := props[k]; !ok {
			unknown = append(unknown, k)
		}
	}
	if len(unknown) == 0 {
		return ""
	}
	for k := range props {
		takes = append(takes, k)
	}
	slices.Sort(takes)
	return records.NotTaken(name, unknown, takes) + ". Nothing was done: call it again with only those"
}

// props is what a tool, as this workspace offers it, takes, and whether
// it says: false for one that takes anything.
func (s *Service) props(op Op) (map[string]any, bool) {
	t := op.Tool
	if op.Offered != nil {
		op.Offered(s, &t)
	}
	props, ok := t.Schema["properties"].(map[string]any)
	if more, set := t.Schema["additionalProperties"]; !ok || len(props) == 0 || set && more != false {
		return nil, false
	}
	return props, true
}
