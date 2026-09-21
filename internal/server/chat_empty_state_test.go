package server

// Tests for chat block empty-state simplification in conversation.html.
// These pin that task 0155's Acceptance 3 is met: single short line telling what to do next.

import (
	"strings"
	"testing"
)

// TestChatBlockEmptyStateIsOneLine checks that the embedded template uses a
// single-line empty state — not two sentences.  After the change it should say just
// "Ask for anything." instead of "Ask for anything. What you ask for appears on the
// canvas."  This covers Acceptance 3.
func TestChatBlockEmptyStateIsOneLine(t *testing.T) {
	// conversationSrc is the //go:embedded template source.
	if strings.Contains(conversationSrc, "What you ask for appears on the canvas") {
		t.Error("chat block empty state must not have the second sentence 'What you ask for appears on the canvas' — it should be a single line: 'Ask for anything.'")
	}

	if !strings.Contains(conversationSrc, "Ask for anything") {
		t.Error("chat block empty state should say 'Ask for anything' as the single-line instruction")
	}
}
