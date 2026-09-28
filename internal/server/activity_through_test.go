package server_test

import (
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
)

// ActivityPageDoesNotShowThroughTheAPI checks that old activity entries with
// via="through the API" no longer render ", through the API" in their heading
// or body on /activity (acceptance 1–2). The event renderer should skip the
// via span entirely for such values.
func TestActivityPageDoesNotShowThroughTheAPI(t *testing.T) {
	a, h := newApp(t)

	// Seed an old-style activity record with via="through the API" and no
	// headingSummary — this is what older records look like.
	_, err := a.Store.Create(chat.ActivityType, map[string]any{
		"summary":   "You deleted note Water the plants, through the API",
		"actor":     "human",
		"action":    "deleted",
		"target":    "note",
		"detail":    "Water the plants",
		"via":       "through the API",
		"target_id": "old1",
	})
	if err != nil {
		t.Fatal(err)
	}

	body := get(t, h, "/activity").Body.String()

	if strings.Contains(body, ", through the API") {
		t.Errorf("activity page must not contain ', through the API' in rendered event\n%s", truncate(body))
	}
	if strings.Contains(said(body), "through the API") {
		t.Errorf("activity page text must not say 'through the API'\n%q", said(body))
	}

	// The entry should still be readable as a plain sentence.
	if !anyH3Says(body, "You deleted note Water the plants") {
		t.Errorf("expected heading 'You deleted note Water the plants' in activity page\n%s", truncate(body))
	}
}

// ActivityPageDoesNotShowThroughCLI checks that via="through the command line"
// is also stripped from rendered event text.
func TestActivityPageDoesNotShowThroughCLI(t *testing.T) {
	a, h := newApp(t)

	_, err := a.Store.Create(chat.ActivityType, map[string]any{
		"summary":   "You deleted note Seed, through the command line",
		"actor":     "human",
		"action":    "deleted",
		"target":    "note",
		"detail":    "Seed",
		"via":       "through the command line",
		"target_id": "old2",
	})
	if err != nil {
		t.Fatal(err)
	}

	body := get(t, h, "/activity").Body.String()

	if strings.Contains(body, ", through the command line") {
		t.Errorf("activity page must not contain ', through the command line' in rendered event\n%s", truncate(body))
	}
	if strings.Contains(said(body), "through the command line") {
		t.Errorf("activity page text must not say 'through the command line'\n%q", said(body))
	}

	if !anyH3Says(body, "You deleted note Seed") {
		t.Errorf("expected heading 'You deleted note Seed' in activity page\n%s", truncate(body))
	}
}

// ActivityPageStillShowsNonThroughVia checks that a non-"through" via value
// (like "pixel-7") still renders as ", on pixel-7".
func TestActivityPageStillShowsNonThroughVia(t *testing.T) {
	a, h := newApp(t)

	_, err := a.Store.Create(chat.ActivityType, map[string]any{
		"summary":   "You deleted note Water the plants",
		"actor":     "human",
		"action":    "deleted",
		"target":    "note",
		"detail":    "Water the plants",
		"via":       "pixel-7",
		"target_id": "old3",
	})
	if err != nil {
		t.Fatal(err)
	}

	body := get(t, h, "/activity").Body.String()

	if !strings.Contains(body, ", on pixel-7") {
		t.Errorf("non-through via should still render as ', on <device>'\n%s", truncate(body))
	}
}
