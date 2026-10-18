package server_test

import (
	"regexp"
	"strings"
	"testing"
)

// TestHabitDetailLedeCreationTimeIsLabelled asserts that a habit detail page
// shows "Created" before the creation timestamp in its lede paragraph.
func TestHabitDetailLedeCreationTimeIsLabelled(t *testing.T) {
	t.Parallel()
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

	re := regexp.MustCompile(`class="sw-detail__when[^"]*">Created at <time[^>]*>([^<]+)</time>`)
	matches := re.FindStringSubmatch(page)
	if len(matches) < 2 {
		t.Fatal("habit detail page lede should have a sw-detail__when span with \"Created at\" label")
	}
	whenText := matches[1]

	// It must contain relative time text.
	if !strings.Contains(whenText, "ago") && !strings.HasPrefix(whenText, "today") {
		t.Errorf("habit detail lede should show a relative timestamp; got %q", whenText)
	}
}

// TestAllRecordTypesShowCreatedLabel asserts that non-habit detail pages show
// "Created" before the creation timestamp. All record types now have a visible
// label (acceptance item 4).
func TestAllRecordTypesShowCreatedLabel(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)

	// A note page should show "Created".
	noteRec, err := a.Store.Create("note", map[string]any{"title": "A note"})
	if err != nil {
		t.Fatal(err)
	}
	notePage := get(t, h, "/t/note/"+noteRec.ID+fieldsView).Body.String()
	re := regexp.MustCompile(`class="sw-detail__when[^"]*">Created at <time[^>]*>([^<]+)</time>`)
	matches := re.FindStringSubmatch(notePage)
	if len(matches) < 2 {
		t.Fatal("note detail page lede should have a sw-detail__when span with \"Created at\" label")
	}

	// An activity entry page should show "Created".
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
		t.Fatal("activity detail page lede should have a sw-detail__when span with \"Created\" label")
	}

	// A message page should show "Created".
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
		t.Fatal("message detail page lede should have a sw-detail__when span with \"Created\" label")
	}
}

// TestHabitDetailLedeStillShowsAllInformation ensures that adding the label
// prefix does not remove any date or value information from a habit's lede.
func TestHabitDetailLedeStillShowsAllInformation(t *testing.T) {
	t.Parallel()
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

	re := regexp.MustCompile(`class="sw-detail__when[^"]*">Created at <time[^>]*>([^<]+)</time>`)
	matches := re.FindStringSubmatch(page)
	if len(matches) < 2 {
		t.Fatal("habit detail should have a sw-detail__when span with \"Created at\" label")
	}

	// The habit name and settings must still be present.
	if !strings.Contains(page, "<dt>How much</dt>") {
		t.Error("habit detail should show the target setting")
	}
	if !strings.Contains(page, "<dt>Counted in</dt>") {
		t.Error("habit detail should show the unit setting")
	}

	// The lede must still be rendered.
	if !strings.Contains(page, `class="sw-lede"`) {
		t.Error("habit detail page must still have a lede paragraph")
	}
}

// TestHabitDetailLedeNoCadenceBadge asserts that a habit detail page does not
// show the raw cadence value "Each day" as a visible badge chip in its lede.
func TestHabitDetailLedeNoCadenceBadge(t *testing.T) {
	t.Parallel()
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

	re := regexp.MustCompile(`class="sw-detail__when[^"]*">Created at <time[^>]*>([^<]+)</time>`)
	matches := re.FindStringSubmatch(page)
	if len(matches) < 2 {
		t.Fatal("habit detail page lede should have a sw-detail__when span with \"Created at\" label")
	}
}
