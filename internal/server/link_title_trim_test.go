package server_test

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
)

// TestLinkTitleShowsTrimmedTitle verifies that inline record links in chat
// replies display the trimmed title (six words max), not the raw URL path.
// This pins acceptance item 1: when the assistant creates a record, every
// inline link to it shows the trimmed title as visible link text.
func TestLinkTitleShowsTrimmedTitle(t *testing.T) {
	a, h := newApp(t)

	// Create a note with a long title — more than six words.
	rec, err := a.Store.Create("note", map[string]any{
		"title": "This is the very first meeting of the planning committee today",
	})
	if err != nil {
		t.Fatal(err)
	}

	// Post an assistant message that references this note via /t/note/<id>.
	postJSON(t, h, http.MethodPost, "/api/message", map[string]any{
		"role":    "assistant",
		"content": fmt.Sprintf("/t/note/%s is ready for review.", rec.ID),
	})

	body := get(t, h, "/chat").Body.String()

	// The link href must point to the correct path.
	if !strings.Contains(body, `href="/t/note/`+rec.ID) {
		t.Errorf("/chat should contain a link to /t/note/%s\n%s", rec.ID, truncate(body))
	}

	// The visible text of that link must NOT be the raw URL path.
	rawLink := `<a class="sw-link" href="/t/note/` + rec.ID + `">/t/note/` + rec.ID + "</a>"
	if strings.Contains(body, rawLink) {
		t.Errorf("/chat should not show raw URL as link text:\n%s", truncate(body))
	}

	// The visible text must be the trimmed title (at most six words).
	linkPattern := `<a class="sw-link" href="/t/note/` + rec.ID + `">`
	idx := strings.Index(body, linkPattern)
	if idx < 0 {
		t.Fatalf("could not find the link anchor in chat page\n%s", truncate(body))
	}

	// Extract text between the opening <a> tag and closing </a>.
	rest := body[idx+len(linkPattern):]
	closeIdx := strings.Index(rest, "</a>")
	if closeIdx < 0 {
		t.Fatalf("could not find closing </a> of the link\n%s", truncate(body))
	}
	linkText := rest[:closeIdx]

	words := strings.Fields(linkText)
	if len(words) > 6 {
		t.Errorf("link text must be trimmed to at most six words, got %d: %q\npage:\n%s", len(words), linkText, truncate(body))
	}

	// The first three words should match the start of the original title.
	wantStart := "This is the"
	if !strings.HasPrefix(strings.TrimSpace(linkText), wantStart) {
		t.Errorf("link text should start with trimmed title %q; got %q\npage:\n%s", wantStart, linkText, truncate(body))
	}

	_ = a
}

// TestLinkTitleMatchesProseStatedTitle verifies that inline link text in an
// assistant reply matches the record title stated in surrounding prose. This
// pins acceptance item 2: link text matches the record title from the same reply.
func TestLinkTitleMatchesProseStatedTitle(t *testing.T) {
	a, h := newApp(t)

	// Create a note with a long title.
	rec, err := a.Store.Create("note", map[string]any{
		"title": "Budget review for Q3 2024 quarterly report summary draft",
	})
	if err != nil {
		t.Fatal(err)
	}

	// The assistant's reply prose states the trimmed title and also links to it.
	postJSON(t, h, http.MethodPost, "/api/message", map[string]any{
		"role":    "assistant",
		"content": fmt.Sprintf("I wrote the Budget review for Q3 2024 quarterly report summary draft at /t/note/%s.", rec.ID),
	})

	body := get(t, h, "/chat").Body.String()

	// Find the link's visible text.
	linkPattern := `<a class="sw-link" href="/t/note/` + rec.ID + `">`
	idx := strings.Index(body, linkPattern)
	if idx < 0 {
		t.Fatalf("could not find the /t/note link in chat page\n%s", truncate(body))
	}

	rest := body[idx+len(linkPattern):]
	closeIdx := strings.Index(rest, "</a>")
	if closeIdx < 0 {
		t.Fatalf("could not find closing </a> of the link\n%s", truncate(body))
	}
	linkText := rest[:closeIdx]

	// The trimmed title (first 6 words) is what both prose and link should show.
	wantTrimmed := "Budget review for Q3 2024 quarterly…"
	if linkText != wantTrimmed {
		t.Errorf("link text %q should match the trimmed prose-stated title\nwant: %q\ngot: %q\npage:\n%s", linkText, wantTrimmed, linkText, truncate(body))
	}

	_ = a
}

