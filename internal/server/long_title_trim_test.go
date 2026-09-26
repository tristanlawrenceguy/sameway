package server_test

import (
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/render/htmltest"
)

// TestDetailPageH1TrimsLongSingleWordTitle verifies that a detail page h1
// trims a very-long single-word title (no spaces). A 160-character string of
// repeated "a" characters is not split into words by strings.Fields, so the
// existing trimTitle word-count check does nothing. This test pins that the h1
// no longer displays the full untrimmed string. (Backlog #0449; Acceptance 1.)
func TestDetailPageH1TrimsLongSingleWordTitle(t *testing.T) {
	a, h := newApp(t)

	// Create a note with a single very-long word — no spaces.
	longWord := strings.Repeat("a", 160)
	rec, err := a.Store.Create("note", map[string]any{
		"title": longWord,
	})
	if err != nil {
		t.Fatal(err)
	}

	doc := parse(t, get(t, h, "/t/note/"+rec.ID))
	h1s := doc.Elements("h1")
	if len(h1s) != 1 {
		t.Fatalf("expected one h1, got %d", len(h1s))
	}
	text := htmltest.Text(h1s[0])

	// The full long word must NOT appear in the heading.
	if strings.Contains(text, longWord) {
		t.Errorf("h1 should not contain the untrimmed 160-char title: got %q (len=%d)", text, len(text))
	}

	// The heading must be short — at most ~85 characters including ellipsis.
	if len([]rune(text)) > 85 {
		t.Errorf("h1 should be trimmed to a reasonable length; got %d runes: %q", len([]rune(text)), text)
	}
}

// TestDetailPageH1LeavesSixWordTitleUntouched verifies that a title of exactly
// six words (even if the total character count is high) is displayed in full,
// with no ellipsis. (Acceptance 5.)
func TestDetailPageH1LeavesSixWordTitleUntouched(t *testing.T) {
	a, h := newApp(t)

	// Six medium-length words — well within any reasonable char limit.
	want := "The quick brown fox jumps over"
	rec, err := a.Store.Create("note", map[string]any{
		"title": want,
	})
	if err != nil {
		t.Fatal(err)
	}

	doc := parse(t, get(t, h, "/t/note/"+rec.ID))
	h1s := doc.Elements("h1")
	if len(h1s) != 1 {
		t.Fatalf("expected one h1, got %d", len(h1s))
	}
	text := htmltest.Text(h1s[0])

	if text != want {
		t.Errorf("six-word title should be unchanged: got %q, want %q", text, want)
	}
}

// TestDetailPageH1TrimsLongTokenWithSpaces verifies that a title where one token
// exceeds the character limit (even though total words ≤ 6) is still trimmed.
// For example "verylongword word2 word3 word4 word5 word6" has 6 words but the
// first token alone is very long. (Backlog #0449; Acceptance 1.)
func TestDetailPageH1TrimsLongTokenWithSpaces(t *testing.T) {
	a, h := newApp(t)

	oneVeryLong := strings.Repeat("x", 120)
	fullTitle := oneVeryLong + " word2 word3 word4 word5" // 6 words total
	rec, err := a.Store.Create("note", map[string]any{
		"title": fullTitle,
	})
	if err != nil {
		t.Fatal(err)
	}

	doc := parse(t, get(t, h, "/t/note/"+rec.ID))
	h1s := doc.Elements("h1")
	if len(h1s) != 1 {
		t.Fatalf("expected one h1, got %d", len(h1s))
	}
	text := htmltest.Text(h1s[0])

	// The full untrimmed title must not appear in the heading.
	if text == fullTitle {
		t.Errorf("h1 should be trimmed when a single token is very long: got %q (len=%d)", text, len(text))
	}

	// Should still contain some of the original words for context.
	words := strings.Fields(text)
	if len(words) > 6 {
		t.Errorf("h1 should not exceed 6 words after trim: got %d words in %q", len(words), text)
	}
}

// TestListRowH2TrimsLongSingleWordTitle verifies that a list row h2
// heading trims a very-long single-word title. (Acceptance 3.)
func TestListRowH2TrimsLongSingleWordTitle(t *testing.T) {
	a, h := newApp(t)

	longWord := strings.Repeat("c", 160)
	_, err := a.Store.Create("note", map[string]any{
		"title": longWord,
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

		// The full long word must not appear in the heading text.
		if text == longWord {
			t.Errorf("row h2 should not contain untrimmed 160-char title: got %q (len=%d)", text, len(text))
		}

		// Must be reasonably short.
		if len([]rune(text)) > 85 {
			t.Errorf("row h2 should be trimmed; got %d runes: %q", len([]rune(text)), text)
		}
	}

	// The full untrimmed title must still appear somewhere on the page.
	if !strings.Contains(body, longWord) {
		t.Errorf("page should show the full title in a non-heading context")
	}
}
