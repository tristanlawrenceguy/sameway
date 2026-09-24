package server_test

// Tests for task 0440/0441: search empty state link and copy fixes.
// These pin down that the zero-result search page links to /chat (not an
// empty search) and uses lowercase "try" after the em-dash. They fail today
// because the code still has href="/search?q=&prompt=something." and
// capitalised "Try".

import (
	"net/http"
	"strings"
	"testing"
)

// TestSearchEmptyStateLinkPointsToChat checks that clicking the empty-state
// link on a zero-result search page navigates to /chat, not back to an empty
// search. Covers Acceptance 1: the link must point to /chat, not to
// /search?q=&prompt=something which does nothing useful.
func TestSearchEmptyStateLinkPointsToChat(t *testing.T) {
	_, h := newApp(t)

	rec := get(t, h, "/search?q=nonexistent")
	wantStatus(t, rec, http.StatusOK)
	body := rec.Body.String()

	if !strings.Contains(body, `href="/chat?prompt=`) {
		t.Errorf("empty search must link to /chat with a prompt parameter\n%s", truncate(body))
	}

	// It must NOT link back to an empty search — that is the bug.
	if strings.Contains(body, `/search?q=&`) || strings.Contains(body, "/search?q=") && !strings.Contains(body, `href="/chat?prompt=`) {
		t.Errorf("empty search must not link to /search?q= (dead end)\n%s", truncate(body))
	}

	// The specific prompt text should match the canvas pattern: "Create+something."
	if !strings.Contains(body, "Create+something.") && !strings.Contains(body, "Create%20something.") {
		t.Errorf("prompt must say 'Create something.' to match the canvas empty state\n%s", truncate(body))
	}
}

// TestSearchEmptyStateTryIsLowercase checks that the word after the em-dash is
// lowercase "try" — matching sentence case. Covers Acceptance 2: the text after
// the em-dash reads "try different words here" with a lowercase t, not "Try".
func TestSearchEmptyStateTryIsLowercase(t *testing.T) {
	_, h := newApp(t)

	rec := get(t, h, "/search?q=nonexistent")
	wantStatus(t, rec, http.StatusOK)
	body := rec.Body.String()

	// The em-dash must be followed by lowercase "try", not capitalised.
	if strings.Contains(body, "— Try different words") {
		t.Errorf("the word after the em-dash must be lowercase 'try', not 'Try'\n%s", truncate(body))
	}
	if !strings.Contains(body, "— try different words") {
		t.Errorf("the empty-state text after the em-dash must read '— try different words'\n%s", truncate(body))
	}
}

// TestSearchEmptyStateStillShowsQuery checks that the empty-state page still
// displays "No matches for <query>" so the person knows what was searched.
// Covers Acceptance 3: the main message is preserved.
func TestSearchEmptyStateStillShowsQuery(t *testing.T) {
	_, h := newApp(t)

	rec := get(t, h, "/search?q=plumber")
	wantStatus(t, rec, http.StatusOK)
	body := rec.Body.String()

	if !strings.Contains(body, "No matches for") && !strings.Contains(body, `%q`) {
		t.Error("empty search must say 'No matches for'")
	}
	if !strings.Contains(body, "plumber") {
		t.Errorf("the query term must appear so the person knows what was searched\n%s", truncate(body))
	}

	// The heading must still be "No results".
	if !strings.Contains(body, `<h2 class="sw-empty__title">No results</h2>`) {
		t.Errorf("empty search must have an h2 'No results'\n%s", truncate(body))
	}
}

// TestSearchEmptyStateLinkIsFunctional verifies that the /chat page is reachable
// from the empty-state link. This confirms navigation actually works rather than
// returning a 404 or other error. Covers Acceptance 1 end-to-end: clicking the
// link takes you to /chat, not back to an empty search.
func TestSearchEmptyStateLinkIsFunctional(t *testing.T) {
	_, h := newApp(t)

	rec := get(t, h, "/search?q=nonexistent")
	wantStatus(t, rec, http.StatusOK)
	body := rec.Body.String()

	// Extract the href from the <a> tag in .sw-empty.
	linkStart := strings.Index(body, `href="/chat?prompt=`)
	if linkStart == -1 {
		t.Fatalf("cannot find chat link in empty-state\n%s", truncate(body))
	}
	substr := body[linkStart:]
	firstQuote := strings.Index(substr, `"`)
	if firstQuote == -1 {
		t.Fatalf("cannot find closing quote for chat link\n%s", truncate(body))
	}
	linkEnd := strings.Index(substr[firstQuote+1:], `"`)
	if linkEnd == -1 {
		t.Fatalf("cannot find closing quote for chat link\n%s", truncate(body))
	}
	href := substr[firstQuote+1 : firstQuote+1+linkEnd]

	// Follow the link and check status.
	resp := get(t, h, href)
	wantStatus(t, resp, http.StatusOK)

	// The chat page should show the prompt text pre-filled.
	if !strings.Contains(resp.Body.String(), "Create something.") {
		t.Errorf("chat page must show the 'Create something.' prompt\n%s", truncate(resp.Body.String()))
	}
}
