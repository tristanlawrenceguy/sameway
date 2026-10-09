package server_test

import (
	"regexp"
	"strings"
	"testing"
)

// TestFileDetailLedeNoAddedLabel asserts that a file detail page does not show
// "Added", "Created" or any other label prefix before the creation timestamp in
// its lede paragraph. The span still exists with just the relative time text.
func TestFileDetailLedeSaysAddedNotCreated(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)

	rec, err := a.Store.Create("file", map[string]any{"title": "Readme"})
	if err != nil {
		t.Fatal(err)
	}

	page := get(t, h, "/t/file/"+rec.ID).Body.String()

	if strings.Contains(page, ">Created ") {
		t.Error("file detail page should not show \"Created\" — it uses the database field name instead of plain language")
	}
	// Check that the span exists with timestamp text (no label prefix).
	re := regexp.MustCompile(`class="sw-detail__when[^"]*"><time[^>]*>([^<]+)</time>`)
	matches := re.FindStringSubmatch(page)
	if len(matches) < 2 {
		t.Error("file detail page lede should have a sw-detail__when span with the creation time")
		return
	}
	whenText := matches[1]
	if strings.HasPrefix(whenText, "Added ") || strings.HasPrefix(whenText, "Created ") {
		t.Errorf("file detail page lede should not start with a label prefix; got %q", whenText)
	}
}

// TestPersonDetailLedeNoAddedLabel asserts that a person detail page does not
// show "Added" or "Created" as a label prefix before the creation timestamp.
func TestPersonDetailLedeSaysAddedNotCreated(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)

	rec, err := a.Store.Create("person", map[string]any{"name": "Ada"})
	if err != nil {
		t.Fatal(err)
	}

	page := get(t, h, "/t/person/"+rec.ID).Body.String()

	if strings.Contains(page, ">Created ") {
		t.Error("person detail page should not show \"Created\" — it uses the database field name instead of plain language")
	}
	re := regexp.MustCompile(`class="sw-detail__when[^"]*"><time[^>]*>([^<]+)</time>`)
	matches := re.FindStringSubmatch(page)
	if len(matches) < 2 {
		t.Error("person detail page lede should have a sw-detail__when span with the creation time")
		return
	}
	whenText := matches[1]
	if strings.HasPrefix(whenText, "Added ") || strings.HasPrefix(whenText, "Created ") {
		t.Errorf("person detail page lede should not start with a label prefix; got %q", whenText)
	}
}

// TestActionDetailLedeNoStartedLabel asserts that an action detail page does
// not show "Started" or "Created" as a label prefix before the creation timestamp.
func TestActionDetailLedeSaysStartedNotCreated(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)

	rec, err := a.Store.Create("action", map[string]any{"title": "Sync data"})
	if err != nil {
		t.Fatal(err)
	}

	page := get(t, h, "/t/action/"+rec.ID).Body.String()

	if strings.Contains(page, ">Created ") {
		t.Error("action detail page should not show \"Created\" — it uses the database field name instead of plain language")
	}
	re := regexp.MustCompile(`class="sw-detail__when[^"]*"><time[^>]*>([^<]+)</time>`)
	matches := re.FindStringSubmatch(page)
	if len(matches) < 2 {
		t.Error("action detail page lede should have a sw-detail__when span with the creation time")
		return
	}
	whenText := matches[1]
	if strings.HasPrefix(whenText, "Started ") || strings.HasPrefix(whenText, "Created ") {
		t.Errorf("action detail page lede should not start with a label prefix; got %q", whenText)
	}
}

// TestTaskDetailLedeNoAddedLabel asserts that a task detail page does not show
// "Added" or "Created" as a label prefix before the creation timestamp.
func TestTaskDetailLedeSaysAddedNotCreated(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)

	rec, err := a.Store.Create("task", map[string]any{"title": "Buy milk"})
	if err != nil {
		t.Fatal(err)
	}

	page := get(t, h, "/t/task/"+rec.ID).Body.String()

	if strings.Contains(page, ">Created ") {
		t.Error("task detail page should not show \"Created\" — it uses the database field name instead of plain language")
	}
	re := regexp.MustCompile(`class="sw-detail__when[^"]*"><time[^>]*>([^<]+)</time>`)
	matches := re.FindStringSubmatch(page)
	if len(matches) < 2 {
		t.Error("task detail page lede should have a sw-detail__when span with the creation time")
		return
	}
	whenText := matches[1]
	if strings.HasPrefix(whenText, "Added ") || strings.HasPrefix(whenText, "Created ") {
		t.Errorf("task detail page lede should not start with a label prefix; got %q", whenText)
	}
}

// TestNoteDetailLedeNoAddedLabel asserts that a note detail page does not show
// "Added" or any other label prefix before the creation timestamp.
func TestNoteDetailLedeSaysAddedNotCreated(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)

	rec, err := a.Store.Create("note", map[string]any{"title": "Plans"})
	if err != nil {
		t.Fatal(err)
	}

	page := get(t, h, "/t/note/"+rec.ID).Body.String()

	if strings.Contains(page, ">Created ") {
		t.Error("note detail page should not show \"Created\" — it uses the database field name instead of plain language")
	}
	re := regexp.MustCompile(`class="sw-detail__when[^"]*"><time[^>]*>([^<]+)</time>`)
	matches := re.FindStringSubmatch(page)
	if len(matches) < 2 {
		t.Error("note detail page lede should have a sw-detail__when span with the creation time")
		return
	}
	whenText := matches[1]
	if strings.HasPrefix(whenText, "Added ") || strings.HasPrefix(whenText, "Created ") {
		t.Errorf("note detail page lede should not start with a label prefix; got %q", whenText)
	}
}

// TestProjectDetailLedeNoAddedLabel asserts that a project detail page does not
// show "Added" or "Created" as a label prefix before the creation timestamp.
func TestProjectDetailLedeSaysAddedNotCreated(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)

	rec, err := a.Store.Create("project", map[string]any{"title": "My Project"})
	if err != nil {
		t.Fatal(err)
	}

	page := get(t, h, "/t/project/"+rec.ID).Body.String()

	if strings.Contains(page, ">Created ") {
		t.Error("project detail page should not show \"Created\" — it uses the database field name instead of plain language")
	}
	re := regexp.MustCompile(`class="sw-detail__when[^"]*"><time[^>]*>([^<]+)</time>`)
	matches := re.FindStringSubmatch(page)
	if len(matches) < 2 {
		t.Error("project detail page lede should have a sw-detail__when span with the creation time")
		return
	}
	whenText := matches[1]
	if strings.HasPrefix(whenText, "Added ") || strings.HasPrefix(whenText, "Created ") {
		t.Errorf("project detail page lede should not start with a label prefix; got %q", whenText)
	}
}
