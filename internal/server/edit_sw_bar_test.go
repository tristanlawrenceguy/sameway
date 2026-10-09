package server_test

import (
	"strings"
	"testing"
)

// TestEditScriptHidesSwBarOnActivate checks that the inline edit script hides
// .sw-bar elements (the floating action bar) when editing starts, so they do not
// appear alongside the form inputs. This is the fix for backlog 0338: clicking
// Edit block on an activity detail page was showing duplicate controls — the
// inline form and the floating action bar simultaneously. Acceptance items 2–4.
func TestEditScriptHidesSwBarOnActivate(t *testing.T) {
	script := readEditScript(t)

	// The sibling-hiding loop in edit() must NOT skip .sw-bar elements; it should
	// hide them (style.display = "none") and add them to the covered array. The old
	// buggy code had: if (c.classList.contains("sw-bar") || ...) continue; — this
	// test asserts that the sw-bar skip is gone, so .sw-bar elements are now hidden.
	if strings.Contains(script, `c.classList.contains("sw-bar")`) {
		t.Error(`09-edit.js edit(): expected .sw-bar to be hidden (not skipped) when editing starts — removing the "sw-bar" skip condition fixes backlog 0338`)
	}

	// The loop must still skip sw-visually-hidden elements, so they are not double-
	// handled. Verify that only sw-visually-hidden remains in the skip condition.
	if !strings.Contains(script, `c.classList.contains("sw-visually-hidden")`) {
		t.Error(`09-edit.js edit(): expected sw-visually-hidden to still be skipped in the sibling-hiding loop`)
	}

	// The .sw-bar elements must end up in the covered array so cancel() restores them.
	// After removing the skip, any element that passes through (including .sw-bar) gets:
	//   c.style.display = "none"; covered.push(c);
	// This is already asserted indirectly by the covered.push pattern below, but we make
	// it explicit: the loop body must push every non-skipped sibling into covered.
	if !strings.Contains(script, `covered.push(c)`) {
		t.Error(`09-edit.js edit(): expected covered.push(c) in the sibling-hiding loop so all hidden elements are tracked for restoration by cancel()`)
	}
}

// TestEditScriptRestoresSwBarOnCancel checks that pressing Cancel or Escape
// restores visibility of .sw-bar elements (the floating action bar), so the
// read-only controls reappear after editing is abandoned. Acceptance item 4.
func TestEditScriptRestoresSwBarOnCancel(t *testing.T) {
	script := readEditScript(t)

	// cancel() iterates over covered and restores each element's display and hidden
	// state. Since .sw-bar elements will now be in the covered array (per the fix to
	// edit()), they are automatically restored by this loop — no separate code needed.
	// Assert that both properties are cleared so the bar becomes visible again.
	if !strings.Contains(script, `covered[i].style.display = ""`) &&
		!strings.Contains(script, "covered[i].style.display=\"\"") {
		t.Error(`09-edit.js cancel(): expected covered[i].style.display = "" to restore hidden elements like .sw-bar when cancelling`)
	}

	if !strings.Contains(script, `covered[i].hidden = false`) &&
		!strings.Contains(script, "covered[i].hidden=false") {
		t.Error(`09-edit.js cancel(): expected covered[i].hidden = false to restore hidden elements like .sw-bar when cancelling`)
	}
}
