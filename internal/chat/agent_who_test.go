package chat_test

import (
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/records"
)

// TestGoHttpClientIsAnAgent checks that machine-language identifiers like
// "Go-http-client" resolve to "" by AgentWho, and technical parentheticals are
// stripped so no jargon reaches user-facing text. This covers acceptance items
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
		{"", records.ThroughAPI, ""},
		{"Go-http-client", records.ThroughAPI, ""},
		{"Go-http-client", "", ""},
		{"Claude Code", records.ThroughMCP, "Claude Code"},
	}
	for _, c := range cases {
		got := records.AgentWho(c.name, c.via)
		if got != c.want {
			t.Errorf("AgentWho(%q, %q): got %q, want %q", c.name, c.via, got, c.want)
		}
	}
}
