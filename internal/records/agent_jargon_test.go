package records_test

import (
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/records"
)

// AgentWhoUnnamedAgentReturnsEmpty checks that AgentWho returns "" for
// unnamed agents and machine names, so activity log entries start with the
// action rather than "An agent". (Acceptance 1 — no word "agent" in entry)
func TestAgentWhoUnnamedAgentReturnsEmpty(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		through  string
		wantName string // what AgentWho returns for the name part alone
	}{
		{"", records.ThroughAPI, ""},
		{"Go-http-client", records.ThroughAPI, ""},
		{"", "", ""},
	}
	for _, c := range cases {
		got := records.AgentWho(c.name, "")
		if got != "" {
			t.Errorf("AgentWho(%q, \"\") for unnamed agent: got %q, want empty string (no 'agent' word)", c.name, got)
		}
	}
}

// AgentWhoStripsTechnicalParentheticals checks that "(through the API)",
// "(through MCP)" and "(through the command line)" are removed from output.
// (Acceptance 1 — no "API" or technical jargon in entry text)
func TestAgentWhoStripsTechnicalParentheticals(t *testing.T) {
	t.Parallel()

	// Unnamed agent with technical through-values returns "" (no actor label).
	got := records.AgentWho("", records.ThroughAPI)
	if got != "" {
		t.Errorf("unnamed + API: got %q, want empty (acceptance 1: no 'agent' or 'API')", got)
	}

	got = records.AgentWho("", "through MCP")
	if got != "" {
		t.Errorf("unnamed + MCP: got %q, want empty (acceptance 1: no 'agent' or 'MCP')", got)
	}

	got = records.AgentWho("", "through the command line")
	if got != "" {
		t.Errorf("unnamed + CLI: got %q, want empty (acceptance 1: no 'command line')", got)
	}
}

// AgentWhoNamedKeepsNameNoParenthetical checks that named agents like
// "Claude Code" keep their name but without "(through X)". (Acceptance 1 —
// proper nouns are fine, technical parentheticals are not.)
func TestAgentWhoNamedKeepsNameNoParenthetical(t *testing.T) {
	t.Parallel()

	got := records.AgentWho("Claude Code", records.ThroughMCP)
	want := "Claude Code"
	if got != want {
		t.Errorf("named agent + MCP: got %q, want %q (no parenthetical)", got, want)
	}

	got = records.AgentWho("backup service", records.ThroughAPI)
	want = "backup service"
	if got != want {
		t.Errorf("named agent + API: got %q, want %q (no parenthetical)", got, want)
	}

	// Non-technical through-value should still be appended.
	got = records.AgentWho("Claude Code", "on pixel-7")
	want = "Claude Code (on pixel-7)"
	if got != want {
		t.Errorf("named agent + device: got %q, want %q", got, want)
	}

	// No through-value at all.
	got = records.AgentWho("Claude Code", "")
	want = "Claude Code"
	if got != want {
		t.Errorf("named agent no through: got %q, want %q", got, want)
	}
}

// SentenceAgentEntryHasNoJargon checks that the full sentence for an
// agent-created activity entry contains neither "agent" nor "API".
// (Acceptance 1 — direct assertion on stored summary text)
func TestSentenceAgentEntryHasNoJargon(t *testing.T) {
	t.Parallel()
	st := newTestStore(t)

	fields := map[string]any{
		"actor":  records.ActorAgent,
		"by":     "",
		"via":    records.ThroughAPI,
		"action": "created",
		"target": "note",
		"detail": "Plan",
	}

	got := records.Sentence(st, fields)
	if containsJargon(got) {
		t.Errorf("agent-created entry should not contain 'agent' or 'API': got %q", got)
	}
}

// SentenceNamedAgentEntryHasNoTechnicalParenthetical checks that a named
// agent's entry says "Claude Code created note Plan" without "(through MCP)".
func TestSentenceNamedAgentEntryHasNoTechnicalParenthetical(t *testing.T) {
	t.Parallel()
	st := newTestStore(t)

	fields := map[string]any{
		"actor":  records.ActorAgent,
		"by":     "Claude Code",
		"via":    records.ThroughMCP,
		"action": "created",
		"target": "note",
		"detail": "Plan",
	}

	got := records.Sentence(st, fields)
	if got != "Claude Code created note Plan" {
		t.Errorf("named agent entry: got %q, want %q (no parenthetical)", got, "Claude Code created note Plan")
	}
}

// SentenceUnamedAgentEntryStartsWithAction checks that an unnamed agent's
// activity log entry starts with the action verb, not a label.
func TestSentenceUnnamedAgentEntryStartsWithAction(t *testing.T) {
	t.Parallel()
	st := newTestStore(t)

	fields := map[string]any{
		"actor":  records.ActorAgent,
		"by":     "",
		"via":    "through the command line",
		"action": "updated",
		"target": "task",
		"detail": "Call plumber",
	}

	got := records.Sentence(st, fields)
	if got != "updated task Call plumber" {
		t.Errorf("unnamed agent updated: got %q, want %q (starts with action)", got, "updated task Call plumber")
	}
	if containsJargon(got) {
		t.Errorf("entry must not contain jargon words: got %q", got)
	}
}

// SentenceAgentUpdatedMultipleRecords checks the plural form for agent updates.
func TestSentenceAgentUpdatedMultipleRecords(t *testing.T) {
	t.Parallel()
	st := newTestStore(t)

	fields := map[string]any{
		"actor":  records.ActorAgent,
		"by":     "",
		"via":    records.ThroughAPI,
		"action": "updated",
		"target": "tasks",
		"detail": "2 tasks: Test my finding, Study for biology exam",
	}

	got := records.Sentence(st, fields)
	if containsJargon(got) {
		t.Errorf("agent update entry should not contain 'agent' or 'API': got %q", got)
	}
}

// containsJargon reports whether a string contains the words "agent" (case-insensitive) or "API".
func containsJargon(s string) bool {
	lower := ""
	for _, r := range s {
		if r >= 'A' && r <= 'Z' {
			lower += string(r + ('a' - 'A'))
		} else {
			lower += string(r)
		}
	}
	return contains(lower, "agent") || contains(lower, "api")
}

func contains(s, substr string) bool {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
