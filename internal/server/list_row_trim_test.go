package server_test

import (
	"strings"
	"testing"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/render/htmltest"
)

// TestListRowH2TrimsLongTitle verifies that a listing page's row h2 heading is
// trimmed to at most six words when the note title exceeds them. (Acceptance 2.)
func TestListRowH2TrimsLongTitle(t *testing.T) {
	a, h := newApp(t)

	// Create a note with a long title — more than six words.
	wantFull := "This is a very long title that definitely exceeds six"
	_, err := a.Store.Create("note", map[string]any{
		"title": wantFull,
	})
	if err != nil {
		t.Fatal(err)
	}

	page := get(t, h, "/t/note")
	wantStatus(t, page, 200)
	body := page.Body.String()

	doc := parse(t, page)

	for _, h2 := range doc.Elements("h2") {
		class, hasClass := htmltest.Attr(h2, "class")
		if !hasClass || !strings.Contains(class, "sw-row__title") {
			continue
		}
		text := strings.TrimSpace(htmltest.Text(h2))
		words := strings.Fields(text)
		if len(words) > 6 {
			t.Errorf("row h2 should be trimmed to at most 6 words, got %d: %q", len(words), text)
		}

		// The full title should not appear as visible row heading text.
		if text == wantFull {
			t.Errorf("row h2 should NOT contain the full untrimmed %d-word title: %q", len(strings.Fields(wantFull)), wantFull)
		}
	}

	// The full untrimmed title must still appear somewhere on the page (e.g. activity log).
	if !strings.Contains(body, wantFull) {
		t.Errorf("page should show the full untrimmed title in a non-heading context: %q", wantFull)
	}
}

// TestListRowH3TrimsLongTitle verifies that grouped listing rows (which use h3)
// are also trimmed to at most six words. Grouped rows appear when the type has a
// datetime field and records fall into time-based groups like "Today". (Acceptance 2.)
func TestListRowH3TrimsLongTitle(t *testing.T) {
	a, h := newApp(t)

	_, err := a.Store.Create("task", map[string]any{
		"title": "This is a very long task title that definitely exceeds six words limit today now please",
		"due":   time.Now().Format(time.RFC3339),
	})
	if err != nil {
		t.Fatal(err)
	}

	page := get(t, h, "/t/task")
	wantStatus(t, page, 200)
	body := page.Body.String()

	doc := parse(t, page)
	fullTitle := "This is a very long task title that definitely exceeds six words limit today now please"

	for _, h3 := range doc.Elements("h3") {
		class, hasClass := htmltest.Attr(h3, "class")
		if !hasClass || !strings.Contains(class, "sw-row__title") {
			continue
		}
		text := strings.TrimSpace(htmltest.Text(h3))
		words := strings.Fields(text)
		if len(words) > 6 {
			t.Errorf("row h3 should be trimmed to at most 6 words, got %d: %q", len(words), text)
		}

		if text == fullTitle {
			t.Errorf("row h3 should NOT contain the full untrimmed %d-word title: %q", len(strings.Fields(fullTitle)), fullTitle)
		}
	}

	// The full untrimmed title must still appear somewhere on the page.
	if !strings.Contains(body, fullTitle) {
		t.Errorf("page should show the full untrimmed title in a non-heading context: %q", fullTitle)
	}
}

// TestListRowH2LeavesShortTitleUntouched verifies that a listing page's row h2 is
// unchanged when the title is within the six-word limit. (Acceptance 2.)
func TestListRowH2LeavesShortTitleUntouched(t *testing.T) {
	a, h := newApp(t)

	want := "Three words only"
	_, err := a.Store.Create("note", map[string]any{
		"title": want,
	})
	if err != nil {
		t.Fatal(err)
	}

	page := get(t, h, "/t/note")
	wantStatus(t, page, 200)

	doc := parse(t, page)

	for _, h2 := range doc.Elements("h2") {
		class, hasClass := htmltest.Attr(h2, "class")
		if !hasClass || !strings.Contains(class, "sw-row__title") {
			continue
		}
		text := strings.TrimSpace(htmltest.Text(h2))
		if text != want {
			t.Errorf("short title should be unchanged in row h2: got %q, want %q", text, want)
		}
	}
}
