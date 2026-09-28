package server_test

import (
	"net/url"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
)

// TestNoFillerInReminderEditError checks that when a reminder creation fails,
// the Chat.Notice message does not use "That reminder did not save." as filler.
func TestNoFillerInReminderEditError(t *testing.T) {
	_, h := newApp(t)

	// Post an invalid reminder creation request (missing required "at" field).
	postForm(t, h, "/clock/set", url.Values{"minutes": {"10"}})

	var msgs struct {
		Records []struct{ Fields map[string]any }
	}
	decode(t, get(t, h, "/api/message"), &msgs)

	for _, m := range msgs.Records {
		if role, ok := m.Fields["role"].(string); ok && (role == "error" || role == "notice") {
			content, _ := m.Fields["content"].(string)
			if strings.Contains(content, "That reminder did not save") ||
				strings.Contains(content, "did not save") {
				t.Errorf("reminder error must not use filler phrases like \"That reminder did not save.\",\n"+
					"it should start directly with the problem description.\n"+
					"Got: %s", truncate(content))
			}
		}
	}

	_ = h
}

// TestNoFillerInProposalError checks that when a proposal accept/dismiss fails,
// the Chat.Notice message does not use "That did not go through." as filler.
func TestNoFillerInProposalError(t *testing.T) {
	_, h := newApp(t)

	// Post to an invalid proposal ID — this will trigger the error path in answer().
	res := postForm(t, h, "/proposal/nonexistent-id/accept", url.Values{
		"from": []string{"/"},
	})
	body := res.Body.String()

	if strings.Contains(body, "That did not go through") {
		t.Errorf("proposal error must not use filler \"That did not go through.\",\n"+
			"messages should start directly with what went wrong.\n"+
			"Body: %s", truncate(body))
	}

	_ = h
	_ = res
}

// TestNoFillerInChatOperations checks that chat operations (open, delete) do not
// use conversational filler phrases in their error messages.
func TestNoFillerInChatOperations(t *testing.T) {
	a, h := newApp(t)

	// Try to open a non-existent chat — this hits the "That chat is not here any more." path.
	postForm(t, h, "/chat/open", url.Values{
		"id":   []string{"nonexistent-chat-id"},
		"from": []string{"/"},
	})

	var msgs struct {
		Records []struct{ Fields map[string]any }
	}
	decode(t, get(t, h, "/api/message"), &msgs)

	for _, m := range msgs.Records {
		if role, ok := m.Fields["role"].(string); ok && (role == "error" || role == "notice") {
			content, _ := m.Fields["content"].(string)
			if strings.Contains(content, "That chat is not here any more") ||
				strings.Contains(content, "could not be deleted") ||
				strings.Contains(content, "did not go through") ||
				strings.Contains(content, "did not move") {
				t.Errorf("chat operation error must not use filler phrases like \"That chat is not here any more.\",\n"+
					"messages should start directly with what went wrong.\n"+
					"Got: %s", truncate(content))
			}
		}
	}

	_ = a
	_ = h
}

// TestNoFillerInHabitLogError checks that when habit logging fails, the error
// message does not use "That did not log." as filler.
func TestNoFillerInHabitLogError(t *testing.T) {
	h, _ := canvasWithABlock(t)

	// Post an invalid amount to trigger validation path.
	postForm(t, h, "/habit/00000000-0000-0000-0000-000000000000/log", url.Values{"amount": {"notanumber"}})

	var msgs struct {
		Records []struct{ Fields map[string]any }
	}
	decode(t, get(t, h, "/api/message"), &msgs)

	for _, m := range msgs.Records {
		if role, ok := m.Fields["role"].(string); ok && (role == "error" || role == "notice") {
			content, _ := m.Fields["content"].(string)
			if strings.Contains(content, "That did not log") {
				t.Errorf("habit logging error must not use filler \"That did not log.\",\n"+
					"messages should start directly with what went wrong.\n"+
					"Got: %s", truncate(content))
			}
		}
	}

	_ = h
}

// TestNoConversationalFillerInActionResult checks that chat tool results do not
// use filler phrases like "action X did not go through." — they start directly.
func TestNoConversationalFillerInActionResult(t *testing.T) {
	h, _ := canvasWithABlock(t)

	// Try to run a non-existent action via the API (which is what chat tools use).
	res := get(t, h, "/api/action/nonexistent-id")

	body := res.Body.String()

	if strings.Contains(body, "did not go through") {
		t.Errorf("action results must not say \"did not go through.\",\n"+
			"messages should start directly with what went wrong.\n"+
			"Body: %s", truncate(body))
	}

	_ = h
	_ = res
}

// TestNoFillerInChatWithScriptedModel checks that when using the chat with a
// scripted model, error messages from tool calls do not contain filler phrases.
func TestNoFillerInChatWithScriptedModel(t *testing.T) {
	a, h := newApp(t)
	a.Chat.Provider, a.Chat.ProviderErr = &scripted{steps: []*llm.Response{
		toolCall("run_action", map[string]any{"id": "nonexistent-action"}),
	}}, nil

	postForm(t, h, "/chat", url.Values{"message": {"run the nonexistent action"}})

	var msgs struct {
		Records []struct{ Fields map[string]any }
	}
	decode(t, get(t, h, "/api/message"), &msgs)

	for _, m := range msgs.Records {
		if role, ok := m.Fields["role"].(string); ok && (role == "error" || role == "notice") {
			content, _ := m.Fields["content"].(string)
			if strings.Contains(content, "did not go through") {
				t.Errorf("chat action error must not use filler \"did not go through.\",\n"+
					"messages should start directly with what went wrong.\n"+
					"Got: %s", truncate(content))
			}
		}
	}

	_ = a
	_ = h
}
