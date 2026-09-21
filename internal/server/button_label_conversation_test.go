package server_test

// Tests for button-label simplification in conversation.html (pop out → open).
// These pin that task 0155's Acceptance 2 is met: no phrasal verbs.

import (
	"net/http"
	"strings"
	"testing"
)

// TestConversationNoPhrasalVerb checks the rendered page does not contain "Pop out".
// After the change this button should say "Open" with context for screen readers.
func TestConversationNoPhrasalVerb(t *testing.T) {
	_, h := newApp(t)

	// The pop-out link appears on pages that embed a chat block but are NOT /chat itself.
	// So we hit the home page (which has a canvas with a chat block).
	rec := get(t, h, "/")
	wantStatus(t, rec, http.StatusOK)
	body := rec.Body.String()

	if strings.Contains(body, "Pop out") {
		t.Error("the pop-out link label must not say 'Pop out'; it should be simplified to 'Open' with context for screen readers (Acceptance 2)")
	}
}
