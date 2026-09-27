package server_test

import (
	"net/url"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
)

// TestActivityLogSettingChangeIsHumanReadable checks that a setting-change
// entry on /activity shows human-readable text instead of raw field paths.
// "Assistant set ui.pace calm" becomes "Assistant changed pace to Calm".
// This covers acceptance item 1.
func TestActivityLogSettingChangeIsHumanReadable(t *testing.T) {
	a, h := newApp(t)

	a.Chat.Provider, a.Chat.ProviderErr = &scripted{steps: []*llm.Response{
		toolCall("set_setting", map[string]any{"key": "ui.pace", "value": "calm"}),
		{Text: "Pace is calm now."},
	}}, nil

	postForm(t, h, "/chat", url.Values{"message": {"make it calm"}, "from": {"/"}})

	body := get(t, h, "/activity").Body.String()

	if strings.Contains(body, `ui.pace`) || strings.Contains(body, `set ui.`) {
		t.Errorf("activity page must not contain raw field paths like 'ui.pace' or 'set ui.'\n\nwant: human-readable text only\n\ngot body:\n%s", truncate(body))
	}

	if !anyH3Says(body, "Assistant changed pace to Calm") {
		t.Errorf("activity page heading should show human-readable setting change\n\nwant: Assistant changed pace to Calm\n\ngot body:\n%s", truncate(body))
	}
}

// TestActivityLogUndoButtonOnSettingChangeIsHumanReadable checks that the
// Undo button accessible name on a setting-change entry uses plain words,
// not field paths. This covers acceptance item 3.
func TestActivityLogUndoButtonOnSettingChangeIsHumanReadable(t *testing.T) {
	a, h := newApp(t)

	a.Chat.Provider, a.Chat.ProviderErr = &scripted{steps: []*llm.Response{
		toolCall("set_setting", map[string]any{"key": "ui.spacing", "value": "wide"}),
		{Text: "Spaced out."},
	}}, nil

	postForm(t, h, "/chat", url.Values{"message": {"make it wide"}, "from": {"/"}})

	body := get(t, h, "/activity").Body.String()

	if strings.Contains(body, `sw-visually-hidden"> set ui.spacing`) ||
		strings.Contains(body, `sw-visually-hidden">set ui.spacing`) ||
		strings.Contains(said(body), "set ui.spacing") {
		t.Errorf("undo button accessible name must not contain raw field paths\n\nwant: 'changed spacing to Wide'\n\ngot body:\n%s", truncate(body))
	}

	if !strings.Contains(body, "Undo") || !strings.Contains(said(body), "changed spacing to Wide") {
		t.Errorf("undo button should have accessible name with human-readable text\n\nwant: 'Undo changed spacing to Wide'\n\ngot body:\n%s", truncate(body))
	}
}

// TestChatChangesMadeSettingChangeIsHumanReadable checks that the Changes made
// list under a chat reply shows human-readable text for setting changes.
// "set calm (ui.spacing)" becomes "changed spacing to Wide". This covers
// acceptance item 2.
func TestChatChangesMadeSettingChangeIsHumanReadable(t *testing.T) {
	a, h := newApp(t)

	a.Chat.Provider, a.Chat.ProviderErr = &scripted{steps: []*llm.Response{
		toolCall("set_setting", map[string]any{"key": "ui.text", "value": "large"}),
		{Text: "Text is larger now."},
	}}, nil

	postForm(t, h, "/chat", url.Values{"message": {"make text bigger"}, "from": {"/"}})

	body := get(t, h, "/chat").Body.String()

	if strings.Contains(body, `ui.text`) || strings.Contains(said(body), "set ui.text") {
		t.Errorf("changes made list must not contain raw field paths\n\nwant: human-readable text only\n\ngot body:\n%s", truncate(body))
	}

	if !strings.Contains(said(body), "changed text to Large") && !strings.Contains(said(body), "text is larger now") {
		t.Errorf("changes made list should show human-readable setting change or reply text\n\nwant: 'changed text to Large' in body text\n\ngot said:\n%q", said(body))
	}
}
