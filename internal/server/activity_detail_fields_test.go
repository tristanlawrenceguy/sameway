package server_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
)

// TestActivityDetailBeforeFieldIsReadable checks that a set-type activity
// entry shows a human-readable value in the "Before" field on its detail page,
// not raw JSON like { "value": "medium" }. It should show just "Medium".
// This covers acceptance item 1.
func TestActivityDetailBeforeFieldIsReadable(t *testing.T) {
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

// TestAPIActivityBeforeFieldIsReadable checks that GET /api/activity/{id} returns a
// cleaned, human-readable string for the "before" field on set-type entries. The
// Before value should be just the text (e.g. "Medium") not a JSON object with a
// "value" key. This covers acceptance item 3.
func TestAPIActivityBeforeFieldIsReadable(t *testing.T) {
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
		Title  string         `json:"title"`
		Fields map[string]any `json:"fields"`
	}
	resp := get(t, h, "/api/activity/"+rec.ID)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.Code)
	}
	json.Unmarshal(resp.Body.Bytes(), &one)

	beforeVal := one.Fields["before"]

	// The before field should be a plain string, not a map.
	if m, ok := beforeVal.(map[string]any); ok {
		t.Errorf("API 'before' field must be a plain string for set-type entries, got JSON object: %+v", m)
	}

	// The value should be capitalised "Medium".
	if s, ok := beforeVal.(string); !ok || strings.ToLower(s) != "medium" {
		t.Errorf("API 'before' field should be a string like 'Medium', got %T: %v", beforeVal, beforeVal)
	}
}

// TestAPIActivityTargetFieldIsReadable checks that GET /api/activity/{id} returns the
// human-readable setting name for the "target" field on set-type entries. The API
// should return "Text size" not "ui.text". This covers acceptance item 3.
func TestAPIActivityTargetFieldIsReadable(t *testing.T) {
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
		Title  string         `json:"title"`
		Fields map[string]any `json:"fields"`
	}
	resp := get(t, h, "/api/activity/"+rec.ID)
	json.Unmarshal(resp.Body.Bytes(), &one)

	targetVal := one.Fields["target"]

	if s, ok := targetVal.(string); !ok || strings.Contains(s, "ui.text") {
		t.Errorf("API 'target' field should be the human-readable name (e.g. 'Text size'), got %q", targetVal)
	}

	if !strings.Contains(fmt.Sprint(targetVal), "text size") && !strings.Contains(fmt.Sprint(targetVal), "Text size") {
		t.Errorf("API 'target' field should show 'Text size', not 'ui.text'; got %v", targetVal)
	}
}

// TestAPIAndHTMLSetEntryFieldsMatch checks that the cleaned values returned by the
// API for a set-type activity entry match what the HTML detail page renders. This
// covers acceptance item 3 (consistency between surfaces).
func TestAPIAndHTMLSetEntryFieldsMatch(t *testing.T) {
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
		Title  string         `json:"title"`
		Fields map[string]any `json:"fields"`
	}
	resp := get(t, h, "/api/activity/"+rec.ID)
	json.Unmarshal(resp.Body.Bytes(), &one)

	htmlBody := get(t, h, "/t/activity/"+rec.ID).Body.String()

	// API before field should match what the HTML renders.
	apiBefore := fmt.Sprint(one.Fields["before"])
	if !strings.Contains(said(htmlBody), apiBefore) && !strings.Contains(said(htmlBody), strings.ToLower(apiBefore)) {
		t.Errorf("API 'before' value %q should appear in HTML detail page\nHTML said: %q", apiBefore, said(htmlBody))
	}

	// API target field should match what the HTML renders.
	apiTarget := fmt.Sprint(one.Fields["target"])
	if !strings.Contains(said(htmlBody), apiTarget) && !strings.Contains(said(htmlBody), strings.ToLower(apiTarget)) {
		t.Errorf("API 'target' value %q should appear in HTML detail page\nHTML said: %q", apiTarget, said(htmlBody))
	}

	// API detail field should match what the HTML renders.
	apiDetail := fmt.Sprint(one.Fields["detail"])
	if !strings.Contains(said(htmlBody), apiDetail) && !strings.Contains(said(htmlBody), strings.ToLower(apiDetail)) {
		t.Errorf("API 'detail' value %q should appear in HTML detail page\nHTML said: %q", apiDetail, said(htmlBody))
	}
}
