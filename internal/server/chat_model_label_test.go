package server_test

import (
	"net/http"
	"strings"
	"testing"
)

// The chat page never shows an internal provider name like "Claude Code" as a
// static label. It says "Assistant" instead, so no model-specific identifier
// leaks into the user-facing UI. Covers acceptance item 1.
func TestChatPageShowsAssistantNotProviderName(t *testing.T) {
	t.Parallel()
	_, h := newApp(t)

	rec := get(t, h, "/chat")
	wantStatus(t, rec, http.StatusOK)
	body := rec.Body.String()

	if strings.Contains(body, "Claude Code") {
		t.Errorf("/chat must not contain the provider name \"Claude Code\"\nbody:\n%s", excerpt(body))
	}

	if !strings.Contains(body, ">Assistant</span>") {
		t.Errorf("/chat should show \"Assistant\" as the model label\nbody:\n%s", excerpt(body))
	}
}
