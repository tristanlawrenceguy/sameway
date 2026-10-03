package chat_test

import (
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
)

// TestGoHttpClientIsAnAgent checks that machine-language identifiers like
// "Go-http-client" are resolved to "An agent" by AgentWho, so every surface
// that calls it picks up the fix automatically. This covers acceptance items
// 1 and 2 — filter dropdowns, detail ledes, list entries all go through
// AgentWho (directly or via Sentence/whoDid).

func TestGoHttpClientIsAnAgent(t *testing.T) {
	// Acceptance 1 & 2: "Go-http-client" is treated as anonymous.
	cases := []struct {
		name string
		via  string
		want string
	}{
		{"", chat.ThroughAPI, "An agent (through the API)"},
		{"Go-http-client", chat.ThroughAPI, "An agent (through the API)"},
		{"Go-http-client", "", "An agent"},
		{"Claude Code", chat.ThroughMCP, "Claude Code (through MCP)"},
	}
	for _, c := range cases {
		got := chat.AgentWho(c.name, c.via)
		if got != c.want {
			t.Errorf("AgentWho(%q, %q): got %q, want %q", c.name, c.via, got, c.want)
		}
	}
}
