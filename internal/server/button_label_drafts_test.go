package server_test

// Tests for button-label simplification in drafts.js (Continue editing → Edit).
// These pin that task 0155's Acceptance 2 is met: no gerund-based labels.

import (
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/design"
)

// TestDraftsButtonLabelIsShort checks that the drafts.js enhancement script uses
// a short, plain-verb button label — not "Continue editing".  After the change it
// should say just "Edit".  This covers Acceptance 2 (active verb only).
func TestDraftsButtonLabelIsShort(t *testing.T) {
	data, err := design.FS.ReadFile("base/16-drafts.js")
	if err != nil {
		t.Fatalf("cannot read drafts.js: %v", err)
	}
	body := string(data)

	if strings.Contains(body, "Continue editing") {
		t.Error("drafts.js must not use 'Continue editing'; it should say just 'Edit' (one word, plain verb)")
	}
}
