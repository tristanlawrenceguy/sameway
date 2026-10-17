package server_test

import (
	"net/http"
	"strings"
	"testing"
)

// Reminder detail pages must label their state (status) value with "Status:" so it is
// clear what the value means, instead of showing bare enum text like "Rang" or "Done".
// Acceptance items 1–2.
func TestReminderDetailPageLabelsStateValueWithStatusLabel(t *testing.T) {
	t.Parallel()
	_, h := newAppAt(t, tonight)

	now := "2026-03-11T08:00:00Z" // same as today so it rings
	var rec struct{ ID string }
	decode(t, postJSON(t, h, http.MethodPost, "/api/reminder", map[string]any{
		"title": "Call the dentist",
		"at":    now,
		"state": "rang",
	}), &rec)

	page := get(t, h, "/t/reminder/"+rec.ID).Body.String()

	// The page must contain a visible "Status:" label before the badge value.
	if !strings.Contains(page, "Status:") {
		t.Errorf("reminder detail page must label its state value with \"Status: \" before the badge; found near Rang:\n%s", around(page, "Rang"))
	}

	// The badge must still render the value.
	if !strings.Contains(page, ">Rang</span>") {
		t.Errorf("fired reminder detail page should show the \"Rang\" badge; page:\n%s", truncate(page))
	}
}

// Reminder detail pages without a state value (default "set") must not show any
// status line at all — only their title link and time elements. Acceptance item 2.
func TestReminderDetailPageHasNoStatusLineWithoutState(t *testing.T) {
	t.Parallel()
	_, h := newAppAt(t, tonight)

	var rec struct{ ID string }
	decode(t, postJSON(t, h, http.MethodPost, "/api/reminder", map[string]any{
		"title": "Take medicine tomorrow",
		"at":    "2026-03-12T08:00:00Z",
	}), &rec)

	page := get(t, h, "/t/reminder/"+rec.ID).Body.String()

	// A reminder without a non-default state must not show any Status: label.
	if strings.Contains(page, "Status:") {
		t.Errorf("reminder detail page with default state must not show a \"Status:\" line; found in:\n%s", truncate(page))
	}

	// The title link and time elements must still be present.
	if !strings.Contains(page, `href="/t/reminder/`+rec.ID) {
		t.Errorf("detail page should contain the reminder's title link\n%s", truncate(page))
	}
}
