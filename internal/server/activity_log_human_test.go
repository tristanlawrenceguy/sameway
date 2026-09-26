package server_test

import (
	"net/url"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
)

// TestActivityLogBodyTextIsHumanReadable checks that setting changes in the
// activity log show human-readable body text rather than raw field names.
// Acceptance 1: "Changed pace to Calm" instead of "set ui.pace calm".
func TestActivityLogBodyTextIsHumanReadable(t *testing.T) {
	a, h := newApp(t)

	// Ask the assistant to set a setting. The tool call writes an activity
	// record with action="set", target="ui.pace", detail="quick".
	a.Chat.Provider, a.Chat.ProviderErr = &scripted{steps: []*llm.Response{
		toolCall("set_setting", map[string]any{"key": "ui.pace", "value": "quick"}),
		{Text: "Faster from now on."},
	}}, nil
	postForm(t, h, "/chat", url.Values{"message": {"a bit faster please"}, "from": {"/"}})

	body := get(t, h, "/activity").Body.String()

	if strings.Contains(body, "ui.pace") {
		t.Errorf("activity log body text should not contain raw field name ui.pace\n%s", truncate(body))
	}
	if !strings.Contains(body, "pace") && !strings.Contains(body, "Changed") {
		t.Errorf("activity log body text should mention the setting label; expected 'pace' or 'Changed'\n%s", truncate(body))
	}
}

// TestActivityLogUndoButtonsAreHumanReadable checks that Undo button labels
// on /activity use plain wording instead of machine language.
// Acceptance 2: "Undo changed pace to Quickly" instead of "Undo set ui.pace quick".
func TestActivityLogUndoButtonsAreHumanReadable(t *testing.T) {
	a, h := newApp(t)

	a.Chat.Provider, a.Chat.ProviderErr = &scripted{steps: []*llm.Response{
		toolCall("set_setting", map[string]any{"key": "ui.pace", "value": "still"}),
		{Text: "Slower."},
	}}, nil
	postForm(t, h, "/chat", url.Values{"message": {"slower please"}, "from": {"/"}})

	body := get(t, h, "/activity").Body.String()

	// The visible text and accessible name both use action + target + detail.
	// After the fix they should not contain raw field names like "ui.pace".
	if strings.Contains(body, `sw-event__undo`) && (strings.Contains(body, `(ui.pace)`) || strings.Contains(body, "set ui.pace")) {
		t.Errorf("Undo button text should not contain raw field names\n%s", truncate(body))
	}
}

// TestChatPageChangesListIsHumanReadable checks that the Changes made list
// under assistant replies on /chat shows human-readable descriptions.
// Acceptance 3: "Changed pace to Quick" instead of "set quick (ui.pace)".
func TestChatPageChangesListIsHumanReadable(t *testing.T) {
	a, h := newApp(t)

	a.Chat.Provider, a.Chat.ProviderErr = &scripted{steps: []*llm.Response{
		toolCall("set_setting", map[string]any{"key": "ui.pace", "value": "calm"}),
		{Text: "Slower from now on."},
	}}, nil
	postForm(t, h, "/chat", url.Values{"message": {"make it slower"}, "from": {"/"}})

	body := get(t, h, "/chat").Body.String()

	if strings.Contains(body, `(ui.pace)`) {
		t.Errorf("Changes made list should not show raw field name in parentheses: (ui.pace)\n%s", truncate(body))
	}
	if !strings.Contains(body, "pace") && !strings.Contains(body, "Changed") {
		t.Errorf("Changes made list should mention the setting label; expected 'pace' or 'Changed'\n%s", truncate(body))
	}
}

// TestChatPageUndoButtonsAreHumanReadable checks that Undo buttons in the chat
// Changes made list use plain wording, not machine language.
// Acceptance 4: "Undo changed pace to Calm" instead of "Undo set ui.pace calm".
func TestChatPageUndoButtonsAreHumanReadable(t *testing.T) {
	a, h := newApp(t)

	a.Chat.Provider, a.Chat.ProviderErr = &scripted{steps: []*llm.Response{
		toolCall("set_setting", map[string]any{"key": "ui.pace", "value": "still"}),
		{Text: "All at once."},
	}}, nil
	postForm(t, h, "/chat", url.Values{"message": {"all at once please"}, "from": {"/"}})

	body := get(t, h, "/chat").Body.String()

	if strings.Contains(body, `sw-message__undo`) && (strings.Contains(body, `(ui.pace)`) || strings.Contains(body, `name="ui.pace"`)) {
		t.Errorf("Chat Undo button should not contain raw field names\n%s", truncate(body))
	}
}

// TestActivitySummaryIsHumanReadable checks that the activity record summary
// itself uses human-readable text (this is what logged() asserts against).
// This also covers Acceptance 1 for the log entry heading.
func TestActivitySummaryIsHumanReadable(t *testing.T) {
	a, h := newApp(t)

	a.Chat.Provider, a.Chat.ProviderErr = &scripted{steps: []*llm.Response{
		toolCall("set_setting", map[string]any{"key": "ui.pace", "value": "quick"}),
		{Text: "Faster."},
	}}, nil
	postForm(t, h, "/chat", url.Values{"message": {"faster"}, "from": {"/"}})

	if logged(t, h, "Assistant set ui.pace quick") {
		t.Error("summary should use human-readable text, not raw field names")
	}
	if !logged(t, h, "Assistant changed pace to Quickly") && !logged(t, h, "Assistant changed pace to Quick") {
		t.Error("summary should say 'Assistant changed pace to' with a value label")
	}
}
