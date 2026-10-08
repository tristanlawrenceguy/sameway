package server_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// TestGoHttpClientDetailLedeSaysAnAgent checks that an activity detail page
// shows "From: An agent" (or similar human-readable label) rather than the
// raw machine-language identifier. This covers acceptance item 2 — activity
// detail pages show "From: an agent" for API-attributed entries.

func TestGoHttpClientDetailLedeSaysAnAgent(t *testing.T) {
	a, h := newApp(t)
	seedLog(t, a, h)

	// Create a note via the Go-http-client API call.
	apiAs(t, h, http.MethodPost, "/api/note", `{"title":"Go-Client Detail Note"}`, map[string]string{
		"User-Agent": "Go-http-client/1.1",
	})

	body := get(t, h, "/activity").Body.String()

	// The summary on the activity list must not say "Go-http-client".
	if strings.Contains(said(body), "Go-http-client") {
		t.Errorf("activity list must not display 'Go-http-client': %s", said(body))
	}

	// Find the newest entry to check its detail page.
	entries, _ := a.Store.List(records.ActivityType, store.ListOptions{OrderBy: "created_at", Desc: true, Limit: 1})
	if len(entries) == 0 {
		t.Fatal("no activity entries found")
	}
	entryID := entries[0].ID

	detail := get(t, h, "/t/activity/"+entryID).Body.String()

	// Acceptance 2: detail page must not say "Go-http-client".
	if strings.Contains(said(detail), "Go-http-client") {
		t.Errorf("activity detail page must not show 'Go-http-client': %s", said(detail))
	}

	// It should say something human-readable like "An agent" or "an agent".
	if !strings.Contains(strings.ToLower(said(detail)), "agent") {
		t.Errorf("activity detail page should mention 'agent' for an agent attribution: %s", said(detail))
	}
}
