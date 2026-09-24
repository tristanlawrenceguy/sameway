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
	body := get(t, h, "/t/note/"+rec.ID+fieldsView).Body.String()

	// Title is excluded from the dl (it's in the h1); body is empty so it is skipped.
	if strings.Contains(body, "<dt>Title</dt>") {
		t.Error("detail page should not show <dt>Title</dt> — title is the heading")
	}

	// Tags must NOT appear since it was never set.
	if strings.Contains(body, "<dt>Tags</dt>") {
		t.Error("detail page should not show a Tags row for an empty field")
	}

	// Created and Updated always render even when no other data is present.
	if !strings.Contains(body, `class="sw-detail__when sw-muted sw-small">Created `) {
		t.Error("detail page should always say when the record was made")
	}
}

// TestDetailPageShowsNonEmptyFields ensures that fields with real values still
// render correctly when empty fields are skipped. Bool and enum types are
// excluded from the dl because they appear as chips/heading instead.
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
	body := get(t, h, "/t/note/"+rec.ID+fieldsView).Body.String()

	if !strings.Contains(body, "<dt>Tags</dt>") || !strings.Contains(body, "a, b") {
		t.Errorf("detail page should show tag values a, b\n%s", truncate(body))
	}

	// Pinned (bool) is excluded from the dl — it appears as a chip/mark instead.
	if strings.Contains(body, "<dt>Pinned</dt>") {
		t.Error("detail page should not show <dt>Pinned</dt> in the dl")
	}
}

// TestDetailPageSkipsEmptyFieldsActivity ensures activity detail pages also
// skip empty fields in their definition list. The enum actor field is excluded
// from the dl because it appears as a chip; summary (title) and Action remain.
func TestDetailPageSkipsEmptyFieldsActivity(t *testing.T) {
	a, h := newApp(t)

	rec, err := a.Store.Create("activity", map[string]any{
		"actor":  "human",
		"action": "added",
	})
	if err != nil {
		t.Fatal(err)
	}
	body := get(t, h, "/t/activity/"+rec.ID+fieldsView).Body.String()

	// Actor (enum) is excluded from the dl; Action (string) still renders.
	if strings.Contains(body, "<dt>Actor</dt>") {
		t.Error("detail page should not show <dt>Actor</dt> in the dl")
	}
	if !strings.Contains(body, "<dl class=\"sw-fields\"") || !strings.Contains(body, "<dt>Action</dt>") || !strings.Contains(body, "added") {
		t.Errorf("detail page should show Action row\n%s", truncate(body))
	}

	// Fields like target, target_id, detail were never set — their labels must be absent.
	for _, label := range []string{"Target", "Detail"} {
		if strings.Contains(body, "<dt>"+label+"</dt>") {
			t.Errorf("detail page should not show <%s> for an empty field on activity\n%s", label, truncate(body))
		}
	}

	if !strings.Contains(body, `class="sw-detail__when sw-muted sw-small">Created `) {
		t.Error("detail page should always say when the record was made")
	}
}

// TestDetailPageSkipsEmptyFieldsMessage ensures message detail pages also skip
// empty fields in their definition list. The enum role and title (content) are
// excluded from the dl because they appear as chips/heading instead.
func TestDetailPageSkipsEmptyFieldsMessage(t *testing.T) {
	a, h := newApp(t)

	rec, err := a.Store.Create("message", map[string]any{
		"role":    "user",
		"content": "hello world",
	})
	if err != nil {
		t.Fatal(err)
	}
	body := get(t, h, "/t/message/"+rec.ID+fieldsView).Body.String()

	// Role (enum) and Content (title) are excluded from the dl.
	if strings.Contains(body, "<dt>Role</dt>") {
		t.Error("detail page should not show <dt>Role</dt> in the dl")
	}
	if strings.Contains(body, "<dt>Content</dt>") {
		t.Error("detail page should not show <dt>Content</dt> — content is the title/h1")
	}

	// Changes was never set — its label must be absent.
	if strings.Contains(body, "<dt>Changes</dt>") {
		t.Error("detail page should not show <dt>Changes</dt> for an empty field on message")
	}

	if !strings.Contains(body, `class="sw-detail__when sw-muted sw-small">Created `) {
		t.Error("detail page should always say when the record was made")
	}
}

// TestDetailPageNonEmptyStatusStillRenders verifies that a non-empty enum field
// like "status" (which defaults to "draft") is shown as a chip/heading, not
// repeated in the dl. The status badge still appears on the page.
func TestDetailPageNonEmptyStatusStillRenders(t *testing.T) {
	a, h := newApp(t)

	rec, err := a.Store.Create("note", map[string]any{
		"title":  "Draft note",
		"status": "draft",
	})
	if err != nil {
		t.Fatal(err)
	}
	body := get(t, h, "/t/note/"+rec.ID+fieldsView).Body.String()

	// Status (enum) is excluded from the dl — it appears as a chip.
	if strings.Contains(body, "<dt>Status</dt>") {
		t.Error("detail page should not show <dt>Status</dt> in the dl")
	}

	// The status badge still renders on the page.
	page := get(t, h, "/t/note/"+rec.ID+fieldsView).Body.String()
	if !strings.Contains(page, "sw-badge--info") || !strings.Contains(page, "Draft") {
		t.Errorf("page should show status as a chip\n%s", truncate(body))
	}

	// The title is still excluded from the dl.
	if strings.Contains(body, "<dt>Title</dt>") {
		t.Error("detail page should not show <dt>Title</dt> in the dl")
	}
}
