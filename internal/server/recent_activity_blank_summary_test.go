package server_test

import (
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
)

// TestRecentActivityBlankSummaryShowsFallback checks that the recent activity
// section on any page falls back to actor+action+detail text when summary is
// blank, rather than rendering an empty h3.
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
	if strings.Contains(body, `<h3 class="sw-event__heading"></h3>`) {
		t.Errorf("blank-summary activity must not render <h3 class=\"sw-event__heading\"></h3>\n%s", truncate(body))
	}

	// Fallback text should appear: "You said hello".
	if !strings.Contains(body, "You said hello") {
		t.Errorf("expected fallback 'You said hello' for blank-summary human activity\n%s", truncate(body))
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

	if strings.Contains(body, `<h3 class="sw-event__heading"></h3>`) {
		t.Error("blank-summary activity must not render an empty <h3>")
	}

	if !strings.Contains(body, "Assistant added card") {
		t.Errorf("expected fallback 'Assistant added card' for blank-summary assistant activity\n%s", truncate(body))
	}
}
