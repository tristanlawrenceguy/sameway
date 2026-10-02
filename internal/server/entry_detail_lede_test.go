package server_test

import (
	"strings"
	"testing"
)

// TestEntryDetailLedeNoAtLabel asserts that an entry detail page does not show
// the raw database column name "At" as a visible label in its lede paragraph.
// The date value is still present via whenMade() ("Added ...") at the end of
// the lede (acceptance item 1 and 3).
func TestEntryDetailLedeNoAtLabel(t *testing.T) {
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

	// It should still say "Added" to indicate when the entry was created.
	if !strings.Contains(page, ">Added ") {
		t.Error("entry detail page lede should say \"Added\" with the creation time")
	}
}
