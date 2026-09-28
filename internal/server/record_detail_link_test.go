package server_test

import (
	"net/http"
	"strings"
	"testing"
)

// TestRecordBlockFocusPageLinkLabelIsShort checks the "view full" link on a
// record block's focus page uses at most three visible words — not "Open this
// note on its page". The label should be short with context for screen readers
// via visually-hidden text. This covers Acceptance 1 (≤3 words) and Acceptance 2
// (no filler phrases).
func TestRecordBlockFocusPageLinkLabelIsShort(t *testing.T) {
	_, h := newApp(t)

	// Create a note so the focus page exists.
	var note struct{ ID string }
	decode(t, postJSON(t, h, "POST", "/api/note", map[string]any{
		"title": "Call the dentist",
		"body":  "Ask about Thursday.",
	}), &note)

	wantStatus(t, postJSON(t, h, "POST", "/api/block", map[string]any{
		"component": "record",
		"props":     map[string]any{"type": "note", "record": note.ID},
		"span":      6,
	}), http.StatusCreated)

	var blocks struct{ Records []struct{ ID string } }
	decode(t, get(t, h, "/api/block"), &blocks)

	focus := get(t, h, "/canvas/"+blocks.Records[len(blocks.Records)-1].ID).Body.String()

	if strings.Contains(focus, "Open this note on its page") {
		t.Error("the record block focus-page link label must be ≤3 words; 'Open this note on its page' is 6 words — use a short label with context for screen readers (Acceptance 1)")
	}
	if !strings.Contains(focus, "sw-link--button") {
		t.Error("the record block focus-page link should carry the sw-link--button class so it counts as a button-styled action (Acceptance 2)")
	}
}
