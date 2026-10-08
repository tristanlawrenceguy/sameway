package server_test

import (
	"strings"
	"testing"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/server"
	"github.com/tristanlawrenceguy/sameway/internal/when"
)

// TestOldEntryActivityShowsReadableTitle seeds an activity entry where the
// detail field is a raw entry ID (simulating old storage before recordTitle
// was fixed for EntryType), verifies that /activity shows "Reading: 30 minutes"
// instead of the bare ID, and checks the link text is human-readable.
func TestOldEntryActivityShowsReadableTitle(t *testing.T) {
	a, h := newApp(t)

	// Seed a habit so we have something to name the entry after.
	habit, err := a.Store.Create(server.HabitType, map[string]any{
		"name": "Reading", "unit": "minutes", "cadence": "day",
	})
	if err != nil {
		t.Fatal(err)
	}

	// Seed an entry linked to that habit.
	at := when.Store(time.Now().Add(-time.Minute), false)
	entry, err := a.Store.Create(server.EntryType, map[string]any{
		"habit":  habit.ID,
		"at":     at,
		"amount": 30.0,
	})
	if err != nil {
		t.Fatal(err)
	}

	// Seed an old-style activity entry where detail = raw entry ID
	// (the way it was stored before the recordTitle fix for EntryType).
	records.Record(a.Store, "assistant", records.Change{
		Action:    "created",
		Component: "entry",
		ID:        entry.ID,
		Detail:    entry.ID, // raw ID, not a title — simulating old storage
		Href:      "/t/entry/" + entry.ID,
	})

	body := get(t, h, "/activity").Body.String()

	// Acceptance 1: the readable fallback title appears instead of the raw ID.
	if strings.Contains(said(body), entry.ID) {
		t.Errorf("the activity log should not show a raw database ID %q in visible text\n%s", entry.ID, said(body))
	}
	if !strings.Contains(said(body), "Reading: 30 minutes") {
		t.Errorf("the activity log should show the readable fallback title 'Reading: 30 minutes', got:\n%s", said(body))
	}

	// Acceptance 2: the link text is human-readable, not a raw ID.
	if !anyH3Says(body, "Assistant created entry Reading: 30 minutes") {
		t.Errorf("the <h3> heading should say 'Assistant created entry Reading: 30 minutes', not a bare ID\n%s", truncate(body))
	}

	// The link should lead to the entry's page.
	if !strings.Contains(body, `href="/t/entry/`+entry.ID) {
		t.Errorf("the heading link should lead to /t/entry/%s\n%s", entry.ID, truncate(body))
	}
}
