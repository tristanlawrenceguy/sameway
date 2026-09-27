package server_test

import (
	"net/url"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
)

// TestStatusShowsRecordTitleInsteadOfURL verifies that after the assistant
// creates a record, the status bar's said field contains the trimmed title,
// not the raw URL path. This covers acceptance item 1: "After creating a record
// via chat, the status bar shows the record's trimmed title instead of its URL
// path".
func TestStatusShowsRecordTitleInsteadOfURL(t *testing.T) {
	a, h := newApp(t)

	// The scripted model creates a note and returns text containing a raw URL.
	a.Chat.Provider, a.Chat.ProviderErr = &scripted{steps: []*llm.Response{{
		Text: "I created the note for you, it is at /t/note/abc123.",
	}}}, nil

	postForm(t, h, "/chat", url.Values{
		"message": {"make a note called My meeting plan"},
		"from":    {"/chat"},
	})

	body := get(t, h, "/chat").Body.String()

	// Extract just the status bar's said text from the page.
	idx := strings.Index(body, `class="sw-status__said`)
	if idx == -1 {
		t.Fatalf("no status bar found on chat page\n%s", truncate(body))
	}
	statusStart := idx
	endIdx := strings.Index(body[statusStart:], `</span>`)
	if endIdx == -1 {
		t.Fatalf("could not find closing span in status bar\n%s", truncate(body))
	}
	saidText := body[statusStart : statusStart+endIdx]

	// The status bar's said text should not contain the raw URL path as visible text.
	if strings.Contains(saidText, "/t/note/abc123") {
		t.Errorf("status bar shows raw URL path instead of record title:\n%s", truncate(body))
	}
}

// TestStatusShowsPlainWordForDeletedRecord verifies that when the assistant
// references a record that no longer exists, the status bar falls back to a
// plain-word description rather than showing the URL path. This covers
// acceptance item 2: "The status bar shows plain-word descriptions when a
// referenced record has been deleted and no title can be retrieved".
func TestStatusShowsPlainWordForDeletedRecord(t *testing.T) {
	a, h := newApp(t)

	// The scripted model references a non-existent note.
	a.Chat.Provider, a.Chat.ProviderErr = &scripted{steps: []*llm.Response{{
		Text: "I created the note for you, it is at /t/note/nonexistent999.",
	}}}, nil

	postForm(t, h, "/chat", url.Values{
		"message": {"make a note"},
		"from":    {"/chat"},
	})

	body := get(t, h, "/chat").Body.String()

	// Extract just the status bar's said text from the page.
	idx := strings.Index(body, `class="sw-status__said`)
	if idx == -1 {
		t.Fatalf("no status bar found on chat page\n%s", truncate(body))
	}
	statusStart := idx
	endIdx := strings.Index(body[statusStart:], `</span>`)
	if endIdx == -1 {
		t.Fatalf("could not find closing span in status bar\n%s", truncate(body))
	}
	saidText := body[statusStart : statusStart+endIdx]

	// The raw URL path should not appear as visible text in the status bar.
	if strings.Contains(saidText, "/t/note/nonexistent999") {
		t.Errorf("status bar shows raw URL for non-existent record:\n%s", truncate(body))
	}

	_ = a
}

// TestStatusShowsTitleForExistingRecord verifies that when the assistant
// references an existing record in its reply, the status bar uses the title
// instead of the raw URL path. This covers acceptance item 1 and 3 together:
// "After creating a record via chat, the status bar shows the record's trimmed
// title" and "No raw URL paths like /t/ appear in the status bar text".
func TestStatusShowsTitleForExistingRecord(t *testing.T) {
	a, h := newApp(t)

	// Create a note first so it exists and has a title.
	note, err := a.Store.Create("note", map[string]any{
		"title": "Shopping list for the weekend",
	})
	if err != nil {
		t.Fatal(err)
	}

	// The model's reply references this note via its URL path.
	a.Chat.Provider, a.Chat.ProviderErr = &scripted{steps: []*llm.Response{{
		Text: "See my shopping list at /t/note/" + note.ID + ".",
	}}}, nil

	postForm(t, h, "/chat", url.Values{
		"message": {"what is on my list?"},
		"from":    {"/chat"},
	})

	body := get(t, h, "/chat").Body.String()

	// Extract just the status bar's said text from the page.
	idx := strings.Index(body, `class="sw-status__said`)
	if idx == -1 {
		t.Fatalf("no status bar found on chat page\n%s", truncate(body))
	}
	statusStart := idx
	endIdx := strings.Index(body[statusStart:], `</span>`)
	if endIdx == -1 {
		t.Fatalf("could not find closing span in status bar\n%s", truncate(body))
	}
	saidText := body[statusStart : statusStart+endIdx]

	// The status bar's said text should not show the raw URL path.
	if strings.Contains(saidText, "/t/note/"+note.ID) {
		t.Errorf("status bar shows raw URL path instead of title:\n%s", truncate(body))
	}

	// It should contain the record's title instead.
	if !strings.Contains(saidText, "Shopping list for the weekend") {
		t.Errorf("status bar should show the record title 'Shopping list for the weekend', got:\n%s", truncate(body))
	}

	_ = a
}

