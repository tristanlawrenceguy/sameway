package server_test

import (
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
)

// TestActivityFilterHidesGoHttpClient checks that "Go-http-client" is not
// listed as a who-filter option — it appears as "An agent" instead. This
// covers acceptance item 1: the /activity who-filter dropdown no longer
// lists "Go-http-client" as an option.

func TestActivityFilterHidesGoHttpClient(t *testing.T) {
	a, h := newApp(t)
	seedLog(t, a, h)
	// Create an activity entry with Go-http-client as the agent name.
	chat.Record(a.Store, chat.ActorAgent, chat.Change{
		Action:    "added",
		Component: "note",
		ID:        "n-gohc",
		Detail:    "Go-Client note",
		By:        "Go-http-client",
		Via:       chat.ThroughAPI,
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
