package server_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// TestAPIClearConversation exercises the POST /api/chat/clear endpoint: it
// creates messages, calls clear via the API, and verifies that messages are
// gone and an activity entry was logged. It covers acceptance items 1–3.
func TestAPIClearConversation(t *testing.T) {
	a, h := newApp(t)

	// Send a message to create data in the conversation.
	rec := postJSON(t, h, http.MethodPost, "/api/chat", map[string]any{
		"message": "hello world",
	})
	wantStatus(t, rec, http.StatusOK)

	// Verify at least one message exists now.
	msgs := get(t, h, "/api/message")
	wantStatus(t, msgs, http.StatusOK)
	var msgList struct {
		Count   int
		Records []any
	}
	decode(t, msgs, &msgList)
	if msgList.Count == 0 {
		t.Fatalf("expected at least one message after posting; got %d", msgList.Count)
	}

	// Call the clear endpoint. (It does not exist yet — this will return
	// a 404 from apiNotFound.)
	clear := postJSON(t, h, http.MethodPost, "/api/chat/clear", nil)

	// Acceptance 1: status 200 with {"status":"cleared"}.
	wantStatus(t, clear, http.StatusOK)
	var clearResp struct {
		Status string `json:"status"`
	}
	decode(t, clear, &clearResp)
	if clearResp.Status != "cleared" {
		t.Errorf("expected status cleared, got %q", clearResp.Status)
	}

	// Acceptance 2: messages are gone.
	msgs = get(t, h, "/api/message")
	wantStatus(t, msgs, http.StatusOK)
	var msgListAfter struct {
		Count   int
		Records []any
	}
	decode(t, msgs, &msgListAfter)
	if msgListAfter.Count != 0 {
		t.Errorf("expected zero messages after clear, got %d", msgListAfter.Count)
	}

	// Acceptance 3: activity log has a "cleared" entry.
	log, err := a.Store.List(chat.ActivityType, store.ListOptions{OrderBy: "created_at", Desc: true, Limit: 1})
	if err != nil {
		t.Fatalf("could not read activity log: %v", err)
	}
	if len(log) == 0 {
		t.Fatal("expected an activity entry after clearing the conversation")
	}
	entry := log[0]
	action, _ := entry.Fields["action"].(string)
	if action != "cleared" {
		t.Errorf("activity entry action: expected cleared, got %q", action)
	}
	target, _ := entry.Fields["target"].(string)
	if target != "conversation" {
		t.Errorf("activity entry target: expected conversation, got %q", target)
	}
	actor, _ := entry.Fields["actor"].(string)
	if actor != "human" {
		t.Errorf("activity entry actor: expected human, got %q", actor)
	}
}

// TestAPIClearConversationEmpty verifies that calling clear on an empty
// conversation succeeds without error. This covers the edge case where there
// are no messages to delete.
func TestAPIClearConversationEmpty(t *testing.T) {
	a, h := newApp(t)

	clear := postJSON(t, h, http.MethodPost, "/api/chat/clear", nil)
	wantStatus(t, clear, http.StatusOK)

	var body map[string]any
	if err := json.Unmarshal(clear.Body.Bytes(), &body); err != nil {
		t.Fatalf("bad JSON: %v", err)
	}
	if body["status"] != "cleared" {
		t.Errorf("expected status cleared, got %v", body["status"])
	}

	// Activity log should still have the cleared entry.
	log, err := a.Store.List(chat.ActivityType, store.ListOptions{OrderBy: "created_at", Desc: true, Limit: 1})
	if err != nil {
		t.Fatalf("could not read activity log: %v", err)
	}
	if len(log) == 0 {
		t.Fatal("expected an activity entry after clearing")
	}
	action, _ := log[0].Fields["action"].(string)
	if action != "cleared" {
		t.Errorf("activity action: expected cleared, got %q", action)
	}
}
