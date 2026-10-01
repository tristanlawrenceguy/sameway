package server_test

import (
	"strings"
	"testing"
)

// TestFileDetailLedeSaysAddedNotCreated asserts that a file detail page uses
// "Added" instead of the database field name "Created" in its lede paragraph
// (acceptance item 4). The word Created must not appear as a human-readable
// label on a file page.
func TestFileDetailLedeSaysAddedNotCreated(t *testing.T) {
	a, h := newApp(t)

	rec, err := a.Store.Create("file", map[string]any{"title": "Readme"})
	if err != nil {
		t.Fatal(err)
	}

	page := get(t, h, "/t/file/"+rec.ID).Body.String()

	if strings.Contains(page, ">Created ") {
		t.Error("file detail page should not show \"Created\" — it uses the database field name instead of plain language")
	}
	if !strings.Contains(page, ">Added ") {
		t.Error("file detail page lede should say \"Added\" before the creation time")
	}
}

// TestPersonDetailLedeSaysAddedNotCreated asserts that a person detail page
// uses "Added" instead of the raw field name "Created" in its lede paragraph
// (acceptance item 5).
func TestPersonDetailLedeSaysAddedNotCreated(t *testing.T) {
	a, h := newApp(t)

	rec, err := a.Store.Create("person", map[string]any{"name": "Ada"})
	if err != nil {
		t.Fatal(err)
	}

	page := get(t, h, "/t/person/"+rec.ID).Body.String()

	if strings.Contains(page, ">Created ") {
		t.Error("person detail page should not show \"Created\" — it uses the database field name instead of plain language")
	}
	if !strings.Contains(page, ">Added ") {
		t.Error("person detail page lede should say \"Added\" before the creation time")
	}
}

// TestActionDetailLedeSaysStartedNotCreated asserts that an action detail page
// uses "Started" instead of "Created" in its lede paragraph (acceptance item 6).
// Actions are initiated actions, like habits.
func TestActionDetailLedeSaysStartedNotCreated(t *testing.T) {
	a, h := newApp(t)

	rec, err := a.Store.Create("action", map[string]any{"title": "Sync data"})
	if err != nil {
		t.Fatal(err)
	}

	page := get(t, h, "/t/action/"+rec.ID).Body.String()

	if strings.Contains(page, ">Created ") {
		t.Error("action detail page should not show \"Created\" — it uses the database field name instead of plain language")
	}
	if !strings.Contains(page, ">Started ") {
		t.Error("action detail page lede should say \"Started\" before the creation time, like habits do")
	}
}

// TestTaskDetailLedeSaysAddedNotCreated asserts that a task detail page uses
// "Added" instead of "Created" in its lede paragraph (acceptance item 2).
func TestTaskDetailLedeSaysAddedNotCreated(t *testing.T) {
	a, h := newApp(t)

	rec, err := a.Store.Create("task", map[string]any{"title": "Buy milk"})
	if err != nil {
		t.Fatal(err)
	}

	page := get(t, h, "/t/task/"+rec.ID).Body.String()

	if strings.Contains(page, ">Created ") {
		t.Error("task detail page should not show \"Created\" — it uses the database field name instead of plain language")
	}
	if !strings.Contains(page, ">Added ") {
		t.Error("task detail page lede should say \"Added\" before the creation time")
	}
}

// TestNoteDetailLedeSaysAddedNotCreated asserts that a note detail page uses
// "Added" instead of the raw field name "Created" in its lede paragraph
// (acceptance item 3).
func TestNoteDetailLedeSaysAddedNotCreated(t *testing.T) {
	a, h := newApp(t)

	rec, err := a.Store.Create("note", map[string]any{"title": "Plans"})
	if err != nil {
		t.Fatal(err)
	}

	page := get(t, h, "/t/note/"+rec.ID).Body.String()

	if strings.Contains(page, ">Created ") {
		t.Error("note detail page should not show \"Created\" — it uses the database field name instead of plain language")
	}
	if !strings.Contains(page, ">Added ") {
		t.Error("note detail page lede should say \"Added\" before the creation time")
	}
}

// TestProjectDetailLedeSaysAddedNotCreated asserts that a project detail page
// uses "Added" instead of "Created" in its lede paragraph (acceptance item 5).
func TestProjectDetailLedeSaysAddedNotCreated(t *testing.T) {
	a, h := newApp(t)

	rec, err := a.Store.Create("project", map[string]any{"title": "My Project"})
	if err != nil {
		t.Fatal(err)
	}

	page := get(t, h, "/t/project/"+rec.ID).Body.String()

	if strings.Contains(page, ">Created ") {
		t.Error("project detail page should not show \"Created\" — it uses the database field name instead of plain language")
	}
	if !strings.Contains(page, ">Added ") {
		t.Error("project detail page lede should say \"Added\" before the creation time")
	}
}
