package server_test

import (
	"regexp"
	"strings"
	"testing"
)

// TestNoteDetailLedeNoAddedLabel asserts that the note detail page lede shows
// the creation time as natural language without any field-name label prefix
// such as "Added". Acceptance item 1.
func TestNoteDetailLedeNoAddedLabel(t *testing.T) {
	a, h := newApp(t)

	rec, err := a.Store.Create("note", map[string]any{"title": "Plans"})
	if err != nil {
		t.Fatal(err)
	}

	body := get(t, h, "/t/note/"+rec.ID).Body.String()

	re := regexp.MustCompile(`class="sw-detail__when[^"]*">([^<]+)</span>`)
	matches := re.FindStringSubmatch(body)
	if len(matches) < 2 {
		t.Fatal("note detail page should have a sw-detail__when span in the lede")
	}
	whenText := matches[1]

	// The when text must NOT start with "Added" (a field-name label).
	if strings.HasPrefix(whenText, "Added ") {
		t.Errorf("note detail lede should not have \"Added\" label; got %q", whenText)
	}

	// It must still contain relative time text showing when it was made.
	if !strings.Contains(whenText, "ago") && !strings.HasPrefix(whenText, "Today") {
		t.Errorf("note detail lede should show a relative timestamp; got %q", whenText)
	}
}

// TestTaskDetailLedeNoCreatedLabel asserts that the task detail page lede shows
// dates in natural language without any field-name label prefix such as
// "Created". Acceptance item 2.
func TestTaskDetailLedeNoCreatedLabel(t *testing.T) {
	a, h := newApp(t)

	rec, err := a.Store.Create("task", map[string]any{"title": "Buy milk"})
	if err != nil {
		t.Fatal(err)
	}

	body := get(t, h, "/t/task/"+rec.ID).Body.String()

	re := regexp.MustCompile(`class="sw-detail__when[^"]*">([^<]+)</span>`)
	matches := re.FindStringSubmatch(body)
	if len(matches) < 2 {
		t.Fatal("task detail page should have a sw-detail__when span in the lede")
	}
	whenText := matches[1]

	// The when text must NOT start with "Added" (the label that was used for
	// tasks before; it is not a field name but still a label prefix).
	if strings.HasPrefix(whenText, "Added ") {
		t.Errorf("task detail lede should not have \"Added\" label; got %q", whenText)
	}

	// It must still contain relative time text.
	if !strings.Contains(whenText, "ago") && !strings.HasPrefix(whenText, "Today") {
		t.Errorf("task detail lede should show a relative timestamp; got %q", whenText)
	}
}

// TestTaskDetailLedeNoUpdatedLabel asserts that the task detail page lede does
// not include an " · Updated ..." suffix — the last-change time always appeared
// with a field-name label ("Updated") and must be removed too. Acceptance item 2.
func TestTaskDetailLedeNoUpdatedLabel(t *testing.T) {
	a, h := newApp(t)

	rec, err := a.Store.Create("task", map[string]any{
		"title": "Buy milk",
	})
	if err != nil {
		t.Fatal(err)
	}

	body := get(t, h, "/t/task/"+rec.ID).Body.String()

	re := regexp.MustCompile(`class="sw-detail__when[^"]*">([^<]+)</span>`)
	matches := re.FindStringSubmatch(body)
	if len(matches) < 2 {
		t.Fatal("task detail page should have a sw-detail__when span in the lede")
	}
	whenText := matches[1]

	// The when text must NOT contain "Updated" anywhere — that label is removed.
	if strings.Contains(whenText, "Updated") || strings.Contains(whenText, " · Updated") {
		t.Errorf("task detail lede should not contain \"Updated\"; got %q", whenText)
	}

	// The span must still start with relative time text (no label).
	if !strings.HasPrefix(whenText, "ago") && !strings.HasPrefix(whenText, "Today") {
		t.Errorf("task detail lede should start with a relative timestamp; got %q", whenText)
	}
}

// TestHabitDetailLedeNoStartedLabel asserts that the habit detail page lede shows
// the creation time without any field-name label prefix such as "Started".
// Acceptance item 3.
func TestHabitDetailLedeNoStartedLabel(t *testing.T) {
	a, h := newApp(t)

	rec, err := a.Store.Create("habit", map[string]any{
		"name":   "Water",
		"target": 8.0,
		"unit":   "glasses",
	})
	if err != nil {
		t.Fatal(err)
	}

	body := get(t, h, "/t/habit/"+rec.ID).Body.String()

	re := regexp.MustCompile(`class="sw-detail__when[^"]*">([^<]+)</span>`)
	matches := re.FindStringSubmatch(body)
	if len(matches) < 2 {
		t.Fatal("habit detail page should have a sw-detail__when span in the lede")
	}
	whenText := matches[1]

	// The when text must NOT start with "Started" (a field-name label).
	if strings.HasPrefix(whenText, "Started ") {
		t.Errorf("habit detail lede should not have \"Started\" label; got %q", whenText)
	}

	// It must still contain relative time text.
	if !strings.Contains(whenText, "ago") && !strings.HasPrefix(whenText, "Today") {
		t.Errorf("habit detail lede should show a relative timestamp; got %q", whenText)
	}
}
