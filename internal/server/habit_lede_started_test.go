package server_test

import (
	"strings"
	"testing"
)

// TestHabitDetailLedeSaysStartedNotCreated asserts that a habit detail page
// uses "Started" instead of the database field name "Created" in its lede
// paragraph under the title (backlog 0606). After the fix, the word Created
// should not appear as a human-readable label on any habit page.
func TestHabitDetailLedeSaysStartedNotCreated(t *testing.T) {
	a, h := newApp(t)

	rec, err := a.Store.Create("habit", map[string]any{
		"name":   "Water",
		"target": 8.0,
		"unit":   "glasses",
	})
	if err != nil {
		t.Fatal(err)
	}

	page := get(t, h, "/t/habit/"+rec.ID+fieldsView).Body.String()

	// The lede must NOT contain "Created" as a label.
	if strings.Contains(page, ">Created ") {
		t.Error("habit detail page should not show \"Created\" — it uses the database field name instead of plain language")
	}

	// It should say "Started" to indicate when the habit was begun.
	if !strings.Contains(page, ">Started ") {
		t.Error("habit detail page lede should say \"Started\" before the creation time")
	}
}

// TestNonHabitDetailPagesStillSayCreated ensures that non-habit detail pages
// continue to use "Created" in their lede. This task only changes habit ledes;
// note, activity and message pages must be unaffected (acceptance item 3).
func TestNonHabitDetailPagesStillSayCreated(t *testing.T) {
	a, h := newApp(t)

	// A note page should still say "Created".
	noteRec, err := a.Store.Create("note", map[string]any{"title": "A note"})
	if err != nil {
		t.Fatal(err)
	}
	notePage := get(t, h, "/t/note/"+noteRec.ID+fieldsView).Body.String()
	if !strings.Contains(notePage, ">Created ") {
		t.Error("note detail page should still say \"Created\" in the lede")
	}

	// An activity entry page should still say "Created".
	actRec, err := a.Store.Create("activity", map[string]any{
		"actor":  "human",
		"action": "added",
	})
	if err != nil {
		t.Fatal(err)
	}
	actPage := get(t, h, "/t/activity/"+actRec.ID+fieldsView).Body.String()
	if !strings.Contains(actPage, ">Created ") {
		t.Error("activity detail page should still say \"Created\" in the lede")
	}

	// A message page should still say "Created".
	msgRec, err := a.Store.Create("message", map[string]any{
		"role":    "user",
		"content": "hello world",
	})
	if err != nil {
		t.Fatal(err)
	}
	msgPage := get(t, h, "/t/message/"+msgRec.ID+fieldsView).Body.String()
	if !strings.Contains(msgPage, ">Created ") {
		t.Error("message detail page should still say \"Created\" in the lede")
	}
}

// TestHabitDetailLedeStillShowsAllInformation ensures that replacing "Created"
// with "Started" does not remove any date or value information from a habit's
// lede (acceptance item 2). The lede should still contain the creation time.
func TestHabitDetailLedeStillShowsAllInformation(t *testing.T) {
	a, h := newApp(t)

	rec, err := a.Store.Create("habit", map[string]any{
		"name":   "Running",
		"target": 5.0,
		"unit":   "km",
	})
	if err != nil {
		t.Fatal(err)
	}

	page := get(t, h, "/t/habit/"+rec.ID+fieldsView).Body.String()

	// The lede must still show a timestamp after "Started".
	if !strings.Contains(page, ">Started ") {
		t.Error("habit detail should say \"Started\" before the time")
	}

	// The habit name and settings must still be present.
	if !strings.Contains(page, "<dt>How much</dt>") {
		t.Error("habit detail should show the target setting")
	}
	if !strings.Contains(page, "<dt>Counted in</dt>") {
		t.Error("habit detail should show the unit setting")
	}

	// The lede paragraph must still be rendered (the <p class="sw-lede">).
	if !strings.Contains(page, `class="sw-lede"`) {
		t.Error("habit detail page must still have a lede paragraph")
	}
}
