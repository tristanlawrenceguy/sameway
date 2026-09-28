package server_test

import (
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
)

// noEmptyH3 fails when any h3 on the page says nothing.
func noEmptyH3(t *testing.T, body string) {
	t.Helper()
	for _, h := range h3Said(body) {
		if h == "" {
			t.Errorf("blank-summary activity must not render an empty <h3>\n%s", truncate(body))
		}
	}
}

// TestActivityPageBlankSummaryShowsFallback checks that when an activity
// record has a blank summary, the /activity page still reads it as a
// sentence built from actor + action + detail, rather than an empty h3.
func TestActivityPageBlankSummaryShowsFallback(t *testing.T) {
	a, h := newApp(t)

	// Seed one normal and one blank-summary activity so we can verify both.
	chat.Record(a.Store, "assistant", chat.Change{Action: "created", Component: "note", ID: "aaa1", Detail: "First note"})

	rec, err := a.Store.Create(chat.ActivityType, map[string]any{
		"summary": "",
		"actor":   "human",
		"action":  "said",
		"detail":  "hello",
	})
	if err != nil {
		t.Fatal(err)
	}

	body := get(t, h, "/activity").Body.String()

	// The blank-summary record should not produce an empty h3.
	noEmptyH3(t, body)

	// It reads "You said: hello" (human + action + detail; with a detail but
	// no target, the action takes a colon).
	if !anyH3Says(body, "You said: hello") {
		t.Errorf("expected fallback 'You said: hello' for blank-summary human activity\n%s", truncate(body))
	}

	// Verify the record was inserted with a stable id so we can check it.
	if rec.ID == "" {
		t.Fatal("created record has empty ID")
	}

	// The normal summary entry should still appear correctly.
	if !anyH3Says(body, "Assistant created note First note") {
		t.Errorf("normal activity summary 'Assistant created note First note' missing\n%s", truncate(body))
	}

	// Total h3 count: one per activity = 2.
	h3Count := strings.Count(body, eventHeading)
	if h3Count != 2 {
		t.Errorf("expected 2 <h3> elements for 2 activities, got %d\n%s", h3Count, truncate(body))
	}
}

// TestActivityPageBlankSummaryAssistantFallback checks that assistant and
// system actors also get a fallback when summary is blank.
func TestActivityPageBlankSummaryAssistantFallback(t *testing.T) {
	a, h := newApp(t)

	// Seed an assistant activity with empty summary and no detail.
	_, err := a.Store.Create(chat.ActivityType, map[string]any{
		"summary": "",
		"actor":   "assistant",
		"action":  "added",
		"target":  "card",
	})
	if err != nil {
		t.Fatal(err)
	}

	body := get(t, h, "/activity").Body.String()

	// No empty h3.
	noEmptyH3(t, body)

	// Fallback for assistant without detail: "Assistant added card".
	if !anyH3Says(body, "Assistant added card") {
		t.Errorf("expected fallback 'Assistant added card' for blank-summary assistant activity\n%s", truncate(body))
	}
}

// TestActivityPageBlankSummarySystemFallback checks that system actors get a
// fallback when summary is blank.
func TestActivityPageBlankSummarySystemFallback(t *testing.T) {
	a, h := newApp(t)

	_, err := a.Store.Create(chat.ActivityType, map[string]any{
		"summary": "",
		"actor":   "system",
		"action":  "failed",
		"detail":  "no model",
	})
	if err != nil {
		t.Fatal(err)
	}

	body := get(t, h, "/activity").Body.String()

	noEmptyH3(t, body)

	// Fallback for system: "System failed: no model".
	if !anyH3Says(body, "System failed: no model") {
		t.Errorf("expected fallback 'System failed: no model'\n%s", truncate(body))
	}
}
