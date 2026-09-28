package server_test

// Tests for trimming all visible button labels to ≤3 words maximum.
// These pin that every button label (button component, link with look="button",
// and hardcoded buttons in Go templates) is three space-separated tokens or fewer.

import (
	"net/http"
	"strings"
	"testing"
)

// TestFocusPageBackLinkIsShort checks the focus page's "back to canvas" link
// uses a label of ≤3 words — not "Back to the canvas".  After the change it
// should say just "Back" with context for screen readers.  This covers Acceptance 1
// (≤3 words) and Acceptance 2 (no filler phrases).
func TestFocusPageBackLinkIsShort(t *testing.T) {
	_, h := newApp(t)

	// Create a canvas block so the focus page exists.
	var blk struct{ ID string }
	decode(t, postJSON(t, h, "POST", "/api/block", map[string]any{
		"component": "clock",
		"region":    "main",
		"props":     map[string]any{"label": "Kitchen clock"},
	}), &blk)

	rec := get(t, h, "/canvas/"+blk.ID)
	wantStatus(t, rec, http.StatusOK)
	body := rec.Body.String()

	if strings.Contains(body, "to the canvas") {
		t.Error("the back-link label must be ≤3 words; 'Back to the canvas' is 4 words — use 'Back' + context for screen readers (Acceptance 1)")
	}
}

// TestImportLinkLabelIsShort checks the import button on a listing page uses
// a label of ≤3 words.  After the change it should say just "Import" with
// context carrying the type name.  This covers Acceptance 1 and 2.
func TestImportLinkLabelIsShort(t *testing.T) {
	_, h := newApp(t)

	// Create a note so the listing page shows records (not empty state).
	wantStatus(t, postJSON(t, h, "POST", "/api/note", map[string]any{
		"title": "Test note",
		"body":  "content here",
	}), http.StatusCreated)

	rec := get(t, h, "/t/note")
	wantStatus(t, rec, http.StatusOK)
	body := rec.Body.String()

	if strings.Contains(body, "from a file") {
		t.Error("the import link label must be ≤3 words; 'Import <type> from a file' is 5 words — use just 'Import' with context for screen readers (Acceptance 1)")
	}
}
