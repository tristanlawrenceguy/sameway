package server_test

import (
	"strings"
	"testing"
)

// TestEntryDetailLedeNoAtFieldLabel asserts that an entry detail page does not show
// "At" as a visible label chip; its at field's label, Logged, names it.
func TestEntryDetailLedeNoAtFieldLabel(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)

	habitRec, err := a.Store.Create("habit", map[string]any{
		"name":   "Exercise",
		"target": 1.0,
	})
	if err != nil {
		t.Fatal(err)
	}

	entryRec, err := a.Store.Create("entry", map[string]any{
		"habit":  habitRec.ID,
		"at":     "2026-10-02T04:32:00Z",
		"amount": 3.0,
	})
	if err != nil {
		t.Fatal(err)
	}

	page := get(t, h, "/t/entry/"+entryRec.ID+fieldsView).Body.String()

	// The lede must NOT contain "At" as a visible label chip.
	if strings.Contains(page, ">At ") {
		t.Error("entry detail page should not show \"At\" — it uses the raw database column name instead of plain language")
	}

	// The entry's time is shown only in the "Logged …" glance chip, not in a
	// separate sw-detail__when span. Acceptance item 1–3.
	if strings.Contains(page, `class="sw-detail__when`) {
		t.Error("entry detail page must not have a sw-detail__when span (creation time is in the Logged chip)")
	}

	// The lede should contain "Logged" from the glance chip with its datetime.
	if !strings.Contains(page, ">Logged") {
		t.Error("entry detail page lede should show \"Logged\" from the glance chip")
	}
}
