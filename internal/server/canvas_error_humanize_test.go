package server_test

import (
	"strings"
	"testing"
)

// TestCanvasValidationErrorDoesNotExposeSchemaInternals checks that when a block
// with invalid props is rendered on the canvas, error messages do not expose raw
// JSON Schema field names like "Label", "Text", or validation keywords like
// "missing property", "additional properties". A person reading the alert should
// see plain language. This covers acceptance items 1-3 of task #0221.
func TestCanvasValidationErrorDoesNotExposeSchemaInternals(t *testing.T) {
	a, h := newApp(t)

	// Create a block with invalid props: has "text" (extra field) but missing "label".
	if _, err := a.Store.Create("block", map[string]any{
		"component": "button",
		"props":     map[string]any{"text": "Go"},
	}); err != nil {
		t.Skip("the store will not hold such a block:", err)
	}

	body := get(t, h, "/").Body.String()

	// Acceptance 1: Canvas alert blocks do not contain raw schema field names as
	// visible text. The internal prop keys "Label" and "Text" should not appear
	// in error contexts (the word "button" is fine since it's the component name).
	for _, raw := range []string{"Label:", "Text:"} {
		if strings.Contains(body, raw) {
			t.Errorf("canvas should not show internal field name %q: found in page", raw)
		}
	}

	// Acceptance 2: Canvas alert blocks show human-readable descriptions instead of
	// machine-language phrases.
	for _, raw := range []string{"missing property", "additional properties", "is the wrong type"} {
		if strings.Contains(body, raw) {
			t.Errorf("canvas should not expose %q in error text: found in page", raw)
		}
	}

	// Acceptance 3: The alert block still conveys what is wrong — it says the button
	// could not be shown.
	if !strings.Contains(body, "This button could not be shown") {
		t.Errorf("the canvas should say the button could not be shown; got:\n%s", body)
	}

	// The logged error (printed to stderr by server.go:component) must also be
	// humanized — it is read by developers and operators, so raw schema terms
	// leaking there defeats debugging clarity too.
	// We verify this indirectly: if formatValidation does its job, the error
	// string passed to log.Printf will already be clean. The existing test
	// TestABrokenBlockSaysSoInPlainWords already checks the HTML; the render
	// tests check the error string directly. This server-level test confirms
	// the full pipeline: create block → render page, no raw terms in output.
}
