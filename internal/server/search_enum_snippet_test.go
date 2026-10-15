package server_test

// Search result body snippets must not include raw enum field values like
// "Draft" or "Published". These are metadata, not searchable content; they
// appear separated by middots from prose and confuse people scanning results.
// Acceptance 3 of task 0286 (backlog 0708).

import (
	"net/http"
	"strings"
	"testing"
)

// TestSearchSnippetDoesNotIncludeEnumValue verifies that the search API
// does not return enum values like "Published" inside hit snippets.
func TestSearchSnippetDoesNotIncludeEnumValue(t *testing.T) {
	t.Parallel()
	_, h := newApp(t)

	var rec struct{ ID string }
	decode(t, postJSON(t, h, http.MethodPost, "/api/note", map[string]any{
		"title":  "Call the plumber",
		"body":   "About the kitchen tap and the bathroom sink.",
		"status": "published",
	}), &rec)

	resp := get(t, h, "/api/search?q=kitchen")
	wantStatus(t, resp, http.StatusOK)
	var result map[string]any
	decode(t, resp, &result)

	hitsAny, ok := result["hits"].([]any)
	if !ok || len(hitsAny) == 0 {
		t.Fatal("expected at least one hit in search results")
	}
	hitMap := hitsAny[0].(map[string]any)
	snippet := hitMap["snippet"].(string)

	if strings.Contains(snippet, "Published") || strings.Contains(snippet, "published") {
		t.Errorf("search snippet must not include the enum value 'Published'; found in:\n%s", truncate(snippet))
	}
}

// TestSearchSnippetDoesNotIncludeDraftValue verifies that a note with status
// "draft" does not leak its status into the search body text.
func TestSearchSnippetDoesNotIncludeDraftValue(t *testing.T) {
	t.Parallel()
	_, h := newApp(t)

	postJSON(t, h, http.MethodPost, "/api/note", map[string]any{
		"title":  "Work in progress",
		"body":   "This is still being written about the project.",
		"status": "draft",
	})

	resp := get(t, h, "/api/search?q=project")
	wantStatus(t, resp, http.StatusOK)
	var result map[string]any
	decode(t, resp, &result)

	hitsAny, ok := result["hits"].([]any)
	if !ok || len(hitsAny) == 0 {
		t.Fatal("expected at least one hit in search results")
	}
	hitMap := hitsAny[0].(map[string]any)
	snippet := hitMap["snippet"].(string)

	if strings.Contains(snippet, "Draft") || strings.Contains(snippet, "draft") {
		t.Errorf("search snippet must not include the enum value 'Draft'; found in:\n%s", truncate(snippet))
	}
}

// TestSearchSnippetStillIncludesBodyText verifies that non-enum body text is
// still included in snippets — only enum values are excluded.
func TestSearchSnippetStillIncludesBodyText(t *testing.T) {
	t.Parallel()
	_, h := newApp(t)

	postJSON(t, h, http.MethodPost, "/api/note", map[string]any{
		"title":  "Kitchen renovation plans",
		"body":   "We need to replace the kitchen tap and update the sink.",
		"status": "draft",
	})

	resp := get(t, h, "/api/search?q=kitchen")
	wantStatus(t, resp, http.StatusOK)
	var result map[string]any
	decode(t, resp, &result)

	hitsAny, ok := result["hits"].([]any)
	if !ok || len(hitsAny) == 0 {
		t.Fatal("expected at least one hit in search results")
	}
	hitMap := hitsAny[0].(map[string]any)
	snippet := hitMap["snippet"].(string)

	if !strings.Contains(snippet, "kitchen") && !strings.Contains(snippet, "Kitchen") {
		t.Errorf("search snippet should include body text matching the query; found:\n%s", truncate(snippet))
	}
}

// TestSearchSnippetDoesNotIncludeProjectStatus verifies that a project's
// status enum value does not appear in search snippets.
func TestSearchSnippetDoesNotIncludeProjectStatus(t *testing.T) {
	t.Parallel()
	_, h := newApp(t)

	postJSON(t, h, http.MethodPost, "/api/project", map[string]any{
		"title":  "Garden redesign plans",
		"notes":  "We are planning a full garden overhaul this spring.",
		"status": "active",
	})

	resp := get(t, h, "/api/search?q=garden")
	wantStatus(t, resp, http.StatusOK)
	var result map[string]any
	decode(t, resp, &result)

	hitsAny, ok := result["hits"].([]any)
	if !ok || len(hitsAny) == 0 {
		t.Fatal("expected at least one hit in search results")
	}
	hitMap := hitsAny[0].(map[string]any)
	snippet := hitMap["snippet"].(string)

	if strings.Contains(snippet, "Active") || strings.Contains(snippet, "active") {
		t.Errorf("search snippet must not include the enum value 'Active'; found in:\n%s", truncate(snippet))
	}
}
