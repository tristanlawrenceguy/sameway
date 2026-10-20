package server_test

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

// An error-role message shows a sanitized title on its detail page — no raw
// provider names like "claude" or "exit status 1" appear in the visible H1.
// Old stored records get this treatment at render time, not just at write
// time. Covers acceptance item 2.
func TestErrorMessageDetailShowsSanitizedTitle(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)

	msg, err := a.Store.Create("message", map[string]any{
		"role":    "error",
		"content": "claude: exit status 1: /home/user/project/main.go",
	})
	if err != nil {
		t.Fatal(err)
	}

	rec := get(t, h, "/t/message/"+msg.ID)
	wantStatus(t, rec, http.StatusOK)
	body := rec.Body.String()

	rawContent := "claude: exit status 1: /home/user/project/main.go"
	if strings.Contains(body, rawContent) {
		t.Errorf("error message detail page must not show the raw content as heading\nbody:\n%s", excerpt(body))
	}

	// The sanitized version (plain "An error occurred") should appear.
	if !strings.Contains(strings.ToLower(body), "an error occurred") &&
		!strings.Contains(strings.ToLower(body), "error") {
		t.Errorf("detail page should show a sanitized title for the error message\nbody:\n%s", excerpt(body))
	}

	// Provider name must not appear in H1.
	if strings.Contains(body, ">claude</h1>") || strings.Contains(body, ">Claude</h1>") {
		t.Errorf("detail page must not show provider name \"claude\" in the heading\nbody:\n%s", excerpt(body))
	}

	// Exit status text must be removed.
	if strings.Contains(strings.ToLower(body), "exit status") {
		t.Errorf("detail page must not show \"exit status\" in visible text\nbody:\n%s", excerpt(body))
	}

	// File path must not appear in the heading.
	if strings.Contains(body, "/home/user/project/main.go") {
		t.Errorf("detail page must not show file paths\nbody:\n%s", excerpt(body))
	}
}

// The API list endpoint returns sanitized titles for error-role messages — no
// raw provider names appear in the JSON title field. Covers acceptance item 3.
func TestErrorMessageAPIReturnsSanitizedTitle(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)

	_, err := a.Store.Create("message", map[string]any{
		"role":    "error",
		"content": "anthropic: key expired: please check your API settings",
	})
	if err != nil {
		t.Fatal(err)
	}

	rec := get(t, h, "/api/message?where=role%3Derror")
	wantStatus(t, rec, http.StatusOK)
	body, _ := io.ReadAll(rec.Body)

	var result map[string]any
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("API response is not valid JSON: %v\nbody: %s", err, string(body))
	}

	recordsSlice, ok := result["records"].([]any)
	if !ok || len(recordsSlice) == 0 {
		t.Fatalf("expected at least one record in API response\nbody: %s", string(body))
	}

	for _, raw := range recordsSlice {
		recMap, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		titleAny, hasTitle := recMap["title"]
		if !hasTitle {
			t.Fatalf("API record missing title field\nbody: %s", string(body))
		}
		titleStr, _ := titleAny.(string)

		if strings.Contains(strings.ToLower(titleStr), "anthropic") {
			t.Errorf("API title must not contain provider name \"anthropic\": got %q", titleStr)
		}

		if strings.Contains(strings.ToLower(titleStr), "key expired") {
			t.Errorf("API title must not show raw error detail like \"key expired\": got %q", titleStr)
		}
	}

	// The sanitized version should appear somewhere in the response.
	bodyLower := strings.ToLower(string(body))
	if !strings.Contains(bodyLower, "an error occurred") &&
		!strings.Contains(bodyLower, `"error"`) {
		t.Errorf("API response should contain a sanitized title\nbody:\n%s", string(body))
	}
}
