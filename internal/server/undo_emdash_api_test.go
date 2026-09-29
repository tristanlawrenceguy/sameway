package server_test

import (
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
)

// GET /api/activity returns cleaned summary fields for undo entries — no raw
// schema descriptions in the summary/title fields. An activity record with a
// stored summary like "You undid: Assistant changed Pace — how changes arrive
// to Calmly" should return the cleaned version "You undid: Assistant changed
// pace to Calmly". This covers acceptance item 2 for the API path.
func TestAPIActivityUndoEmDashSummaryIsCleaned(t *testing.T) {
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

	var resp struct {
		Records []struct{ Fields map[string]any }
	}
	decode(t, get(t, h, "/api/activity"), &resp)

	for _, r := range resp.Records {
		s, _ := r.Fields["summary"].(string)
		if strings.Contains(s, `—`) || strings.Contains(s, "how changes arrive") {
			t.Errorf("API summary for undo entry must not contain em-dash descriptions\nrecord: %v\ngot summary: %q", r.Fields, s)
		}
		if undoes, _ := r.Fields["undoes"].(string); undoes != "" && !strings.Contains(s, "pace to Calmly") {
			t.Errorf("API summary for undo entry should be cleaned\nrecord: %v\ngot summary: %q", r.Fields, s)
		}
	}
}
