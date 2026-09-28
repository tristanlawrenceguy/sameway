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
		t.Error(`08-edit.js edit(): expected .sw-bar to be hidden (not skipped) when editing starts — removing the "sw-bar" skip condition fixes backlog 0338`)
	}

	// The loop must still skip sw-visually-hidden elements, so they are not double-
	// handled. Verify that only sw-visually-hidden remains in the skip condition.
	if !strings.Contains(script, `c.classList.contains("sw-visually-hidden")`) {
		t.Error(`08-edit.js edit(): expected sw-visually-hidden to still be skipped in the sibling-hiding loop`)
	}

	// The .sw-bar elements must end up in the covered array so cancel() restores them.
	// After removing the skip, any element that passes through (including .sw-bar) gets:
	//   c.style.display = "none"; covered.push(c);
	// This is already asserted indirectly by the covered.push pattern below, but we make
	// it explicit: the loop body must push every non-skipped sibling into covered.
	if !strings.Contains(script, `covered.push(c)`) {
		t.Error(`08-edit.js edit(): expected covered.push(c) in the sibling-hiding loop so all hidden elements are tracked for restoration by cancel()`)
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
		t.Error(`08-edit.js cancel(): expected covered[i].style.display = "" to restore hidden elements like .sw-bar when cancelling`)
	}

	if !strings.Contains(script, `covered[i].hidden = false`) &&
		!strings.Contains(script, "covered[i].hidden=false") {
		t.Error(`08-edit.js cancel(): expected covered[i].hidden = false to restore hidden elements like .sw-bar when cancelling`)
	}
}

// TestActivityDetailShowSwBarInReadOnly checks that an activity detail page in
// read-only mode (GET request) renders the floating action bar with "Delete
// activity" inside the data-block-id wrapper. This is acceptance item 1: the bar
// must exist so the user can edit or delete the activity. The bar is hidden by
// JS only when editing starts (see TestEditScriptHidesSwBarOnActivate).
func TestActivityDetailShowSwBarInReadOnly(t *testing.T) {
	a, h := newApp(t)

	rec, err := a.Store.Create("activity", map[string]any{
		"actor": "human", "action": "added", "detail": "A test activity",
	})
	if err != nil {
		t.Fatal(err)
	}

	r := get(t, h, "/t/activity/"+rec.ID)
	wantStatus(t, r, 200)

	body := r.Body.String()

	blockOpen := strings.Index(body, `data-block-id="`+rec.ID+`"`)
	if blockOpen < 0 {
		t.Fatalf("activity detail missing data-block-id wrapper\n%s", truncate(body))
	}

	swBarIdx := strings.Index(body[blockOpen:], `<div class="sw-bar sw-quiet">`)
	if swBarIdx < 0 {
		t.Errorf("activity detail block for %s should contain a div.sw-bar.sw-quiet (floating action bar)\n%s", rec.ID, truncate(body))
		return
	}

	// The bar must contain the Delete activity button.
	swBarInner := body[blockOpen+swBarIdx:]
	closeSwBar := strings.Index(swBarInner, `</div>`)
	if closeSwBar < 0 {
		t.Errorf("cannot find end of sw-bar div\n%s", truncate(body))
		return
	}

	barContent := swBarInner[:closeSwBar]
	if !strings.Contains(barContent, "Delete activity") {
		t.Errorf("sw-bar should contain the Delete activity button\nbar content: %s", truncate(barContent))
	}
}