// TestLinkTitleForShortTitleIsUnchanged verifies that titles already within
// the six-word limit are not altered in link text. This pins acceptance item 1
// (no regression for normal-length titles).
func TestLinkTitleForShortTitleIsUnchanged(t *testing.T) {
	a, h := newApp(t)

	rec, err := a.Store.Create("note", map[string]any{
		"title": "Quick notes",
	})
	if err != nil {
		t.Fatal(err)
	}

	postJSON(t, h, http.MethodPost, "/api/message", map[string]any{
		"role":    "assistant",
		"content": fmt.Sprintf("See /t/note/%s.", rec.ID),
	})

	body := get(t, h, "/chat").Body.String()

	linkPattern := `<a class="sw-link" href="/t/note/` + rec.ID + `">`
	idx := strings.Index(body, linkPattern)
	if idx < 0 {
		t.Fatalf("could not find the /t/note link in chat page\n%s", truncate(body))
	}

	rest := body[idx+len(linkPattern):]
	closeIdx := strings.Index(rest, "</a>")
	if closeIdx < 0 {
		t.Fatalf("could not find closing </a> of the link\n%s", truncate(body))
	}
	linkText := rest[:closeIdx]

	want := "Quick notes"
	if linkText != want {
		t.Errorf("short title must appear unchanged in link text: got %q, want %q\npage:\n%s", linkText, want, truncate(body))
	}

	_ = a
}

// TestLinkTitleForDeletedRecordShowsRawURL verifies that when a record has been
// deleted and cannot be resolved, the raw URL path remains as fallback. This is
// an edge case from the plan: unresolved paths fall back to showing the address.
func TestLinkTitleForDeletedRecordShowsRawURL(t *testing.T) {
	_, h := newApp(t)

	// The assistant references a /t/note/<id> that does not exist.
	postJSON(t, h, http.MethodPost, "/api/message", map[string]any{
		"role":    "assistant",
		"content": "See /t/note/nonexistent123 for details.",
	})

	body := get(t, h, "/chat").Body.String()

	// For a non-existent record the raw path is used as fallback.
	rawLink := `<a class="sw-link" href="/t/note/nonexistent123">/t/note/nonexistent123</a>`
	if !strings.Contains(body, rawLink) {
		t.Errorf("link to deleted record should show raw URL as text:\nwant: %s\n%s", rawLink, truncate(body))
	}

	_ = h
}

// TestReceiptSectionExistsAfterModelAction verifies that the "Changes made"
// receipt section renders after the model makes an action through chat. This is
// acceptance item 3: receipts continue to work correctly with links showing
// trimmed titles (the .detail field, not linkTitle, names them).
func TestReceiptSectionExistsAfterModelAction(t *testing.T) {
	a, h := newApp(t)

	// Use a scripted model that adds a canvas card.
	a.Chat.Provider, a.Chat.ProviderErr = &scripted{steps: []*llm.Response{
		toolCall("add_component", map[string]any{"component": "card", "props": map[string]any{"title": "Shopping"}}),
	}}, nil

	postForm(t, h, "/chat", url.Values{
		"message": {"add a card"},
		"from":    {"/"},
	})

	body := get(t, h, "/chat").Body.String()

	// The receipt section should exist.
	if !strings.Contains(body, `class="sw-message__changes"`) {
		t.Errorf("/chat after model action should show a 'Changes made' list\n%s", truncate(body))
	}

	_ = a
}
