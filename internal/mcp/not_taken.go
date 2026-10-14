package mcp

import (
	"encoding/json"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
)

// MCP's own tools (describe, look, try) refuse an argument they do not
// take, as the assistant's tools do, in the same words (chat.NotTaken):
// describe with nme was answered with the whole index, as if nothing had
// been asked.

// notTaken is why one of MCP's own tools cannot run as sent, "" when it
// can, or when name is the assistant's, which says it itself.
func (s *Server) notTaken(name string, args json.RawMessage) string {
	if name != "describe" && name != "look" && name != "try" {
		return ""
	}
	var sent map[string]any
	if json.Unmarshal(args, &sent) != nil {
		return ""
	}
	for _, t := range append(s.tools()[:2:2], tryTool) {
		if t.Name == name {
			props, _ := t.InputSchema["properties"].(map[string]any)
			return chat.NotTaken(name, props, sent)
		}
	}
	return ""
}
