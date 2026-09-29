package server_test

import (
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
)

// Undo summary headings on /activity show clean property names without em-dash
// descriptions. An activity record with an undo summary like "You undid:
// Assistant changed Pace — how changes arrive to Calmly" should render as
// "Assistant changed pace to Calmly". This covers acceptance item 1 for the
// HTML path (activity page).
func TestActivityUndoEmDashSummaryIsCleaned(t *testing.T) {
	a, h := newApp(t)

	// Insert an undo activity record with an em-dash description directly into
	// the store to simulate older stored data.
	_, err := a.Store.Create(chat.ActivityType, map[string]any{
		"actor":   "assistant",
		"action":  "removed",
		"target":  "setting",
		"undoes":  "fake-id-1",
		"summary": "You undid: Assistant changed Pace — how changes arrive to Calmly",
	})
	if err != nil {
		t.Fatal(err)
	}

	body := get(t, h, "/activity").Body.String()

	if strings.Contains(body, `—`) || strings.Contains(body, "how changes arrive") {
		t.Errorf("activity page must not contain em-dash descriptions in undo headings\n\nwant: no '— how changes arrive'\n\ngot body:\n%s", truncate(body))
	}

	if !anyH3Says(body, "Assistant changed pace to Calmly") &&
		!strings.Contains(said(body), "changed pace to Calmly") {
		t.Errorf("activity page heading should show clean property name\n\nwant: 'changed pace to Calmly'\n\ngot body:\n%s", truncate(body))
	}
}
