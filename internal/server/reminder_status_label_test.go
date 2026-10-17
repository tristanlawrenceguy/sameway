package server_test

import (
	"net/http"
	"strings"
	"testing"
)

// Reminder list rows must label their state (status) value with "Status:" so it is
// clear what the value means, instead of showing bare enum text like "Rang" or "Done".
// Acceptance item 1.
func TestReminderListRowLabelsStateValueWithStatusLabel(t *testing.T) {
	t.Parallel()
	_, h := newAppAt(t, tonight)

	now := "2026-03-11T08:00:00Z" // same as today so it rings
	wantStatus(t, postJSON(t, h, http.MethodPost, "/api/reminder", map[string]any{
		"title": "Call the dentist",
		"at":    now,
		"state": "rang",
	}), http.StatusCreated)

	body := get(t, h, "/t/reminder").Body.String()

	// The row must show a visible "Status:" label before the badge value.
	if !strings.Contains(body, ">Rang</span>") {
		t.Errorf("fired reminder row should show the \"Rang\" badge; page:\n%s", body)
	}
	if !strings.Contains(body, "Status:") {
		t.Errorf("reminder list row must label its state value with \"Status: \" before the badge; found near Rang: %s", around(body, "Rang"))
	}
}
