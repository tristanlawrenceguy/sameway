package server_test

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

// TestActivityDetailBeforeFieldIsReadable checks that a set-type activity
// entry shows a human-readable value in the "Before" field on its detail page,
// not raw JSON like { "value": "medium" }. It should show just "Medium".
// This covers acceptance item 1.
func TestActivityDetailBeforeFieldIsReadable(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)

	rec, err := a.Store.Create("activity", map[string]any{
		"summary": "Assistant changed text size to Large",
		"actor":   "assistant",
		"action":  "set",
		"target":  "ui.text",
		"detail":  "large",
		"before":  map[string]any{"value": "medium"},
	})
	if err != nil {
		t.Fatal(err)
	}

	body := get(t, h, "/t/activity/"+rec.ID).Body.String()

	// Acceptance 1: the Before field must not show raw JSON.
	if strings.Contains(body, `{`) && strings.Contains(body, `"value"`) {
		t.Errorf("Before field on activity detail page must not show raw JSON like \"{\\\"value\\\": \\\"medium\\\"}\"")
	}

	// The cleaned value "Medium" should appear somewhere in the Before row.
	if !strings.Contains(said(body), "medium") && !strings.Contains(said(body), "Medium") {
		t.Errorf("Before field should show 'Medium', not raw JSON\n\ngot said: %q", said(body))
	}
}

// TestActivityDetailTargetFieldIsReadable checks that a set-type activity entry
// shows the human-readable setting name ("Text size") in the "Target" field on
// its detail page, not the internal path like "ui.text". This covers acceptance
// item 2.
func TestActivityDetailTargetFieldIsReadable(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)

	rec, err := a.Store.Create("activity", map[string]any{
		"summary": "Assistant changed text size to Large",
		"actor":   "assistant",
		"action":  "set",
		"target":  "ui.text",
		"detail":  "large",
		"before":  map[string]any{"value": "medium"},
	})
	if err != nil {
		t.Fatal(err)
	}

	body := get(t, h, "/t/activity/"+rec.ID).Body.String()

	// The Target field should show the human-readable name.
	if strings.Contains(said(body), "target ui.text") ||
		strings.Contains(said(body), "Target ui.text") {
		t.Errorf("Target field on activity detail page must not show raw path 'ui.text'\n\ngot said: %q", said(body))
	}

	if !strings.Contains(said(body), "text size") && !strings.Contains(said(body), "Text size") {
		t.Errorf("Target field should show 'Text size', not 'ui.text'\n\ngot said: %q", said(body))
	}
}

// TestActivityDetailAfterFieldIsReadable checks that the detail/value shown for a
// set-type activity entry is capitalised (e.g. "Large" instead of "large"). This
// is part of acceptance item 2 — the detail field should be readable.
func TestActivityDetailAfterFieldIsReadable(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)

	rec, err := a.Store.Create("activity", map[string]any{
		"summary": "Assistant changed text size to Large",
		"actor":   "assistant",
		"action":  "set",
		"target":  "ui.text",
		"detail":  "large",
		"before":  map[string]any{"value": "medium"},
	})
	if err != nil {
		t.Fatal(err)
	}

	body := get(t, h, "/t/activity/"+rec.ID).Body.String()

	// The Detail field should show the capitalised value.
	detailValue := strings.TrimSpace(said(body))
	if !strings.Contains(detailValue, "detail large") && !strings.Contains(detailValue, "Detail Large") && !strings.Contains(detailValue, "Large") {
		t.Errorf("Detail field on activity detail page should show 'Large', not 'large'\n\ngot said: %q", detailValue)
	}

	// Check that the value is capitalised.
	if !strings.Contains(said(body), "Large") {
		t.Errorf("Detail field should show 'Large' (capitalised)\n\ngot said: %q", detailValue)
	}
}

// An entry's fields over the API stay the data they are, for an agent to
// act on (undo reads before as it is); what the entry says, in words, is
// beside them as said, the very words its page shows.
func TestAPIActivityKeepsItsDataAndSaysItAsThePageDoes(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)
	rec, err := a.Store.Create("activity", map[string]any{
		"summary": "Assistant changed text size to Large",
		"actor":   "assistant",
		"action":  "set",
		"target":  "ui.text",
		"detail":  "large",
		"before":  map[string]any{"value": "medium"},
	})
	if err != nil {
		t.Fatal(err)
	}
	var one struct {
		Fields map[string]any   `json:"fields"`
		Said   []map[string]any `json:"said"`
	}
	json.Unmarshal(get(t, h, "/api/activity/"+rec.ID).Body.Bytes(), &one)
	if before, ok := one.Fields["before"].(map[string]any); !ok || before["value"] != "medium" || one.Fields["target"] != "ui.text" {
		t.Errorf("the fields are the entry's data as stored: %v", one.Fields)
	}
	page := said(get(t, h, "/t/activity/"+rec.ID).Body.String())
	found := false
	for _, item := range one.Said {
		label, value := fmt.Sprint(item["label"]), fmt.Sprint(item["value"])
		if label == "Text size was" && value == "Medium" {
			found = true
		}
		if !strings.Contains(page, label) || !strings.Contains(page, value) {
			t.Errorf("the page says what the API says: %s %s not in %q", label, value, page)
		}
	}
	if !found {
		t.Errorf("said is in words: %v", one.Said)
	}
}
