package server_test

import (
	"net/http"
	"strings"
	"testing"
)

// TestChatPageNoSystemDidNotFinish verifies acceptance item 1 and 2 together:
// opening /chat with an error-role message in the store must not display the
// literal machine-language phrase "System did not finish", and instead shows a
// user-friendly label. This is what backlog 0669 reports as visible to users.
func TestChatPageNoSystemDidNotFinish(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)

	_, _ = a.Store.Create("message", map[string]any{
		"role":    "user",
		"content": "hello",
	})
	_, err := a.Store.Create("message", map[string]any{
		"role":    "error",
		"content": "Could not reach the model.",
	})
	if err != nil {
		t.Fatal(err)
	}

	rec := get(t, h, "/chat")
	wantStatus(t, rec, http.StatusOK)
	body := rec.Body.String()

	// Acceptance item 1: no page displays "System did not finish".
	if strings.Contains(body, "System did not finish") {
		t.Errorf("/chat must not display the machine-language phrase \"System did not finish\"\nbody:\n%s", body)
	}

	// Acceptance item 2: failed messages show a user-friendly label.
	if !strings.Contains(body, ">Sameway</span>") {
		t.Errorf("/chat error-role message should default author to \"Sameway\" instead of \"System\"\nbody contains:\n%s", excerpt(body))
	}

	// The failure indicator text must be plain language, not the raw phrase.
	if strings.Contains(strings.ToLower(body), "did not finish") {
		t.Errorf("/chat error-role message must not show \"did not finish\" as visible text\nbody:\n%s", body)
	}

	// The user-friendly failure indicator must appear.
	if !strings.Contains(body, `class="sw-message__failed"`) {
		t.Errorf("error-role message must still render the .sw-message__failed span;\nbody:\n%s", excerpt(body))
	}

	// The error content itself should be visible (sanitised).
	if !strings.Contains(body, "Could not reach the model") {
		t.Errorf("/chat should display the error content text\nbody:\n%s", body)
	}
}

// TestHomePageCanvasNoSystemDidNotFinish checks that the home page canvas also
// does not show "System did not finish" when an error-role message is present.
// The canvas renders messages through the same path as /chat, so this covers
// acceptance item 1 on the other surface where chat appears.
func TestHomePageCanvasNoSystemDidNotFinish(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)

	_, _ = a.Store.Create("message", map[string]any{
		"role":    "user",
		"content": "test",
	})
	_, err := a.Store.Create("message", map[string]any{
		"role":    "error",
		"content": "Could not reach the model.",
	})
	if err != nil {
		t.Fatal(err)
	}

	rec := get(t, h, "/")
	wantStatus(t, rec, http.StatusOK)
	body := rec.Body.String()

	if strings.Contains(body, "System did not finish") {
		t.Errorf("home page must not display \"System did not finish\"\nbody:\n%s", excerpt(body))
	}

	if !strings.Contains(body, ">Sameway</span>") {
		t.Errorf("home page error-role message should default author to \"Sameway\"\nbody:\n%s", excerpt(body))
	}
}

// TestChatPageUserAndAssistantNotAffected verifies that normal user and assistant
// messages are not changed by the error-role fix — "You" for user, "Assistant"
// for assistant still work correctly. This prevents regression on the other roles.
func TestChatPageUserAndAssistantNotAffected(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)

	_, _ = a.Store.Create("message", map[string]any{
		"role":    "user",
		"content": "hello there",
	})
	_, err := a.Store.Create("message", map[string]any{
		"role":    "assistant",
		"content": "Hi! How can I help?",
	})
	if err != nil {
		t.Fatal(err)
	}

	rec := get(t, h, "/chat")
	wantStatus(t, rec, http.StatusOK)
	body := rec.Body.String()

	if !strings.Contains(body, ">You</span>") {
		t.Errorf("user messages should show author \"You\"\nbody:\n%s", excerpt(body))
	}

	if !strings.Contains(body, ">Assistant</span>") {
		t.Errorf("assistant messages should show author \"Assistant\"\nbody:\n%s", excerpt(body))
	}
}

// excerpt returns the first 800 characters of s for error output.
func excerpt(s string) string {
	if len(s) > 800 {
		return s[:800] + "..."
	}
	return s
}
