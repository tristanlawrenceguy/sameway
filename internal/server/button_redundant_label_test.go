package server_test

// Tests for redundant button labels (task 0172).
// These pin that no button on any page has both visible text AND an
// aria-label containing the same words — a screen-reader redundancy.

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	"golang.org/x/net/html"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/render/htmltest"
)

// TestDeleteChatButtonNamesItselfOnce checks that the delete-chat button
// shows a word a person can see and say (WCAG 2.5.3), and takes its whole
// name, "Delete chat <title>", from its content alone: no aria-label beside
// it, so nothing is named twice and the visible word is part of the name.
func TestDeleteChatButtonNamesItselfOnce(t *testing.T) {
	a, h := newApp(t)
	a.Chat.Provider, a.Chat.ProviderErr = &scripted{steps: []*llm.Response{{Text: "Planned."}, {Text: "Noted."}}}, nil

	// Create two chats so the delete button appears in the chat list.
	wantStatus(t, postForm(t, h, "/chat", url.Values{"message": {"plan the garden"}, "from": {"/chat"}}), http.StatusSeeOther)
	wantStatus(t, postForm(t, h, "/chat/new", url.Values{"from": {"/chat"}}), http.StatusSeeOther)
	wantStatus(t, postForm(t, h, "/chat", url.Values{"message": {"and the kitchen"}, "from": {"/chat"}}), http.StatusSeeOther)

	page := get(t, h, "/chat").Body.String()
	if !strings.Contains(page, `>Delete<span class="sw-visually-hidden"> chat and the kitchen</span></button>`) {
		t.Fatalf("the delete-chat button should show Delete and say which chat in hidden text; page:\n%s", truncate(page))
	}
	if strings.Contains(page, `aria-label="Delete chat`) {
		t.Error("the delete-chat button should take its name from its content, not an aria-label beside it")
	}
}

// TestNoButtonHasVisibleTextMatchingAriaLabel walks every page's HTML using the
// html parser, finds all buttons with aria-labels, and checks that none of them
// also have visible text containing words from their own aria-label. This covers
// Acceptance 1 globally across surfaces.
func TestNoButtonHasVisibleTextMatchingAriaLabel(t *testing.T) {
	_, h := newApp(t)

	// Create two chats so the delete button appears in conversation.html.
	wantStatus(t, postForm(t, h, "/chat", url.Values{"message": {"plan the garden"}, "from": {"/chat"}}), http.StatusSeeOther)
	wantStatus(t, postForm(t, h, "/chat/new", url.Values{"from": {"/chat"}}), http.StatusSeeOther)
	wantStatus(t, postForm(t, h, "/chat", url.Values{"message": {"and the kitchen"}, "from": {"/chat"}}), http.StatusSeeOther)

	for _, path := range []string{"/", "/chat", "/activity"} {
		rec := get(t, h, path)
		wantStatus(t, rec, http.StatusOK)

		doc, err := htmltest.Parse(rec.Body.String())
		if err != nil {
			t.Fatalf("%s: parse error: %v", path, err)
		}

		for _, btn := range doc.Elements("button") {
			checkButtonRedundancy(t, btn, path)
		}
	}
}

// checkButtonRedundancy fails if a parsed <button> node has both an aria-label
// and visible text that shares any word with it.
func checkButtonRedundancy(t *testing.T, n *html.Node, path string) {
	t.Helper()

	ariaLabel, ok := htmltest.Attr(n, "aria-label")
	if !ok || ariaLabel == "" {
		return // no aria-label on this button — fine
	}

	visibleText := strings.TrimSpace(htmltest.Text(n))
	if visibleText == "" {
		return // icon-only or empty button — fine
	}

	// Check if any word in the visible text also appears in the aria-label.
	for _, w := range words(visibleText) {
		lowerW := strings.ToLower(w)
		if strings.Contains(strings.ToLower(ariaLabel), lowerW) {
			t.Errorf("%s: button has redundant visible label %q that duplicates aria-label=%q — screen readers announce it twice (Acceptance 1)", path, visibleText, ariaLabel)
		}
	}
}

// words splits a string into alphabetic words.
func words(s string) []string {
	var words []string
	var cur strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
			cur.WriteRune(r)
		} else if cur.Len() > 0 {
			words = append(words, cur.String())
			cur.Reset()
		}
	}
	if cur.Len() > 0 {
		words = append(words, cur.String())
	}
	return words
}
