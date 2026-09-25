package server_test

import (
	"net/url"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// TestLongRecordTitleIsWordTrimmedInActivity checks that when a record with
// a long multi-word title is created, the activity log entry uses 6-word
// word-count trimming (matching trimTitle for detail page h1s), not
// character-boundary truncation at ~75 characters. This covers acceptance
// items 1 and 3: the same trimmed version must appear in both the detail
// page heading and the activity log entry's h3 on /activity.
func TestLongRecordTitleIsWordTrimmedInActivity(t *testing.T) {
	a, h := newApp(t)

	// The long title from the backlog reproduction. trimTitle produces:
	// "This is a note with a…" (6 words + ellipsis).
	const longTitle = "This is a note with a very long multi-word title that exceeds six words and should be trimmed"
	wantTrimmed := "This is a note with a…"

	// Create the record via the model loop so createRecord runs, which
	// calls recordTitle (currently truncate(v, 60), after fix trimWords(v, 6)).
	a.Chat.Provider = &scripted{steps: []*llm.Response{
		toolCall("create_record", map[string]any{
			"type":   "note",
			"fields": map[string]any{"title": longTitle, "body": "test"},
		}),
	}}

	postForm(t, h, "/chat", url.Values{
		"message": {"make a note called " + longTitle},
		"from":    {"/"},
	})

	body := get(t, h, "/activity").Body.String()

	if !strings.Contains(body, wantTrimmed) {
		t.Errorf("activity page should show the 6-word trimmed title %q\n\nwant in h3 heading: Assistant created note %s\n\ngot body:\n%s", wantTrimmed, wantTrimmed, truncate(body))
	}

	// The activity detail field must not be character-boundary truncated.
	// A ~75-char truncation of the long title would look like:
	// "This is a note with a very long multi-word title that excee…" (63 chars).
	// The 6-word trimmed version is only 24 runes.
	activity, _ := a.Store.List(chat.ActivityType, store.ListOptions{OrderBy: "created_at", Desc: true, Limit: 1})
	if len(activity) == 0 {
		t.Fatal("no activity entries found")
	}

	detailField, _ := activity[0].Fields["detail"].(string)
	trimmedLen := len([]rune(wantTrimmed))
	if len([]rune(detailField)) != trimmedLen {
		t.Errorf("activity detail field is %d runes (%q), want 6-word trimmed %q (%d runes)\n\nThis means recordTitle() still uses character-boundary truncation instead of word-count trimming.",
			len([]rune(detailField)), detailField, wantTrimmed, trimmedLen)
	}

	// Verify the summary on the activity page also contains the trimmed title.
	summary := "Assistant created note " + wantTrimmed
	if !strings.Contains(body, summary) {
		t.Errorf("activity page h3 should contain the full summary with 6-word trimmed title\n\nwant: %s\n\ngot body:\n%s", summary, truncate(body))
	}

	// The detail field must end with an ellipsis when trimmed.
	if !strings.HasSuffix(detailField, "…") {
		t.Errorf("trimmed detail should end with ellipsis (U+2026), got: %q\n\nThis means the title was not word-count trimmed.", detailField)
	}
}

// TestLongRecordTitleTrimmedInUndoButtonAccessibleName checks that undo button
// accessible names on the activity page use the same 6-word trimmed title as
// other surfaces, covering acceptance item 2. The event component template puts
// {{.detail}} inside a sw-visually-hidden span within the Undo button's label.
func TestLongRecordTitleTrimmedInUndoButtonAccessibleName(t *testing.T) {
	a, h := newApp(t)

	const longTitle = "This is a note with a very long multi-word title that exceeds six words and should be trimmed"
	wantTrimmed := "This is a note with a…"

	// Create the record through the model loop.
	a.Chat.Provider = &scripted{steps: []*llm.Response{
		toolCall("create_record", map[string]any{
			"type":   "note",
			"fields": map[string]any{"title": longTitle, "body": "test"},
		}),
	}}

	postForm(t, h, "/chat", url.Values{
		"message": {"make a note called " + longTitle},
		"from":    {"/"},
	})

	body := get(t, h, "/activity").Body.String()

	// The undo button has an accessible name built from:
	// "Undo<span class="sw-visually-hidden"> {{.action}}{{if .target}} {{.target}}{{end}}{{if .detail}} {{.detail}}{{end}}</span>"
	// So the body should contain "> created note This is a note with a…" after Undo
	// inside sw-visually-hidden, and NOT the character-boundary truncated version.
	if !strings.Contains(body, "created note "+wantTrimmed) {
		t.Errorf("undo button accessible name should use 6-word trimmed title\n\nwant 'created note %s' in body\n\ngot body:\n%s", wantTrimmed, truncate(body))
	}

	// Make sure it's NOT the character-boundary truncated version.
	charTruncated := "This is a note with a very long multi-word title that excee…"
	if strings.Contains(body, charTruncated) {
		t.Errorf("undo button should not show character-boundary truncated title\n\nFound: %q\n\nActivity log entries must use 6-word word-count trimming like trimTitle.", charTruncated)
	}
}

// TestShortRecordTitleIsNotTrimmedInActivity checks that records with six or
// fewer words in the title are not trimmed, covering that the fix only affects
// titles exceeding six words. Acceptance item 4: trimTitle itself is unchanged,
// so short titles pass through intact.
func TestShortRecordTitleIsNotTrimmedInActivity(t *testing.T) {
	a, h := newApp(t)

	const shortTitle = "Call the dentist"

	// Create via model loop.
	a.Chat.Provider = &scripted{steps: []*llm.Response{
		toolCall("create_record", map[string]any{
			"type":   "note",
			"fields": map[string]any{"title": shortTitle, "body": "test"},
		}),
	}}

	postForm(t, h, "/chat", url.Values{
		"message": {"make a note called " + shortTitle},
		"from":    {"/"},
	})

	body := get(t, h, "/activity").Body.String()

	if !strings.Contains(body, shortTitle) {
		t.Errorf("short title should appear untrimmed in activity\n\nwant: %s\n\ngot body:\n%s", shortTitle, truncate(body))
	}
}
