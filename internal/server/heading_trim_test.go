package server_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/render/htmltest"
)

// TestTrimTitleCutsLongTitles verifies that page titles exceeding six words
// are trimmed to exactly six words, so no h1 on any record detail page
// exceeds the limit. (Acceptance 1: no h1 heading exceeds 6 words.)
func TestTrimTitleCutsLongTitles(t *testing.T) {
	a, h := newApp(t)

	// Create a note with an extremely long title — more than 6 words.
	rec, err := a.Store.Create("note", map[string]any{
		"title": "This is a very long title that definitely exceeds six words limit today",
	})
	if err != nil {
		t.Fatal(err)
	}

	// Fetch the detail page. The h1 should be trimmed to 6 words max.
	doc := parse(t, get(t, h, "/t/note/"+rec.ID))
	h1s := doc.Elements("h1")
	if len(h1s) != 1 {
		t.Fatalf("expected one h1, got %d", len(h1s))
	}
	text := htmltest.Text(h1s[0])

	// Count words in the heading.
	words := strings.Split(text, " ")
	if len(words) > 6 {
		t.Errorf("h1 should be trimmed to at most 6 words, got %d: %q", len(words), text)
	}
}

// TestTrimTitleLeavesShortTitlesUntouched verifies that titles already within
// the six-word limit are not altered. (Acceptance 1.)
func TestTrimTitleLeavesShortTitlesUntouched(t *testing.T) {
	a, h := newApp(t)

	rec, err := a.Store.Create("note", map[string]any{
		"title": "Three words only",
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

	want := "Three words only"
	if text != want {
		t.Errorf("short title should be unchanged: got %q, want %q", text, want)
	}
}

// TestSearchTitleTrimsLongQueries verifies that the search page h1 is trimmed
// when a long query makes it exceed six words. (Acceptance 1.)
func TestSearchTitleTrimsLongQueries(t *testing.T) {
	_, h := newApp(t)

	page := get(t, h, "/search?q=this+is+a+very+long+search+query+that+exceeds+six")
	wantStatus(t, page, http.StatusOK)
	doc := parse(t, page)
	h1s := doc.Elements("h1")
	if len(h1s) != 1 {
		t.Fatalf("expected one h1 on search page, got %d", len(h1s))
	}
	text := htmltest.Text(h1s[0])

	words := strings.Split(text, " ")
	if len(words) > 6 {
		t.Errorf("search h1 should be trimmed to at most 6 words, got %d: %q", len(words), text)
	}
}

// TestSearchTitleWithShortQueryIsUnchanged verifies that a short query does not
// get truncated. (Acceptance 1.)
func TestSearchTitleWithShortQueryIsUnchanged(t *testing.T) {
	_, h := newApp(t)

	page := get(t, h, "/search?q=plumber")
	wantStatus(t, page, http.StatusOK)
	doc := parse(t, page)
	h1s := doc.Elements("h1")
	if len(h1s) != 1 {
		t.Fatalf("expected one h1 on search page, got %d", len(h1s))
	}
	text := htmltest.Text(h1s[0])

	want := "Search: plumber"
	if text != want {
		t.Errorf("short query title should be unchanged: got %q, want %q", text, want)
	}
}

// TestDesignPageHeadingsUnchanged verifies that component example headings on
// /design are not trimmed — they are developer-authored manifest examples and
// out of scope. (Acceptance 4.)
func TestDesignPageHeadingsUnchanged(t *testing.T) {
	_, h := newApp(t)

	rec := get(t, h, "/design")
	wantStatus(t, rec, http.StatusOK)
	body := rec.Body.String()

	// The datepicker examples have specific headings that must not be trimmed.
	for _, wantHeading := range []string{
		"datepicker — default",
		"datepicker — chosen",
		"datepicker — error",
	} {
		if !strings.Contains(body, "<h4 class=\"sw-small\">"+wantHeading+"</h4>") {
			t.Errorf("/design should contain the untrimmed heading %q\n%s", wantHeading, truncate(body))
		}
	}
}

// TestBlockFocusPageTrimsLongTitles verifies that pop-out block pages (canvas
// focus) trim long captions/titles to 6 words. (Acceptance 1.)
func TestBlockFocusPageTrimsLongTitles(t *testing.T) {
	_, h := newApp(t)

	// Use a calendar block with an extremely long caption.
	var blk struct{ ID string }
	decode(t, postJSON(t, h, "POST", "/api/block", map[string]any{
		"component": "calendar",
		"props": map[string]any{
			"month":   "2026-09",
			"today":   "2026-09-11",
			"detail":  "brief",
			"caption": "This is a very long caption that definitely exceeds six words limit today for sure",
		},
	}), &blk)

	doc := parse(t, get(t, h, "/canvas/"+blk.ID))
	h1s := doc.Elements("h1")
	if len(h1s) != 1 {
		t.Fatalf("expected one h1 on focus page, got %d", len(h1s))
	}
	text := htmltest.Text(h1s[0])

	words := strings.Split(text, " ")
	if len(words) > 6 {
		t.Errorf("focus page h1 should be trimmed to at most 6 words, got %d: %q", len(words), text)
	}
}
