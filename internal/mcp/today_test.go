package mcp_test

import (
	"encoding/json"
	"testing"
)

// An agent over MCP reads the person's day with today: listed, read only,
// and the same answer GET /api/today gives.
func TestAnAgentOverMCPReadsToday(t *testing.T) {
	t.Parallel()
	_, replies := drive(t,
		`{"jsonrpc":"2.0","id":1,"method":"tools/list"}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"today","arguments":{}}}`,
	)
	listed := false
	for _, tl := range result(t, replies[0])["tools"].([]any) {
		if m := tl.(map[string]any); m["name"] == "today" {
			ann, _ := m["annotations"].(map[string]any)
			listed = ann["readOnlyHint"] == true
		}
	}
	if !listed {
		t.Error("today is offered over MCP, as read only")
	}
	said, isErr := text(t, replies[1])
	var day struct {
		Day  string `json:"day"`
		Page string `json:"page"`
	}
	if isErr || json.Unmarshal([]byte(said), &day) != nil || day.Page != "/today" || day.Day == "" {
		t.Errorf("today answers with the day as GET /api/today does: %v %q", isErr, said)
	}
}
