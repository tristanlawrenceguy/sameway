package server_test

// A search that finds nothing says so where it is heard, in the page's
// title, says what was looked for in plain quotes, and offers the assistant
// the same words to find it with.

import (
	"net/http"
	"strings"
	"testing"
)

func TestASearchThatFindsNothingSaysSoAndOffersTheAssistant(t *testing.T) {
	_, h := newApp(t)

	body := get(t, h, "/search?q=plumber").Body.String()
	if !strings.Contains(body, "<title>Search: plumber, no results") || !strings.Contains(body, "<h1>Search: plumber</h1>") {
		t.Errorf("the title and heading say the search found nothing, and for what\n%s", truncate(body))
	}
	if !strings.Contains(body, "Nothing matches “plumber”. Try different words, or") {
		t.Errorf("the panel says what was looked for, in plain quotes\n%s", truncate(body))
	}
	if strings.Contains(body, `\"`) || strings.Contains(body, "—") {
		t.Errorf("no Go quoting and no dash in the words\n%s", truncate(body))
	}
	if !strings.Contains(body, `href="/chat?prompt=Find&#43;plumber"`) && !strings.Contains(body, `href="/chat?prompt=Find+plumber"`) {
		t.Errorf("the assistant is asked to find what was searched\n%s", truncate(body))
	}
	if chat := get(t, h, "/chat?prompt=Find+plumber"); chat.Code != http.StatusOK || !strings.Contains(chat.Body.String(), "Find plumber") {
		t.Errorf("the assistant's page opens with the words ready to send")
	}
}

func TestASearchThatFindsSomethingSaysHowManyInItsTitle(t *testing.T) {
	_, h := newApp(t)
	wantStatus(t, postJSON(t, h, http.MethodPost, "/api/note", map[string]any{"title": "Call the plumber"}), http.StatusCreated)
	body := get(t, h, "/search?q=plumber").Body.String()
	if !strings.Contains(body, "<title>Search: plumber, 1 result") {
		t.Errorf("the title says how many were found\n%s", truncate(body))
	}
}
