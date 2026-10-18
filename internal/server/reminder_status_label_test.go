package server_test

import (
	"net/http"
	"strings"
	"testing"
)

// Reminder list rows present their state value as a badge chip without a raw
// schema field name prefix like "Status:". The badge itself communicates the value.
func TestReminderListRowHasNoStatusLabelPrefix(t *testing.T) {
	t.Parallel()
	_, h := newAppAt(t, tonight)

	now := "2026-03-11T08:00:00Z" // same as today so it rings
	wantStatus(t, postJSON(t, h, http.MethodPost, "/api/reminder", map[string]any{
		"title": "Call the dentist",
		"at":    now,
		"state": "rang",
	}), http.StatusCreated)

	body := get(t, h, "/t/reminder").Body.String()

	// The row must show a visible badge with the state value.
	if !strings.Contains(body, ">Rang</span>") {
		t.Errorf("fired reminder row should show the \"Rang\" badge; page:\n%s", body)
	}
	// No raw schema field name prefix before the badge.
	if strings.Contains(body, "Status: Rang") {
		t.Errorf("reminder list row must not show \"Status:\" before the badge; found near Rang: %s", around(body, "Rang"))
	}
}
