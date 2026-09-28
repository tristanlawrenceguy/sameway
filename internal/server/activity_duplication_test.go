package server_test

import (
	"net/url"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/llm"
)

// TestChatPageHasSingleActivityHeading checks that the /chat page renders
// exactly one <h2 class="sw-visually-hidden">Activity</h2> heading (not two).
func TestChatPageHasSingleActivityHeading(t *testing.T) {
	a, h := newApp(t)

	// Seed an activity so the disclosure appears.
	chat.Record(a.Store, "assistant", chat.Change{Action: "created", Component: "note", ID: "aaa1", Detail: "First note"})

	body := get(t, h, "/chat").Body.String()

	h2Count := strings.Count(body, `<h2 class="sw-visually-hidden">Activity</h2>`)
	if h2Count != 1 {
		t.Errorf("expected exactly 1 <h2 class=\"sw-visually-hidden\">Activity</h2> on /chat, got %d\n%s", h2Count, body)
	}

	// No visible (non-visually hidden) Activity heading should appear.
	if strings.Contains(body, `<h2>Activity</h2>`) {
		t.Errorf("no visible <h2>Activity</h2> should appear on /chat\n%s", body)
	}
}

// TestChatPageRecentActivityHasNoDuplicateEntries checks that each logged action
// appears exactly once in the Activity section — no duplicate text+timestamp pairs.
func TestChatPageRecentActivityHasNoDuplicateEntries(t *testing.T) {
	a, h := newApp(t)

	chat.Record(a.Store, "assistant", chat.Change{Action: "created", Component: "note", ID: "aaa1", Detail: "First note"})
	chat.Record(a.Store, "assistant", chat.Change{Action: "updated", Component: "note", ID: "aaa1", Detail: "Updated content"})

	body := get(t, h, "/chat").Body.String()

	for _, summary := range []string{"Assistant created note First note", "Assistant updated note Updated content"} {
		count := saidTimes(body, summary)
		if count != 1 {
			t.Errorf("activity entry %q should appear exactly once on /chat, got %d\n%s", summary, count, truncate(body))
		}
	}

	// Check for duplicate entries: no two h3 headings say the same thing.
	noDuplicateH3(t, body)
}

// noDuplicateH3 fails when two h3 headings on the page say the same thing.
func noDuplicateH3(t *testing.T, body string) {
	t.Helper()
	seen := map[string]bool{}
	for _, text := range h3Said(body) {
		if seen[text] {
			t.Errorf("duplicate activity entry found in recent activity list:\n%q\n%s", text, truncate(body))
		}
		seen[text] = true
	}
}

// TestChatPageRecentActivityNoDuplicateAfterModelAction checks that after the
// model makes a change via chat.Send, no duplicate entries appear in either
// the message receipt or the recent activity section.
func TestChatPageRecentActivityNoDuplicateAfterModelAction(t *testing.T) {
	a, h := newApp(t)

	a.Chat.Provider, a.Chat.ProviderErr = &scripted{steps: []*llm.Response{
		toolCall("add_component", map[string]any{"component": "card", "props": map[string]any{"title": "Shopping"}}),
		{Text: "Done."},
	}}, nil
	postForm(t, h, "/chat", url.Values{"message": {"add a card"}, "from": {"/"}})

	body := get(t, h, "/chat").Body.String()

	cardSummary := "Assistant added card Shopping"
	count := saidTimes(body, cardSummary)
	if count != 1 {
		t.Errorf("activity entry %q should appear exactly once on /chat after model action, got %d\n%s", cardSummary, count, truncate(body))
	}
}

// TestTaskDetailPageRecentActivityHasNoDuplicateEntries checks that task detail
// pages show each activity entry only once. (Acceptance 3)
func TestTaskDetailPageRecentActivityHasNoDuplicateEntries(t *testing.T) {
	a, h := newApp(t)

	taskRec, err := a.Store.Create("task", map[string]any{"title": "Test Task"})
	if err != nil {
		t.Fatal(err)
	}

	chat.Record(a.Store, "assistant", chat.Change{Action: "created", Component: "task", ID: taskRec.ID, Detail: "Test Task"})

	body := get(t, h, "/t/task/"+taskRec.ID).Body.String()

	h2Count := strings.Count(body, `<h2 class="sw-visually-hidden">Activity</h2>`)
	if h2Count != 1 {
		t.Errorf("expected exactly 1 <h2 class=\"sw-visually-hidden\">Activity</h2> on task detail page, got %d\n%s", h2Count, truncate(body))
	}

	for _, summary := range []string{"Assistant created task Test Task"} {
		count := saidTimes(body, summary)
		if count != 1 {
			t.Errorf("activity entry %q should appear exactly once on /t/task page, got %d\n%s", summary, count, truncate(body))
		}
	}
}

// TestNoteDetailPageRecentActivityHasNoDuplicateEntries checks that note detail
// pages show each activity entry only once. (Acceptance 4)
func TestNoteDetailPageRecentActivityHasNoDuplicateEntries(t *testing.T) {
	a, h := newApp(t)

	noteRec, err := a.Store.Create("note", map[string]any{"title": "Test Note"})
	if err != nil {
		t.Fatal(err)
	}

	chat.Record(a.Store, "assistant", chat.Change{Action: "created", Component: "note", ID: noteRec.ID, Detail: "Test Note"})

	body := get(t, h, "/t/note/"+noteRec.ID).Body.String()

	h2Count := strings.Count(body, `<h2 class="sw-visually-hidden">Activity</h2>`)
	if h2Count != 1 {
		t.Errorf("expected exactly 1 <h2 class=\"sw-visually-hidden\">Activity</h2> on note detail page, got %d\n%s", h2Count, truncate(body))
	}

	for _, summary := range []string{"Assistant created note Test Note"} {
		count := saidTimes(body, summary)
		if count != 1 {
			t.Errorf("activity entry %q should appear exactly once on /t/note page, got %d\n%s", summary, count, truncate(body))
		}
	}
}

// TestActivityPageNoDuplicateEntries checks that the full activity listing at
// /activity shows each action only once. (Acceptance 2)
func TestActivityPageNoDuplicateEntries(t *testing.T) {
	a, h := newApp(t)

	chat.Record(a.Store, "assistant", chat.Change{Action: "created", Component: "note", ID: "aaa1", Detail: "First note"})
	chat.Record(a.Store, "assistant", chat.Change{Action: "updated", Component: "note", ID: "aaa1", Detail: "Updated content"})

	body := get(t, h, "/activity").Body.String()

	for _, summary := range []string{"Assistant created note First note", "Assistant updated note Updated content"} {
		count := saidTimes(body, summary)
		if count != 1 {
			t.Errorf("activity entry %q should appear exactly once on /activity page, got %d\n%s", summary, count, truncate(body))
		}
	}

	h3Count := strings.Count(body, eventHeading)
	if h3Count != 2 {
		t.Errorf("expected 2 <h3> activity headings on /activity page, got %d\n%s", h3Count, truncate(body))
	}
}
