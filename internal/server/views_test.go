package server_test

import (
	"strings"
	"testing"
)

// TestDetailPageSkipsEmptyFields ensures the definition list on a detail page
// does not emit <dt>/<dd> pairs for fields whose display value is empty.
func TestDetailPageSkipsEmptyFields(t *testing.T) {
	a, h := newApp(t)

	// Create a note with only title — body, tags, status, pinned are all nil/zero.
	rec, err := a.Store.Create("note", map[string]any{"title": "Just a title"})
	if err != nil {
		t.Fatal(err)
	}
	body := get(t, h, "/t/note/"+rec.ID).Body.String()

	// Title appears; body is empty so it is skipped by the emptiness guard.
	if !strings.Contains(body, "<dt>Title</dt>") {
		t.Errorf("detail page missing <dt>Title</dt>\n%s", truncate(body))
	}

	// Tags must NOT appear since it was never set.
	if strings.Contains(body, "<dt>Tags</dt>") {
		t.Error("detail page should not show a Tags row for an empty field")
	}

	// Created and Updated always render even when no other data is present.
	if !strings.Contains(body, "<dt>Created</dt>") || !strings.Contains(body, "<dt>Updated</dt>") {
		t.Error("detail page should always show Created and Updated rows")
	}
}

// TestDetailPageShowsNonEmptyFields ensures that fields with real values still
// render correctly when empty fields are skipped.
func TestDetailPageShowsNonEmptyFields(t *testing.T) {
	a, h := newApp(t)

	rec, err := a.Store.Create("note", map[string]any{
		"title":  "With tags",
		"tags":   []any{"a", "b"},
		"pinned": true,
	})
	if err != nil {
		t.Fatal(err)
	}
	body := get(t, h, "/t/note/"+rec.ID).Body.String()

	if !strings.Contains(body, "<dt>Tags</dt>") || !strings.Contains(body, "a, b") {
		t.Errorf("detail page should show tag values a, b\n%s", truncate(body))
	}

	if !strings.Contains(body, "<dt>Pinned</dt>") || strings.Contains(body, "Pinned no") {
		t.Error("detail page should show Pinned yes for true bools")
	}
}

// TestDetailPageSkipsEmptyFieldsActivity ensures activity detail pages also
// skip empty fields in their definition list.
func TestDetailPageSkipsEmptyFieldsActivity(t *testing.T) {
	a, h := newApp(t)

	rec, err := a.Store.Create("activity", map[string]any{
		"actor":  "human",
		"action": "added",
	})
	if err != nil {
		t.Fatal(err)
	}
	body := get(t, h, "/t/activity/"+rec.ID).Body.String()

	if !strings.Contains(body, "<dt>Actor</dt>") || !strings.Contains(body, "<dt>Action</dt>") {
		t.Errorf("detail page should show Actor and Action rows\n%s", truncate(body))
	}

	// Fields like target, target_id, detail were never set — their labels must be absent.
	for _, label := range []string{"Target", "Detail"} {
		if strings.Contains(body, "<dt>"+label+"</dt>") {
			t.Errorf("detail page should not show <%s> for an empty field on activity\n%s", label, truncate(body))
		}
	}

	if !strings.Contains(body, "<dt>Created</dt>") || !strings.Contains(body, "<dt>Updated</dt>") {
		t.Error("detail page should always show Created and Updated rows")
	}
}

// TestDetailPageSkipsEmptyFieldsMessage ensures message detail pages also skip
// empty fields in their definition list.
func TestDetailPageSkipsEmptyFieldsMessage(t *testing.T) {
	a, h := newApp(t)

	rec, err := a.Store.Create("message", map[string]any{
		"role":    "user",
		"content": "hello world",
	})
	if err != nil {
		t.Fatal(err)
	}
	body := get(t, h, "/t/message/"+rec.ID).Body.String()

	if !strings.Contains(body, "<dt>Role</dt>") || !strings.Contains(body, "<dt>Content</dt>") {
		t.Errorf("detail page should show Role and Content rows\n%s", truncate(body))
	}

	// Changes was never set — its label must be absent.
	if strings.Contains(body, "<dt>Changes</dt>") {
		t.Error("detail page should not show <dt>Changes</dt> for an empty field on message")
	}

	if !strings.Contains(body, "<dt>Created</dt>") || !strings.Contains(body, "<dt>Updated</dt>") {
		t.Error("detail page should always show Created and Updated rows")
	}
}

// TestDetailPageNonEmptyStatusStillRenders verifies that a non-empty enum field
// like "status" (which defaults to "draft") still renders correctly.
func TestDetailPageNonEmptyStatusStillRenders(t *testing.T) {
	a, h := newApp(t)

	rec, err := a.Store.Create("note", map[string]any{
		"title":  "Draft note",
		"status": "draft",
	})
	if err != nil {
		t.Fatal(err)
	}
	body := get(t, h, "/t/note/"+rec.ID).Body.String()

	if !strings.Contains(body, "<dt>Status</dt>") || !strings.Contains(body, "Draft") {
		t.Errorf("detail page should show Status with value 'draft'\n%s", truncate(body))
	}
}
