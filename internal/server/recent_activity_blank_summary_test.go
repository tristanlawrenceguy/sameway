package server_test

import (
	"slices"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
)

// TestRecentActivityBlankSummaryShowsFallback checks that the recent activity
// section on any page still reads a blank-summary record as a sentence built
// from actor + action + detail, rather than rendering an empty h3.
func TestRecentActivityBlankSummaryShowsFallback(t *testing.T) {
	a, h := newApp(t)

	// Seed a human activity with an empty summary directly into the store.
	_, err := a.Store.Create(chat.ActivityType, map[string]any{
		"summary": "",
		"actor":   "human",
		"action":  "said",
		"detail":  "hello",
	})
	if err != nil {
		t.Fatal(err)
	}

	body := get(t, h, "/chat").Body.String()

	// No empty h3 in recent activity.
	if slices.Contains(h3Said(body), "") {
		t.Errorf("blank-summary activity must not render an empty <h3>\n%s", truncate(body))
	}

	// It reads "You said: hello" (with a detail but no target, the action
	// takes a colon).
	if !anyH3Says(body, "You said: hello") {
		t.Errorf("expected fallback 'You said: hello' for blank-summary human activity\n%s", truncate(body))
	}
}

// TestRecentActivityBlankSummaryAssistantFallback checks that assistant actor
// also gets a fallback in the recent activity section.
func TestRecentActivityBlankSummaryAssistantFallback(t *testing.T) {
	a, h := newApp(t)

	_, err := a.Store.Create(chat.ActivityType, map[string]any{
		"summary": "",
		"actor":   "assistant",
		"action":  "added",
		"target":  "card",
	})
	if err != nil {
		t.Fatal(err)
	}

	body := get(t, h, "/chat").Body.String()

	if slices.Contains(h3Said(body), "") {
		t.Error("blank-summary activity must not render an empty <h3>")
	}

	if !anyH3Says(body, "Assistant added card") {
		t.Errorf("expected fallback 'Assistant added card' for blank-summary assistant activity\n%s", truncate(body))
	}
}
