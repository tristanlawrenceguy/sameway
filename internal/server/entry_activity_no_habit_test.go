package server_test

import (
	"strings"
	"testing"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/server"
	"github.com/tristanlawrenceguy/sameway/internal/when"
)

// TestOldEntryActivityShowsReadableFallback verifies that an old-style activity
// entry where detail = raw entry ID resolves to a readable title via the
// recordTitle fallback. This covers acceptance item 3: no raw 16-char hex IDs
// in entry headings on /activity and /t/entry pages.
func TestOldEntryActivityShowsReadableFallback(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)

	// Seed a habit so the entry has something to name it after.
	habit, err := a.Store.Create(server.HabitType, map[string]any{
		"name": "Reading", "unit": "minutes", "cadence": "day",
	})
	if err != nil {
		t.Fatal(err)
	}

	// Seed an entry linked to that habit.
	at := when.Store(time.Now(), false)
	entry, err := a.Store.Create(server.EntryType, map[string]any{
		"habit": habit.ID,
		"at":    at,
	})
	if err != nil {
		t.Fatal(err)
	}

	// Seed an old-style activity entry where detail = raw entry ID.
	records.Record(a.Store, "assistant", records.Change{
		Action:    "created",
		Component: "entry",
		ID:        entry.ID,
		Detail:    entry.ID, // raw ID — simulating pre-fix storage
		Href:      "/t/entry/" + entry.ID,
	})

	body := get(t, h, "/activity").Body.String()

	// Acceptance 3: no raw database ID in visible text.
	if strings.Contains(said(body), entry.ID) {
		t.Errorf("the activity log should not show a raw database ID %q in visible text\n%s", entry.ID, said(body))
	}
	// The readable fallback title appears instead of the raw ID.
	if !strings.Contains(said(body), "Reading: 1") {
		t.Errorf("the activity log should show the readable fallback 'Reading: 1', got:\n%s", said(body))
	}

	// Acceptance 2: the heading uses readable text.
	if !anyH3Says(body, "Assistant created entry Reading: 1") {
		t.Errorf("the <h3> heading should say 'Assistant created entry Reading: 1', not a bare ID\n%s", truncate(body))
	}

	// Acceptance 2: the same on the /t/entry page.
	entryBody := get(t, h, "/t/entry/"+entry.ID).Body.String()
	if strings.Contains(said(entryBody), entry.ID) {
		t.Errorf("the entry detail page should not show a raw database ID %q in visible text\n%s", entry.ID, said(entryBody))
	}
}
