package server_test

import (
	"strings"
	"testing"
)

// TestEntryDetailLedeAtFieldIsLabelled asserts that an entry's `at` datetime field
// shown as a glance chip in the lede has a visible label prefix explaining what it is.
// The provided entry type labels its at field Logged, so the chip says it.
func TestEntryDetailLedeAtFieldIsLabelled(t *testing.T) {
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
		"at":     "2026-10-09T04:32:00Z",
		"amount": 3.0,
	})
	if err != nil {
		t.Fatal(err)
	}

	page := get(t, h, "/t/entry/"+entryRec.ID).Body.String()

	// The lede must name the `at` field's chip by its label, Logged.
	if !strings.Contains(page, ">Logged ") {
		t.Error("entry detail page lede should name its day Logged; got an unlabelled date chip")
	}

	// The entry's at value must still appear somewhere on the page.
	if !strings.Contains(page, "4:32am") && !strings.Contains(page, "04:32") {
		t.Error("entry detail page should show the `at` datetime value from the record")
	}
}
