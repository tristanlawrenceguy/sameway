package server_test

import (
	"net/http"
	"strings"
	"testing"
)

// Reminder snippet body text does not contain raw field-label prefixes like
// "Type:" or "State:" before enum values (acceptance 3). The reminder's kind
// and state should appear without the technical field-name prefix.
func TestReminderSnippetHasNoRawFieldLabelPrefix(t *testing.T) {
	t.Parallel()
	_, h := newApp(t)

	// Create a reminder with both kind=alarm (enum) and state=set (enum).
	wantStatus(t, postJSON(t, h, http.MethodPost, "/api/reminder", map[string]any{
		"title": "Floss teeth",
		"at":    "2026-10-18T09:00:00Z",
		"kind":  "alarm",
		"state": "set",
	}), http.StatusCreated)

	var out struct {
		Count int
		Hits  []struct{ Type, Title, Href, Snippet string }
	}
	decode(t, get(t, h, "/api/search?q=floss&type=reminder"), &out)
	if out.Count == 0 {
		t.Fatalf("one reminder matches 'floss', got %d", out.Count)
	}
	snippet := out.Hits[0].Snippet

	// Raw field-label prefixes must not appear in the snippet.
	for _, label := range []string{"Type:", "State:"} {
		if strings.Contains(snippet, label) {
			t.Errorf("reminder snippet body must not contain raw field-label prefix %q; got: %q", label, snippet)
		}
	}

	// The enum value labels should still appear.
	if !strings.Contains(snippet, "Alarm") {
		t.Errorf("the kind value 'Alarm' should appear in the reminder snippet; got: %q", snippet)
	}
	if !strings.Contains(snippet, "Waiting") && !strings.Contains(snippet, "Set") {
		t.Errorf("the state value should appear in the reminder snippet; got: %q", snippet)
	}
}
