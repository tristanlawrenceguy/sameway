package server_test

// The reminders list and detail pages must not expose the raw schema field
// name "kind" as visible text on any paragraph or lede (acceptance items 1–3).
// This is the last record type with this problem after tasks 0260–0264.

import (
	"net/http"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/server"
)

// TestReminderListDoesNotShowKindBadge verifies that the reminders listing page
// does not render a badge showing the reminder's kind enum value or its field
// name "kind". Each row must read naturally, like "Alarm · Today at 12:27pm",
// without exposing internal schema names (acceptance items 1 & 3).
func TestReminderListDoesNotShowKindBadge(t *testing.T) {
	a, h := newApp(t)

	_, err := a.Store.Create(server.ReminderType, map[string]any{
		"title": "Water the beans",
		"at":    "2026-10-05T14:30:00Z",
	})
	if err != nil {
		t.Fatal(err)
	}

	body := get(t, h, "/t/reminder").Body.String()

	// The list row must not contain a badge with the kind value "Alarm".
	rowStart := strings.Index(body, `href="/t/reminder/`)
	if rowStart < 0 {
		t.Fatalf("could not find reminder row in page\n%s", truncate(body))
	}
	rowEnd := strings.Index(body[rowStart:], "</li>")
	if rowEnd < 0 {
		t.Fatalf("could not find end of reminder row\n%s", truncate(body))
	}
	row := body[rowStart : rowStart+rowEnd]

	if strings.Contains(row, ">Alarm</span>") || strings.Contains(row, ">Timer</span>") {
		t.Errorf("reminder list row must not show kind badge 'Alarm'/'Timer'; found in:\n%s", truncate(row))
	}

	// The field name "kind" must not appear anywhere in the row — not even as a
	// visually-hidden span that screen readers announce.
	if strings.Contains(row, `sw-visually-hidden"> kind</span>`) ||
		strings.Contains(row, `sw-visually-hidden">kind</span>`) {
		t.Errorf("reminder list row must not expose the field name 'kind' even visually hidden; found in:\n%s", truncate(row))
	}

	// The word "Alarm kind" (the leaked pattern) must not appear.
	if strings.Contains(strings.ToLower(row), "alarm kind") ||
		strings.Contains(strings.ToLower(row), "timer kind") {
		t.Errorf("reminder list row must not contain raw 'kind' label; found in:\n%s", truncate(row))
	}

	// The reminder title and date should still be visible.
	if !strings.Contains(row, "Water the beans") {
		t.Errorf("reminder list row should still show its title; found in:\n%s", truncate(row))
	}
}

// TestReminderDetailLedeDoesNotShowKindBadge verifies that on a reminder's own
// detail page the lede does not start with or contain a badge for the kind enum.
// The lede must read naturally without a raw field label prefix (acceptance items 2 & 3).
func TestReminderDetailLedeDoesNotShowKindBadge(t *testing.T) {
	_, h := newApp(t)

	var result struct{ ID string }
	decode(t, postJSON(t, h, http.MethodPost, "/api/reminder", map[string]any{
		"title": "Call the vet",
		"at":    "2026-10-05T09:00:00Z",
	}), &result)

	rec := get(t, h, "/t/reminder/"+result.ID)
	body := rec.Body.String()

	// Extract just the lede div.
	ledeStart := strings.Index(body, `<div class="sw-lede">`)
	if ledeStart < 0 {
		t.Fatalf("reminder detail has no lede\n%s", truncate(body))
	}
	ledeEnd := strings.Index(body[ledeStart:], "</div>")
	if ledeEnd < 0 {
		t.Fatalf("could not find end of lede\n%s", truncate(body))
	}
	lede := body[ledeStart : ledeStart+ledeEnd]

	// The lede must NOT contain a badge with the kind value.
	if strings.Contains(lede, ">Alarm</span>") || strings.Contains(lede, ">Timer</span>") {
		t.Errorf("reminder detail lede must not show kind badge; found in:\n%s", truncate(lede))
	}

	// The field name "kind" must not appear as a visually-hidden span.
	if strings.Contains(lede, `sw-visually-hidden"> kind</span>`) ||
		strings.Contains(lede, `sw-visually-hidden">kind</span>`) {
		t.Errorf("reminder detail lede must not expose 'kind' as a field name; found in:\n%s", truncate(lede))
	}

	// The date should still be present (as a chip or span).
	if !strings.Contains(lede, "sw-badge") && !strings.Contains(lede, "sw-when") {
		t.Errorf("reminder detail lede should still show the datetime as a badge or span; found in:\n%s", truncate(lede))
	}

	// The creation timestamp via whenMade should still be present.
	if !strings.Contains(body, `class="sw-detail__when sw-muted sw-small"`) {
		t.Errorf("reminder detail page lede should still show creation time")
	}
}

// TestReminderDetailLedeNoRawKindWord verifies that the word "kind" does not
// appear as a standalone visible label in the reminder detail lede at all,
// matching acceptance item 3.
func TestReminderDetailLedeNoRawKindWord(t *testing.T) {
	_, h := newApp(t)

	var result struct{ ID string }
	decode(t, postJSON(t, h, http.MethodPost, "/api/reminder", map[string]any{
		"title": "Stretch",
		"at":    "2026-10-05T17:00:00Z",
	}), &result)

	body := get(t, h, "/t/reminder/"+result.ID).Body.String()

	ledeStart := strings.Index(body, `<div class="sw-lede">`)
	if ledeStart < 0 {
		t.Fatalf("no lede found\n%s", truncate(body))
	}
	ledeEnd := strings.Index(body[ledeStart:], "</div>")
	if ledeEnd < 0 {
		t.Fatalf("no closing tag for lede\n%s", truncate(body))
	}
	lede := body[ledeStart : ledeStart+ledeEnd]

	// "kind" must not appear anywhere in the lede.
	if strings.Contains(strings.ToLower(lede), "kind") {
		t.Errorf("reminder detail lede must not contain the word 'kind' at all; found in:\n%s", truncate(lede))
	}
}
