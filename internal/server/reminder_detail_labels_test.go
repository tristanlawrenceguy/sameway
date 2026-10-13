package server_test

// Tests for raw field labels on the reminder detail page. The schema file
// examples/workspaces/starter/schema/reminder.yaml defines the kind field with
// label: Type, so the detail page shows "Type" as a <dt> value (backlog 0695).

import (
	"net/http"
	"strings"
	"testing"
)

// TestReminderDetailDoesNotShowRawKindLabel asserts that the reminder detail
// page does not display "Kind" as a visible field label when the kind field
// has a value. The schema should provide a plain-language label such as "Type".
// (Acceptance 1.)
func TestReminderDetailDoesNotShowRawKindLabel(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)

	rec, err := a.Store.Create("reminder", map[string]any{
		"title": "Morning alarm",
		"at":    "2026-10-01T07:30:00Z",
		"kind":  "alarm",
	})
	if err != nil {
		t.Fatal(err)
	}

	page := get(t, h, "/t/reminder/"+rec.ID+fieldsView).Body.String()

	if strings.Contains(page, "<dt>Kind</dt>") {
		t.Error("reminder detail must not show \"Kind\" as a field label; use plain language like \"Type\"")
	}
}

// TestReminderDetailShowsTypeLabel asserts that the reminder detail page renders
// "Type" (or another human-readable equivalent) instead of the raw schema name
// "Kind" for the kind enum field. The value itself ("alarm" or "timer") must
// still appear in its <dd>. (Acceptance 1–2.)
func TestReminderDetailShowsTypeLabel(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)

	rec, err := a.Store.Create("reminder", map[string]any{
		"title": "Morning alarm",
		"at":    "2026-10-01T07:30:00Z",
		"kind":  "alarm",
	})
	if err != nil {
		t.Fatal(err)
	}

	page := get(t, h, "/t/reminder/"+rec.ID+fieldsView).Body.String()

	if !strings.Contains(page, "<dt>Type</dt>") {
		t.Errorf("reminder detail should show \"<dt>Type</dt>\" for the kind field; page body:\n%s", truncate(page))
	}

	// The value must still appear alongside the label.
	if !strings.Contains(page, "Alarm") {
		t.Errorf("reminder detail should show the kind value \"Alarm\" next to its label; page body:\n%s", truncate(page))
	}
}

// TestReminderDetailTimerKindShowsTypeLabel asserts that a reminder with kind
// set to "timer" also shows "Type" as the field label (not "Kind"), and that
// the value "Timer" appears alongside it. (Acceptance 1–2.)
func TestReminderDetailTimerKindShowsTypeLabel(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)

	rec, err := a.Store.Create("reminder", map[string]any{
		"title": "Cooking timer",
		"at":    "2026-10-01T08:00:00Z",
		"kind":  "timer",
	})
	if err != nil {
		t.Fatal(err)
	}

	page := get(t, h, "/t/reminder/"+rec.ID+fieldsView).Body.String()

	if strings.Contains(page, "<dt>Kind</dt>") {
		t.Error("reminder detail must not show \"<dt>Kind</dt>\"; use plain language like \"Type\"")
	}

	if !strings.Contains(page, "<dt>Type</dt>") {
		t.Errorf("reminder detail should show \"<dt>Type</dt>\" for the kind field; page body:\n%s", truncate(page))
	}

	if !strings.Contains(page, "Timer") {
		t.Errorf("reminder detail should show the kind value \"Timer\" next to its label; page body:\n%s", truncate(page))
	}
}

// TestReminderDetailKindLabelCombination verifies that both negative and positive
// assertions hold simultaneously: no raw "Kind" label appears and the correct
// "Type" label does. This is the comprehensive acceptance test for backlog 0695.
func TestReminderDetailKindLabelCombination(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)

	rec, err := a.Store.Create("reminder", map[string]any{
		"title": "Combined label test",
		"at":    "2026-10-01T07:30:00Z",
		"kind":  "alarm",
	})
	if err != nil {
		t.Fatal(err)
	}

	page := get(t, h, "/t/reminder/"+rec.ID+fieldsView).Body.String()

	// Negative check: raw label must not appear.
	if strings.Contains(page, "<dt>Kind</dt>") {
		t.Error("raw label \"<dt>Kind</dt>\" found in reminder detail page; use a human-readable label like \"Type\"")
	}

	// Positive checks: correct label and value must appear.
	if !strings.Contains(page, "<dt>Type</dt>") {
		t.Errorf("reminder detail should show \"<dt>Type</dt>\" for the kind field; found in:\n%s", truncate(page))
	}

	if !strings.Contains(page, "Alarm") {
		t.Errorf("reminder detail should show the kind value \"Alarm\" next to its label; found in:\n%s", truncate(page))
	}
}

// TestReminderDetailAcceptsKindViaAPI verifies that creating a reminder with
// the kind field via the API succeeds and that the correct human-readable label
// appears on the detail page. (Acceptance 1–2.)
func TestReminderDetailAcceptsKindViaAPI(t *testing.T) {
	t.Parallel()
	_, h := newApp(t)

	var rem struct{ ID string }
	decode(t, postJSON(t, h, http.MethodPost, "/api/reminder", map[string]any{
		"title": "Evening alarm",
		"at":    "2026-10-01T19:00:00Z",
		"kind":  "alarm",
	}), &rem)

	rec := get(t, h, "/t/reminder/"+rem.ID+fieldsView)

	body := rec.Body.String()

	if strings.Contains(body, "<dt>Kind</dt>") {
		t.Error("reminder detail must not show \"<dt>Kind</dt>\"")
	}

	if !strings.Contains(body, "<dt>Type</dt>") {
		t.Errorf("reminder detail should show \"<dt>Type</dt>\"; page body:\n%s", truncate(body))
	}

	if !strings.Contains(body, "Alarm") {
		t.Errorf("reminder detail should show the kind value \"Alarm\"; page body:\n%s", truncate(body))
	}
}

// TestReminderDetailDumpDtLabels is a debug-only test that prints all <dt> elements
// from a reminder detail page to verify what labels are actually rendered.
func TestReminderDetailDumpDtLabels(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)

	rec, err := a.Store.Create("reminder", map[string]any{
		"title": "Debug reminder",
		"at":    "2026-10-01T07:30:00Z",
		"kind":  "alarm",
	})
	if err != nil {
		t.Fatal(err)
	}
	page := get(t, h, "/t/reminder/"+rec.ID+fieldsView).Body.String()
	start := 0
	for {
		idx := strings.Index(page[start:], "<dt>")
		if idx < 0 {
			break
		}
		end := strings.Index(page[start+idx:], "</dt>")
		t.Logf("dt: %s", page[start+idx:start+idx+end+5])
		start = start + idx + end + 5
	}
}
