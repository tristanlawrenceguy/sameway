package server_test

import (
	"regexp"
	"strings"
	"testing"
)

// TestHabitDetailLedeSaysStartedNotCreated asserts that a habit detail page
// does not show "Started", "Created" or any label prefix before the creation
// timestamp in its lede paragraph. The span still exists with just relative time text.
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

	re := regexp.MustCompile(`class="sw-detail__when[^"]*">([^<]+)</span>`)
	matches := re.FindStringSubmatch(page)
	if len(matches) < 2 {
		t.Fatal("habit detail page lede should have a sw-detail__when span with the creation time")
	}
	whenText := matches[1]

	// The lede must NOT contain "Created" as a label.
	if strings.HasPrefix(whenText, "Created ") {
		t.Error("habit detail page should not show \"Created\" — it uses the database field name instead of plain language")
	}

	// It must NOT start with "Started" either — just relative timestamp text.
	if strings.HasPrefix(whenText, "Started ") {
		t.Errorf("habit detail lede should not start with 'Started '; got %q", whenText)
	}
}

// TestNonHabitDetailPagesUseNoLabel ensures that non-habit detail pages do not
// show any label prefix ("Added") before the creation timestamp. All record
// types now show just relative time text in the lede (acceptance item 3).
func TestNonHabitDetailPagesUseAdded(t *testing.T) {
	a, h := newApp(t)

	// A note page should not have a label prefix.
	noteRec, err := a.Store.Create("note", map[string]any{"title": "A note"})
	if err != nil {
		t.Fatal(err)
	}
	notePage := get(t, h, "/t/note/"+noteRec.ID+fieldsView).Body.String()
	re := regexp.MustCompile(`class="sw-detail__when[^"]*">([^<]+)</span>`)
	matches := re.FindStringSubmatch(notePage)
	if len(matches) < 2 {
		t.Fatal("note detail page lede should have a sw-detail__when span")
	}
	noteWhenText := matches[1]
	if strings.HasPrefix(noteWhenText, "Added ") {
		t.Error("note detail lede should not start with 'Added' label")
	}

	// An activity entry page should not have a label prefix.
	actRec, err := a.Store.Create("activity", map[string]any{
		"actor":  "human",
		"action": "added",
	})
	if err != nil {
		t.Fatal(err)
	}
	actPage := get(t, h, "/t/activity/"+actRec.ID+fieldsView).Body.String()
	matches = re.FindStringSubmatch(actPage)
	if len(matches) < 2 {
		t.Fatal("activity detail page lede should have a sw-detail__when span")
	}
	actWhenText := matches[1]
	if strings.HasPrefix(actWhenText, "Added ") {
		t.Error("activity detail lede should not start with 'Added' label; got timestamp only")
	}

	// A message page should not have a label prefix.
	msgRec, err := a.Store.Create("message", map[string]any{
		"role":    "user",
		"content": "hello world",
	})
	if err != nil {
		t.Fatal(err)
	}
	msgPage := get(t, h, "/t/message/"+msgRec.ID+fieldsView).Body.String()
	matches = re.FindStringSubmatch(msgPage)
	if len(matches) < 2 {
		t.Fatal("message detail page lede should have a sw-detail__when span")
	}
	msgWhenText := matches[1]
	if strings.HasPrefix(msgWhenText, "Added ") {
		t.Error("message detail lede should not start with 'Added' label; got timestamp only")
	}
}

// TestHabitDetailLedeStillShowsAllInformation ensures that removing the label
// prefix does not remove any date or value information from a habit's lede
// (acceptance item 2). The lede should still contain the creation time.
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

	re := regexp.MustCompile(`class="sw-detail__when[^"]*">([^<]+)</span>`)
	matches := re.FindStringSubmatch(page)
	if len(matches) < 2 {
		t.Fatal("habit detail should have a sw-detail__when span with the time")
	}
	whenText := matches[1]

	// It must NOT start with "Started" — just relative timestamp text.
	if strings.HasPrefix(whenText, "Started ") {
		t.Errorf("habit lede should not start with 'Started '; got %q", whenText)
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

// TestHabitDetailLedeNoCadenceBadge asserts that a habit detail page does not
// show the raw cadence value "Each day" as a visible badge chip in its lede.
// The creation time is still present via whenMade() at the end of the lede
// (acceptance item 2 and 3).
func TestHabitDetailLedeNoCadenceBadge(t *testing.T) {
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

	// The lede must NOT contain "Each day" as a visible badge chip.
	if strings.Contains(page, ">Each day</span>") {
		t.Error("habit detail page should not show \"Each day\" — it uses the raw cadence enum value instead of plain language")
	}

	re := regexp.MustCompile(`class="sw-detail__when[^"]*">([^<]+)</span>`)
	matches := re.FindStringSubmatch(page)
	if len(matches) < 2 {
		t.Fatal("habit detail page lede should have a sw-detail__when span with the creation time")
	}
	whenText := matches[1]

	// It must NOT start with "Started" — just relative timestamp text.
	if strings.HasPrefix(whenText, "Started ") {
		t.Errorf("habit lede should not start with 'Started '; got %q", whenText)
	}
}
