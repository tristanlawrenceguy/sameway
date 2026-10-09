package mcp_test

import (
	"strings"
	"testing"
)

// An agent over MCP has arrange_canvas and gets the layout line, from the
// same list and the same code as the assistant.
func TestAnAgentOverMCPArrangesAndIsToldTheLayout(t *testing.T) {
	t.Parallel()
	_, replies := drive(t,
		`{"jsonrpc":"2.0","id":1,"method":"tools/list"}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"add_component","arguments":{"component":"heading","props":{"text":"Week"},"span":6}}}`,
		`{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"arrange_canvas","arguments":{"blocks":[]}}}`,
	)
	listed := false
	for _, tl := range result(t, replies[0])["tools"].([]any) {
		if tl.(map[string]any)["name"] == "arrange_canvas" {
			listed = true
		}
	}
	if !listed {
		t.Error("arrange_canvas should be offered over MCP")
	}
	said, isErr := text(t, replies[1])
	if isErr || !strings.Contains(said, `Layout now: row 1: "Week" heading 6 (6 of 12 empty)`) || !strings.Contains(said, "arrange_canvas {") {
		t.Errorf("a block written over MCP should say how the page reads, got %v %q", isErr, said)
	}
	said, isErr = text(t, replies[2])
	if !isErr || !strings.Contains(said, "is missing; list every block") {
		t.Errorf("an arrangement that leaves a block out is refused, got %v %q", isErr, said)
	}
}
