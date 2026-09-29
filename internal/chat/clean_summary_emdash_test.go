package chat_test

import (
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
)

// Em-dash field descriptions in undo summaries are stripped before key lookup,
// so "pace — how changes arrive" resolves to "pace". This covers acceptance
// item 1 for the HTML path (activity page headings).
func TestCleanSummaryEmDashPropertyNames(t *testing.T) {
	for in, want := range map[string]string{
		"You undid: Assistant changed Pace — how changes arrive to Calmly":    "You undid: Assistant changed pace to Calmly",
		"Assistant changed text size — the length of words from Short to Big": "Assistant changed text size from Short to Big",
	} {
		if got := chat.CleanSummary(in); got != want {
			t.Errorf("CleanSummary(%q)\ngot:  %q\nwant: %q", in, got, want)
		}
	}
}
