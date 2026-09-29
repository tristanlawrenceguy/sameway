package server_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
)

// TestActivityDetailHeadingUsesCleanSummary checks that the H1 heading on a
// detail page for an undo activity entry shows clean property names without
// em-dash descriptions. "Assistant changed Pace — how changes arrive to Calmly"
// becomes "Assistant changed pace to Calmly". This covers acceptance items 1 and 3.
func TestActivityDetailHeadingUsesCleanSummary(t *testing.T) {
	a, h := newApp(t)

	// Seed an undo entry with a raw em-dash description in the summary field
	// (the kind that titleOf returns today).
	rec, err := a.Store.Create(chat.ActivityType, map[string]any{
		"summary": "You undid: Assistant changed Pace — how changes arrive to Calmly",
		"actor":   "human",
		"action":  "set",
		"target":  "ui.pace",
		"detail":  "calm",
		"undoes":  "aaa1",
	})
	if err != nil {
		t.Fatal(err)
	}

	body := get(t, h, "/t/activity/"+rec.ID).Body.String()

	// Acceptance 1: the H1 must not contain raw em-dash schema descriptions.
	if strings.Contains(body, "how changes arrive") {
		t.Errorf("H1 on activity detail page must not show raw description 'how changes arrive'\n%s", truncate(body))
	}

	// Acceptance 3: non-undo entries with humanized text should also be clean.
	cleanRec, err := a.Store.Create(chat.ActivityType, map[string]any{
		"summary": "Assistant changed Pace to Calmly",
		"actor":   "assistant",
		"action":  "set",
		"target":  "ui.pace",
		"detail":  "calm",
	})
	if err != nil {
		t.Fatal(err)
	}

	cleanBody := get(t, h, "/t/activity/"+cleanRec.ID).Body.String()
	if strings.Contains(cleanBody, "Pace") && !strings.Contains(cleanBody, "pace") {
		// The heading should use the cleaned form "Assistant changed pace to Calmly"
		t.Errorf("non-undo activity heading must show 'pace' not 'Pace'\n%s", truncate(cleanBody))
	}
}

// TestAPIActivityTitleUsesCleanSummary checks that GET /api/activity/<id>
// returns a cleaned summary field matching what the HTML detail page renders.
// Raw em-dash descriptions like "how changes arrive" should not appear in the
// API title, and must match the heading on the HTML page for the same record.
func TestAPIActivityTitleUsesCleanSummary(t *testing.T) {
	a, h := newApp(t)

	// Seed an undo entry with a raw em-dash description.
	rec, err := a.Store.Create(chat.ActivityType, map[string]any{
		"summary": "You undid: Assistant changed Pace — how changes arrive to Calmly",
		"actor":   "human",
		"action":  "set",
		"target":  "ui.pace",
		"detail":  "calm",
		"undoes":  "aaa1",
	})
	if err != nil {
		t.Fatal(err)
	}

	var one struct {
		Title  string         `json:"title"`
		Fields map[string]any `json:"fields"`
	}
	json.Unmarshal(get(t, h, "/api/activity/"+rec.ID).Body.Bytes(), &one)

	// Acceptance 2: the API title must not contain raw em-dash descriptions.
	if strings.Contains(one.Title, "how changes arrive") {
		t.Errorf("API activity title must not show raw description 'how changes arrive'; got %q", one.Title)
	}

	// The API title should match what the HTML detail page heading says.
	htmlBody := get(t, h, "/t/activity/"+rec.ID).Body.String()
	if !strings.Contains(htmlBody, one.Title) {
		t.Errorf("API title %q must match the H1 on the HTML detail page\n%s", one.Title, truncate(htmlBody))
	}

	// Acceptance 3: non-undo entries also get cleaned titles.
	cleanRec, err := a.Store.Create(chat.ActivityType, map[string]any{
		"summary": "Assistant changed Pace to Calmly",
		"actor":   "assistant",
		"action":  "set",
		"target":  "ui.pace",
		"detail":  "calm",
	})
	if err != nil {
		t.Fatal(err)
	}

	var cleanOne struct {
		Title string `json:"title"`
	}
	json.Unmarshal(get(t, h, "/api/activity/"+cleanRec.ID).Body.Bytes(), &cleanOne)

	// Should be cleaned to lowercase "pace" not capitalized "Pace".
	if strings.Contains(cleanOne.Title, "Pace") && !strings.Contains(cleanOne.Title, "pace") {
		t.Errorf("API title for non-undo activity must use 'pace' not 'Pace'; got %q", cleanOne.Title)
	}
}
