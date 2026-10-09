package chat_test

import (
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/records"
)

// TestGoHttpClientIsAnAgent checks that machine-language identifiers like
// "Go-http-client" are resolved to "An agent" by AgentWho, so every surface
// that calls it picks up the fix automatically. This covers acceptance items
// 1 and 2 — filter dropdowns, detail ledes, list entries all go through
// AgentWho (directly or via Sentence/whoDid).

func TestGoHttpClientIsAnAgent(t *testing.T) {
	t.Parallel()
	// Acceptance 1 & 2: "Go-http-client" is treated as anonymous.
	cases := []struct {
		name string
		via  string
		want string
	}{
		{"", records.ThroughAPI, "An agent (through the API)"},
		{"Go-http-client", records.ThroughAPI, "An agent (through the API)"},
		{"Go-http-client", "", "An agent"},
		{"Claude Code", records.ThroughMCP, "Claude Code (through MCP)"},
	}
	for _, c := range cases {
		got := records.AgentWho(c.name, c.via)
		if got != c.want {
			t.Errorf("AgentWho(%q, %q): got %q, want %q", c.name, c.via, got, c.want)
		}
	}
}
