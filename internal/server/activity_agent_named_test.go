package server_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// What an agent does is logged as from an agent: never under Go-http-client,
// the name its HTTP library sends, in the filters or on an entry's page.

// TestActivityFilterHidesGoHttpClient checks that "Go-http-client" is not
// listed as a who-filter option — it appears as "An agent" instead. This
// covers acceptance item 1: the /activity who-filter dropdown no longer
// lists "Go-http-client" as an option.

func TestActivityFilterHidesGoHttpClient(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)
	seedLog(t, a, h)
	// Create an activity entry with Go-http-client as the agent name.
	records.Record(a.Store, records.ActorAgent, records.Change{
		Action:    "added",
		Component: "note",
		ID:        "n-gohc",
		Detail:    "Go-Client note",
		By:        "Go-http-client",
		Via:       records.ThroughAPI,
	})

	body := get(t, h, "/activity").Body.String()

	// Acceptance 1: Go-http-client must NOT appear as a filter option.
	if strings.Contains(body, `>Go-http-client</option>`) {
		t.Errorf("the who-filter dropdown should not list 'Go-http-client' as an option\n%s", truncate(said(body)))
	}

	// The entry itself (as "An agent") SHOULD appear in the dropdown.
	if !strings.Contains(body, ">An agent</option>") && !strings.Contains(body, `value="a-`) {
		t.Errorf("the who-filter should show 'An agent' for Go-http-client entries\n%s", truncate(said(body)))
	}

	// The log line itself must not say "Go-http-client".
	if strings.Contains(said(body), "Go-http-client") {
		t.Errorf("activity page must not display 'Go-http-client'\n%s", said(body))
	}
}

// TestGoHttpClientDetailLedeSaysAnAgent checks that an activity detail page
// shows "From: An agent" (or similar human-readable label) rather than the
// raw machine-language identifier. This covers acceptance item 2 — activity
// detail pages show "From: an agent" for API-attributed entries.

func TestGoHttpClientDetailLedeSaysAnAgent(t *testing.T) {
	t.Parallel()
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
