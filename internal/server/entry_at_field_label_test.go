package server_test

import (
	"strings"
	"testing"
)

// TestEntryDetailLedeAtFieldIsLabelled asserts that an entry's `at` datetime field
// shown as a glance chip in the lede has a visible label prefix explaining what it is.
// Currently DayName("at") returns "" so chips show just "3 days ago" with no context.
// Acceptance item 2.
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

	// The lede must contain a labelled chip for the `at` field — either "Entry" or
	// "Logged" before any relative timestamp text. An unlabeled chip like just
	// "Today at 4am" with no label prefix fails acceptance item 2.
	if !strings.Contains(page, ">Entry ") && !strings.Contains(page, ">Logged ") {
		t.Error("entry detail page lede should show a labelled chip for the `at` field (e.g., \"Entry\" or \"Logged\" before the timestamp); got an unlabeled date chip")
	}

	// The entry's at value must still appear somewhere on the page.
	if !strings.Contains(page, "4:32am") && !strings.Contains(page, "04:32") {
		t.Error("entry detail page should show the `at` datetime value from the record")
	}
}
