package records_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/records"
)

// EventGoldenNoAgentJargon checks that the event component golden examples do not contain
// "An agent", "(through the API)", or similar machine-language phrases. (Acceptance 1.)
func TestEventGoldenNoAgentJargon(t *testing.T) {
	t.Parallel()

	examplesDir := filepath.Join("..", "..", "design", "components", "event", "examples")
	entries, err := os.ReadDir(examplesDir)
	if err != nil {
		t.Fatalf("read event examples: %v", err)
	}

	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".html") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(examplesDir, entry.Name()))
		if err != nil {
			t.Fatalf("read %s: %v", entry.Name(), err)
		}
		content := string(data)

		// Check for machine-language phrases that should not appear.
		for _, bad := range []string{"An agent", "(through the API)", "(through MCP)", "(through the command line)"} {
			if strings.Contains(content, bad) {
				t.Errorf("%s: contains jargon %q (acceptance 1: no 'agent' or technical jargon in activity entries)", entry.Name(), bad)
			}
		}
	}
}

// AgentNameNormalizesUnknownClients checks that unknown agent names are cleaned but kept.
func TestAgentNameNormalizesUnknownClients(t *testing.T) {
	t.Parallel()

	got := records.AgentName("My Custom Bot")
	if got != "My Custom Bot" {
		t.Errorf("unknown client: got %q, want unchanged", got)
	}

	// Machine names are still returned as-is by AgentName (AgentWho handles the mapping).
	got = records.AgentName("Go-http-client/1.1")
	if got != "Go-http-client" {
		t.Errorf("machine name cleaned: got %q, want truncated", got)
	}

	// Known clients map to product names.
	got = records.AgentName("claude-code")
	want := "Claude Code"
	if got != want {
		t.Errorf("known client claude-code: got %q, want %q", got, want)
	}
}

// AgentWhoAgentTypeConstant checks that ActorAgent is the constant used for agent actors.
func TestAgentWhoAgentTypeConstant(t *testing.T) {
	t.Parallel()

	if records.ActorAgent != "agent" {
		t.Errorf("ActorAgent: got %q, want %q", records.ActorAgent, "agent")
	}
}

// ThroughMCPConstant checks the MCP constant value.
func TestThroughMCPConstant(t *testing.T) {
	t.Parallel()

	if records.ThroughMCP != "through MCP" {
		t.Errorf("ThroughMCP: got %q, want %q", records.ThroughMCP, "through MCP")
	}
}

// ThroughAPIConstant checks the API constant value.
func TestThroughAPIConstant(t *testing.T) {
	t.Parallel()

	if records.ThroughAPI != "through the API" {
		t.Errorf("ThroughAPI: got %q, want %q", records.ThroughAPI, "through the API")
	}
}
