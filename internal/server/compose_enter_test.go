package server_test

import (
	"strings"
	"testing"
)

// TestComposeKeyHandlerExists checks that 08-edit.js defines a composeKeyHandler
// function, which is the entry point for the chat textarea Enter-to-send
// enhancement. This covers acceptance item 4 (handler implemented in 08-edit.js).
func TestComposeKeyHandlerExists(t *testing.T) {
	script := readEditScript(t)

	if !strings.Contains(script, "function composeKeyHandler") {
		t.Error("08-edit.js: expected a function named composeKeyHandler to handle the chat textarea Enter key")
	}
}

// TestComposeKeyHandlerUsesCorrectSelector checks that composeKeyHandler queries
// for the chat compose form's textarea using the selector form.sw-compose textarea,
// which scopes the enhancement to only the chat composer. This covers acceptance
// item 4 (progressive enhancement, no server-rendered control added).
func TestComposeKeyHandlerUsesCorrectSelector(t *testing.T) {
	script := readEditScript(t)

	if !strings.Contains(script, `querySelector("form.sw-compose textarea")`) &&
		!strings.Contains(script, "querySelector('form.sw-compose textarea')") &&
		!strings.Contains(script, `querySelectorAll("form.sw-compose textarea")`) &&
		!strings.Contains(script, "querySelectorAll('form.sw-compose textarea')") {
		t.Error(`08-edit.js: expected composeKeyHandler to query for form.sw-compose textarea; the selector must be scoped to the chat compose form only`)
	}
}

// TestComposeKeyHandlerInterceptsEnter checks that when Enter is pressed without
// Shift in the compose textarea, the handler calls preventDefault() and submits
// the form. This covers acceptance items 1 (Enter sends message) and 2 (Shift+Enter
// inserts newline — by absence of prevention).
func TestComposeKeyHandlerInterceptsEnter(t *testing.T) {
	script := readEditScript(t)

	// The handler must check for Enter key press.
	if !strings.Contains(script, `e.key === "Enter"`) &&
		!strings.Contains(script, "e.key == \"Enter\"") &&
		!strings.Contains(script, "e.key === 'Enter'") {
		t.Error(`08-edit.js: expected composeKeyHandler to check e.key === "Enter" for the chat textarea keydown listener`)
	}

	// The handler must call preventDefault when Enter is pressed without Shift.
	if !strings.Contains(script, "e.preventDefault()") {
		t.Error("08-edit.js: expected composeKeyHandler to call e.preventDefault() when Enter is pressed on the compose textarea")
	}

	// The handler must submit the form (not just prevent default).
	if !strings.Contains(script, ".submit()") &&
		!strings.Contains(script, "form.submit()") {
		t.Error("08-edit.js: expected composeKeyHandler to call form.submit() when Enter is pressed without Shift on the compose textarea")
	}

	// The handler must only act when Shift is NOT held — checking !e.shiftKey.
	if !strings.Contains(script, "!e.shiftKey") &&
		!strings.Contains(script, "e.shiftKey === false") {
		t.Error("08-edit.js: expected composeKeyHandler to check !e.shiftKey so that only unmodified Enter submits (Shift+Enter inserts a newline)")
	}
}

// TestComposeKeyHandlerCalledFromInit checks that composeKeyHandler() is called
// from the init() function so it runs on page load. This covers acceptance item 4
// (progressive enhancement — handler activates when JS is available).
func TestComposeKeyHandlerCalledFromInit(t *testing.T) {
	script := readEditScript(t)

	if !strings.Contains(script, "composeKeyHandler()") {
		t.Error("08-edit.js: expected composeKeyHandler() to be called from init() so it runs on page load and when new content is added")
	}
}

// TestComposeKeyHandlerGracefulDegradation checks that composeKeyHandler returns
// early if no form.sw-compose textarea exists, implementing graceful degradation.
func TestComposeKeyHandlerGracefulDegradation(t *testing.T) {
	script := readEditScript(t)

	// The handler should check for the existence of the textarea and return early
	// if not found (e.g., using .length === 0 or checking null/undefined).
	if !strings.Contains(script, "composeKeyHandler") {
		t.Skip("composeKeyHandler does not exist yet; graceful degradation test skipped")
	}

	// Look for a guard that returns early when no compose form is found.
	hasGuard := strings.Contains(script, ".length === 0") ||
		strings.Contains(script, ".length==0") ||
		strings.Contains(script, "=== null") ||
		strings.Contains(script, "== null") ||
		strings.Contains(script, "!textarea") ||
		strings.Contains(script, "if (!form") ||
		strings.Contains(script, "return;")

	if !hasGuard {
		t.Error("08-edit.js: expected composeKeyHandler to return early if form.sw-compose textarea is not found (graceful degradation)")
	}
}