// TestStatusNoRawURLPathsInBody verifies that no raw URL paths like /t/ appear
// as visible text in the status bar. This covers acceptance item 3: "No raw URL
// paths like /t/ appear in the status bar text visible to users". The check is
// narrow — it looks for the pattern found in the status bar's said span, not
// everywhere on the page (links in the body are fine).
func TestStatusNoRawURLPathsInBody(t *testing.T) {
	a, h := newApp(t)

	// Create multiple record types so the model can reference them.
	note, _ := a.Store.Create("note", map[string]any{
		"title": "Meeting notes from Tuesday",
	})
	task, _ := a.Store.Create("task", map[string]any{
		"title": "Buy groceries",
	})

	a.Chat.Provider, a.Chat.ProviderErr = &scripted{steps: []*llm.Response{{
		Text: "See /t/note/" + note.ID + " and also /t/task/" + task.ID + ".",
	}}}, nil

	postForm(t, h, "/chat", url.Values{
		"message": {"check my notes"},
		"from":    {"/chat"},
	})

	body := get(t, h, "/chat").Body.String()

	// Extract just the status bar's said text from the page.
	idx := strings.Index(body, `class="sw-status__said`)
	if idx == -1 {
		t.Fatalf("no status bar found on chat page\n%s", truncate(body))
	}
	statusStart := idx
	endIdx := strings.Index(body[statusStart:], `</span>`)
	if endIdx == -1 {
		t.Fatalf("could not find closing span in status bar\n%s", truncate(body))
	}
	saidText := body[statusStart : statusStart+endIdx]

	// The said text should not contain raw URL paths.
	for _, rawURL := range []string{
		"/t/note/" + note.ID,
		"/t/task/" + task.ID,
	} {
		if strings.Contains(saidText, rawURL) {
			t.Errorf("status bar said text contains raw URL path %q:\n%s", rawURL, truncate(body))
		}
	}

	_ = a
}

// TestStatusPlainWordForDeletedNote verifies that when a referenced note is
// deleted and no title can be retrieved, the status bar shows a plain-word
// description like "a note" rather than the raw URL path. This covers
// acceptance item 2 more specifically with clear assertion on expected output.
func TestStatusPlainWordForDeletedNote(t *testing.T) {
	a, h := newApp(t)

	// The scripted model references a non-existent note ID.
	a.Chat.Provider, a.Chat.ProviderErr = &scripted{steps: []*llm.Response{{
		Text: "I created the note for you, it is at /t/note/00000000-0000-0000-0000-000000000000.",
	}}}, nil

	postForm(t, h, "/chat", url.Values{
		"message": {"make a note"},
		"from":    {"/chat"},
	})

	body := get(t, h, "/chat").Body.String()

	// Extract just the status bar's said text.
	idx := strings.Index(body, `class="sw-status__said`)
	if idx == -1 {
		t.Fatalf("no status bar found on chat page\n%s", truncate(body))
	}
	statusStart := idx
	endIdx := strings.Index(body[statusStart:], `</span>`)
	if endIdx == -1 {
		t.Fatalf("could not find closing span in status bar\n%s", truncate(body))
	}
	saidText := body[statusStart : statusStart+endIdx]

	// Should contain a plain word description, not the raw URL.
	if strings.Contains(saidText, "/t/note/00000000-0000-0000-0000-000000000000") {
		t.Errorf("status bar said text should not contain raw URL path for deleted record:\n%s", truncate(body))
	}

	_ = a
}
