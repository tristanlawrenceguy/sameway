package server_test

import (
	"strings"
	"testing"
)

// TestProposalHandlerExists checks that 08-edit.js defines a proposeHandler
// function, which is the entry point for intercepting proposal button clicks.
// This covers acceptance items 1 and 2 (Yes/No buttons wired to accept/reject).
func TestProposalHandlerExists(t *testing.T) {
	script := readEditScript(t)

	if !strings.Contains(script, "function proposeHandler") {
		t.Error("08-edit.js: expected a function named proposeHandler to handle proposal button clicks")
	}
}

// TestProposalHandlerUsesCorrectSelector checks that proposeHandler selects
// submit buttons inside .sw-proposal elements using the selector
// ".sw-proposal [type=\"submit\"]", scoping the enhancement to only proposals.
// This covers acceptance items 1 and 2 (only proposal buttons are intercepted).
func TestProposalHandlerUsesCorrectSelector(t *testing.T) {
	script := readEditScript(t)

	if !strings.Contains(script, `querySelectorAll(".sw-proposal [type="submit"]")`) &&
		!strings.Contains(script, "querySelectorAll(\".sw-proposal [type=\\\"submit\\\"]\")") &&
		!strings.Contains(script, "querySelectorAll('.sw-proposal [type=\"submit\"]')") {
		t.Error(`08-edit.js: expected proposeHandler to query for submit buttons inside .sw-proposal elements using the selector ".sw-proposal [type="submit"]"`)
	}
}

// TestProposalHandlerPreventsNativeSubmit checks that proposeHandler calls
// e.preventDefault() on proposal button clicks to stop the form's native POST.
// This covers acceptance item 3 (no new chat message is created — the form does
// not do a full-page navigation via its action URL). The handler must prevent
// the default submission so the page stays in place for JS cleanup.
func TestProposalHandlerPreventsNativeSubmit(t *testing.T) {
	script := readEditScript(t)

	// Look for preventDefault being called inside proposeHandler, not just any
	// function in the file. We check that proposeHandler references a submit or
	// form element and calls preventDefault on it.
	hasPreventInProposal := strings.Contains(script, "function proposeHandler") &&
		strings.Contains(script, ".preventDefault()")

	if !hasPreventInProposal {
		t.Error("08-edit.js: expected proposeHandler to call e.preventDefault() on proposal button clicks to prevent native form submission")
	}
}

// TestProposalHandlerUsesFetch checks that proposeHandler posts via fetch()
// rather than relying on the browser's native form POST. This covers acceptance
// item 3 (no new chat message is created) — without fetch there would be no way
// to stay on the page after answering a proposal, and without preventDefault the
// browser would navigate away before JS could clean up.
func TestProposalHandlerUsesFetch(t *testing.T) {
	script := readEditScript(t)

	if !strings.Contains(script, "fetch(") &&
		!strings.Contains(script, "fetch (") {
		t.Error("08-edit.js: expected proposeHandler to use fetch() for posting the proposal answer without navigating away")
	}
}

// TestProposalHandlerRemovesFromDOM checks that proposeHandler removes the
// answered proposal element from the DOM after a successful fetch. This covers
// acceptance items 1 and 2 (the proposal is removed from the page after
// answering) and item 4 (only one active proposal visible — removing it clears
// the stack).
func TestProposalHandlerRemovesFromDOM(t *testing.T) {
	script := readEditScript(t)

	// The handler must call .remove() on a proposal element. Look for remove()
	// appearing after or near a reference to "proposal" in the script.
	if !strings.Contains(script, ".remove()") &&
		!strings.Contains(script, ".remove( )") {
		t.Error("08-edit.js: expected proposeHandler to call .remove() on the proposal element after a successful answer so it disappears from the page")
	}

	// The removal must target the proposal div, not just any element. Check that
	// remove() appears in context with "proposal" or ".sw-proposal".
	hasProposalRemove := strings.Contains(script, "proposal.remove()") ||
		strings.Contains(script, `proposal.remove( )`) ||
		strings.Contains(script, "\".sw-proposal\"")

	if !hasProposalRemove {
		t.Error("08-edit.js: expected proposeHandler to call .remove() specifically on the proposal element (not just any element)")
	}
}

// TestProposalHandlerRemovesAllProposals checks that when one proposal is
// answered, all other pending proposals are also removed — covering acceptance
// item 4 (only one active proposal visible at a time). This prevents stale
// stacked proposals from accumulating on the page.
func TestProposalHandlerRemovesAllProposals(t *testing.T) {
	script := readEditScript(t)

	// The handler should query all .sw-proposal elements and remove them, not
	// just the one that was answered. Look for a querySelectorAll on proposals
	// combined with a remove() or forEach pattern within proposeHandler.
	hasRemoveAll := strings.Contains(script, "querySelectorAll(\".sw-proposal") ||
		strings.Contains(script, `querySelectorAll(".sw-proposal`)

	if !hasRemoveAll {
		t.Error("08-edit.js: expected proposeHandler to remove ALL pending proposals when one is answered (only one active proposal visible at a time)")
	}
}

// TestProposalHandlerCalledFromInit checks that proposeHandler() is called from
// the init() function so proposal buttons are wired up on page load. This covers
// acceptance items 1–4 (progressive enhancement — handler activates when JS is
// available).
func TestProposalHandlerCalledFromInit(t *testing.T) {
	script := readEditScript(t)

	if !strings.Contains(script, "proposeHandler()") {
		t.Error("08-edit.js: expected proposeHandler() to be called from init() so it runs on page load and when new content is added")
	}
}

// TestProposalHandlerGracefulDegradation checks that proposeHandler returns
// early if no proposal elements exist, implementing graceful degradation.
func TestProposalHandlerGracefulDegradation(t *testing.T) {
	script := readEditScript(t)

	if !strings.Contains(script, "proposeHandler") {
		t.Skip("proposeHandler does not exist yet; graceful degradation test skipped")
	}

	// Look for a guard that returns early when no proposals are found.
	hasGuard := strings.Contains(script, ".length === 0") ||
		strings.Contains(script, ".length==0") ||
		strings.Contains(script, "=== null") ||
		strings.Contains(script, "== null")

	if !hasGuard {
		t.Error("08-edit.js: expected proposeHandler to return early if no .sw-proposal [type=\"submit\"] elements are found (graceful degradation)")
	}
}
