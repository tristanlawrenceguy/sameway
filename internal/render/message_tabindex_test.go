package render_test

import (
	"strings"
	"testing"
)

// TestMessageWithIdHasTabIndex verifies acceptance item 2 of task 0303: a
// message article that carries an id attribute also has tabindex="-1", making
// it focusable so the browser can place keyboard focus on it when a person
// navigates to it via the "Skip to latest message" anchor link.
func TestMessageWithIdHasTabIndex(t *testing.T) {
	reg := builtins(t)

	out, err := reg.Render("message", map[string]any{
		"role":    "user",
		"content": "Hello world.",
		"time":    "14:05",
		"id":      "msg-abc123",
	})
	if err != nil {
		t.Fatalf("render message with id: %v", err)
	}

	got := string(out)
	if !strings.Contains(got, `id="msg-abc123"`) {
		t.Errorf("output should contain the anchor id:\n%s", got)
	}
	if !strings.Contains(got, `tabindex="-1"`) {
		t.Errorf("message article with id needs tabindex=\"-1\" for keyboard focus after skip-link navigation;\ngot:\n%s", got)
	}
}

// TestMessageWithoutIdHasNoTabIndex verifies that a message rendered without
// an id does not get tabindex="-1" — only articles that are anchor targets need
// to be programmatically focusable.
func TestMessageWithoutIdHasNoTabIndex(t *testing.T) {
	reg := builtins(t)

	out, err := reg.Render("message", map[string]any{
		"role":    "assistant",
		"content": "I added a list.",
	})
	if err != nil {
		t.Fatalf("render message without id: %v", err)
	}

	got := string(out)
	if strings.Contains(got, `tabindex`) {
		t.Errorf("message article without an id should not have tabindex;\ngot:\n%s", got)
	}
}
