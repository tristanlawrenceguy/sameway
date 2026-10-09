package server_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/records"
)

// TestChatPageShowsPlainErrorMessages checks that when an error-role message
// contains a raw diagnostic string like "claude: exit status 1", the /chat
// page does not render those diagnostics to users. The sanitisation must turn
// provider names, exit codes and file paths into plain language so the person
// reading the transcript sees something understandable. This covers acceptance
// item 1: opening /chat shows no paragraph containing "claude:" or "exit status".

func TestChatPageShowsPlainErrorMessages(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)

	// Seed an error-role message with a raw diagnostic string the way the
	// chat loop stores it when a provider call fails.
	records.Record(a.Store, "system", records.Change{Action: "failed"})
	_, err := a.Store.Create("message", map[string]any{
		"role":    "user",
		"content": "hello",
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = a.Store.Create("message", map[string]any{
		"role":    "error",
		"content": "claude: exit status 1:",
	})
	if err != nil {
		t.Fatal(err)
	}

	rec := get(t, h, "/chat")
	wantStatus(t, rec, http.StatusOK)
	body := rec.Body.String()

	for _, bad := range []string{"claude:", "exit status", "exit_status"} {
		if strings.Contains(body, bad) {
			t.Errorf("/chat page should not show raw diagnostic %q in error messages\nbody: %.600s", bad, body)
		}
	}

	// The plain-language fallback must appear somewhere on the page.
	if !strings.Contains(strings.ToLower(body), "could not reach") &&
		!strings.Contains(strings.ToLower(body), "did not respond") {
		t.Errorf("/chat should show a plain-language error message instead of raw diagnostics\nbody: %.600s", body)
	}

	// The page must still render the message component.
	if !strings.Contains(body, `data-component="message"`) {
		t.Errorf("/chat should still render the message component; got no data-component=message")
	}
}

// TestChatPagePlainErrorWithFilePath checks that file paths in error content are
// removed from /chat rendering. A raw error like "openai: at /tmp/xyz: permission denied"
// must not leak filesystem paths to the user.

func TestChatPagePlainErrorWithFilePath(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)

	_, _ = a.Store.Create("message", map[string]any{
		"role":    "user",
		"content": "ask something",
	})
	_, err := a.Store.Create("message", map[string]any{
		"role":    "error",
		"content": "openai: at /tmp/xyz: permission denied — try again",
	})
	if err != nil {
		t.Fatal(err)
	}

	rec := get(t, h, "/chat")
	wantStatus(t, rec, http.StatusOK)
	body := rec.Body.String()

	if strings.Contains(body, "/tmp/") || strings.Contains(body, "at /tmp") {
		t.Errorf("/chat should not show file paths in error messages\nbody: %.600s", body)
	}

	for _, bad := range []string{"openai:", "permission denied"} {
		if strings.Contains(body, bad) {
			t.Errorf("/chat should sanitise provider prefix and raw OS errors %q\nbody: %.400s", bad, body)
		}
	}
}

// TestChatPagePlainErrorAnthropic checks that "anthropic:" provider prefixes are
// removed from error messages on /chat.

func TestChatPagePlainErrorAnthropic(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)

	_, _ = a.Store.Create("message", map[string]any{
		"role":    "user",
		"content": "test",
	})
	_, err := a.Store.Create("message", map[string]any{
		"role":    "error",
		"content": "anthropic: model timed out after 30s",
	})
	if err != nil {
		t.Fatal(err)
	}

	rec := get(t, h, "/chat")
	wantStatus(t, rec, http.StatusOK)
	body := rec.Body.String()

	if strings.Contains(body, "anthropic:") {
		t.Errorf("/chat should remove the anthropic: provider prefix\nbody: %.600s", body)
	}
}

// TestHomePageCanvasPlainErrorMessages checks that the home page (/) canvas chat
// section also sanitises raw error strings. The canvas embeds a chat block and
// renders messages through the same messageProps path, so it must not leak
// provider names or exit codes. This covers acceptance item 2: the home page
// (/) canvas chat section contains no raw error strings in failure messages.

func TestHomePageCanvasPlainErrorMessages(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)

	_, _ = a.Store.Create("message", map[string]any{
		"role":    "user",
		"content": "hello",
	})
	_, err := a.Store.Create("message", map[string]any{
		"role":    "error",
		"content": "claude: exit status 1: connection refused",
	})
	if err != nil {
		t.Fatal(err)
	}

	rec := get(t, h, "/")
	wantStatus(t, rec, http.StatusOK)
	body := rec.Body.String()

	for _, bad := range []string{"claude:", "exit status"} {
		if strings.Contains(body, bad) {
			t.Errorf("home page should not show raw diagnostic %q in error messages\nbody: %.600s", bad, body)
		}
	}

	// The plain-language fallback must be visible.
	if !strings.Contains(strings.ToLower(body), "could not reach") &&
		!strings.Contains(strings.ToLower(body), "did not respond") {
		t.Errorf("home page should show a plain-language error message instead of raw diagnostics\nbody: %.600s", body)
	}
}
