package mcp

import (
	"encoding/json"
	"slices"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
)

// MCP's own tools refuse an argument they do not
// take, as the assistant's tools do, in the same words (chat.NotTaken):
// describe with nme was answered with the whole index, as if nothing had
// been asked.

// notTaken is why one of MCP's own tools cannot run as sent, "" when it
// can, or when name is the assistant's, which says it itself.
func notTaken(name string, args json.RawMessage) string {
	for _, t := range append(slices.Clone(readTools), tryTool) {
		var sent map[string]any
		if t.Name == name && json.Unmarshal(args, &sent) == nil {
			props, _ := t.InputSchema["properties"].(map[string]any)
			return chat.NotTaken(name, props, sent)
		}
	}
	return ""
}
