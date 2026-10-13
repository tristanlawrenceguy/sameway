package server_test

// Tests for raw field labels on the entry detail page (backlog 0705). The
// entry schema currently has no label on its amount field, so the facts
// definition list shows "<dt>Amount</dt>" instead of a plain-language term.
// Adding label: Quantity to the schema fixes this; the test asserts both that
// "Amount" is gone and "Quantity" appears (acceptance 1–2).

import (
	"strings"
	"testing"
)

// TestEntryDetailDoesNotShowRawAmountLabel asserts that the entry detail page
// does not display "<dt>Amount</dt>" as a field label in its facts definition
// list. The term should be readable, such as "Quantity". (Acceptance 1.)
func TestEntryDetailDoesNotShowRawAmountLabel(t *testing.T) {
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

	page := get(t, h, "/t/entry/"+entryRec.ID).Body.String()

	if strings.Contains(page, "<dt>Amount</dt>") {
		t.Error("entry detail must not show \"<dt>Amount</dt>\" as a field label; use plain language like \"Quantity\"")
	}
}

// TestEntryDetailShowsQuantityLabel asserts that the entry detail page renders
// "Quantity" as the human-readable label for the amount field in its facts
// definition list. (Acceptance 1.)
func TestEntryDetailShowsQuantityLabel(t *testing.T) {
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
		"amount": 5.0,
	})
	if err != nil {
		t.Fatal(err)
	}

	page := get(t, h, "/t/entry/"+entryRec.ID).Body.String()

	if !strings.Contains(page, "<dt>Quantity</dt>") {
		t.Errorf("entry detail should show \"<dt>Quantity</dt>\" for the amount field; page body:\n%s", truncate(page))
	}
}

// TestEntryDetailShowsValueUnchanged asserts that the numeric value stored in
// the amount field is still displayed on the entry detail page, unchanged. Only
// the label changes from "Amount" to "Quantity". (Acceptance 2.)
func TestEntryDetailShowsValueUnchanged(t *testing.T) {
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
		"amount": 7.5,
	})
	if err != nil {
		t.Fatal(err)
	}

	page := get(t, h, "/t/entry/"+entryRec.ID).Body.String()

	if !strings.Contains(page, "7.5") {
		t.Errorf("entry detail should still show the amount value 7.5; page body:\n%s", truncate(page))
	}
}
