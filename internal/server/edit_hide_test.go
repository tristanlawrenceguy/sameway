package server_test

import (
	"strings"
	"testing"
)

// TestEditScriptHidesDefinitionListOnActivate checks that the inline edit
// script explicitly hides any <dl class="sw-dl"> element when editing starts,
// so the read-only definition list does not appear alongside the form inputs.
// This covers acceptance items 1 and 2 of task 0103: clicking Edit block must
// hide the original display (the dl) while showing only the form.
func TestEditScriptHidesDefinitionListOnActivate(t *testing.T) {
	script := readEditScript(t)

	// The edit() function must explicitly find and hide any <dl class="sw-dl">
	// inside the block wrapper, so the static definition list is removed from
	// view when editing begins. Acceptance item 1: clicking Edit hides the dl.
	if !strings.Contains(script, `querySelector("dl.sw-dl")`) &&
		!strings.Contains(script, "querySelector('dl.sw-dl')") {
		t.Error(`08-edit.js edit(): expected querySelector("dl.sw-dl") to find and hide the read-only definition list when editing starts (acceptance 1)`)
	}

	// The found <dl> must have its hidden property set to true. Acceptance item 2:
	// .sw-dl-block remains in DOM but dl children are hidden while form is active.
	if !strings.Contains(script, `dl.hidden = true`) &&
		!strings.Contains(script, "dl.hidden=true") {
		t.Error(`08-edit.js edit(): expected dl.hidden = true to hide the definition list (acceptance 2)`)
	}

	// The hidden <dl> must be added to the covered array so cancel() can restore it.
	if !strings.Contains(script, `covered.push(dl)`) &&
		!strings.Contains(script, "covered.push(dl)") {
		t.Error(`08-edit.js edit(): expected covered.push(dl) so the definition list is tracked for restoration by cancel() (acceptance 3)`)
	}
}

// TestEditScriptRestoresDefinitionListOnCancel checks that pressing Cancel or
// pressing Escape explicitly restores visibility of the <dl class="sw-dl">, so
// the read-only display reappears after editing is abandoned. This covers
// acceptance item 3: Cancel restores the original dl.
func TestEditScriptRestoresDefinitionListOnCancel(t *testing.T) {
	script := readEditScript(t)

	// cancel() must set hidden = false on any <dl class="sw-dl"> it finds,
	// ensuring the definition list becomes visible again after Cancel.
	if !strings.Contains(script, `dl.hidden = false`) &&
		!strings.Contains(script, "dl.hidden=false") {
		t.Error(`08-edit.js cancel(): expected dl.hidden = false to restore the definition list when cancelling (acceptance 3)`)
	}
}
