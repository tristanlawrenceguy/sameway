package chat_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
)

// TestClearConversationToolCallsClear exercises the clear_conversation tool
// through svc.Call(), which is the same path MCP and the API use. It verifies
// that calling the tool clears all messages and returns a success result.
func TestClearConversationToolCallsClear(t *testing.T) {
	svc, _ := newService(t)

	// Send a message to create data.
	_, err := svc.Send(context.Background(), "hello world")
	if err != nil {
		t.Fatal(err)
	}

	countBefore, _ := svc.Store.Count(chat.MessageType)
	if countBefore == 0 {
		t.Fatalf("expected messages before clear_conversation, got %d", countBefore)
	}

	// Call the tool through the same path MCP uses.
	text, isErr := svc.Call("clear_conversation", json.RawMessage("{}"))
	if isErr {
		t.Fatalf("clear_conversation returned error: %s", text)
	}
	wantResult := "cleared the conversation"
	if text != wantResult {
		t.Errorf("expected result %q, got %q", wantResult, text)
	}

	countAfter, _ := svc.Store.Count(chat.MessageType)
	if countAfter != 0 {
		t.Errorf("expected zero messages after clear_conversation, got %d", countAfter)
	}
}

// TestClearConversationToolNotFound verifies that calling an unknown tool
// returns a clear error message. This is the baseline: once clear_conversation
// exists, it should succeed; until then, unknown tools fail with "unknown tool".
func TestUnknownToolReturnsError(t *testing.T) {
	svc, _ := newService(t)
	text, isErr := svc.Call("frobnicate", json.RawMessage("{}"))
	if !isErr || !strings.Contains(text, "unknown tool") {
		t.Errorf("expected unknown tool error, got isErr=%v text=%q", isErr, text)
	}
}

// TestClearConversationSchemaIsEmpty verifies that the clear_conversation
// tool has an empty schema object: type=object with no properties and
// no required fields. This matches how zero-parameter tools like clear_canvas
// are defined in Tools().
func TestClearConversationSchemaIsEmpty(t *testing.T) {
	svc, _ := newService(t)

	tools := svc.Tools()
	var found bool
	for _, tool := range tools {
		if tool.Name != "clear_conversation" {
			continue
		}
		found = true
		if tool.Description == "" {
			t.Error("clear_conversation tool missing description")
		}
		schema := tool.Schema
		if schema["type"] != "object" {
			t.Errorf("clear_conversation schema type should be object, got %v", schema["type"])
		}
		props := schema["properties"]
		if props == nil {
			t.Error("clear_conversation schema properties is nil")
		} else if propsMap, ok := props.(map[string]any); !ok || len(propsMap) != 0 {
			t.Errorf("clear_conversation schema should have no properties, got %v", props)
		}
		if _, hasRequired := schema["required"]; hasRequired {
			t.Error("clear_conversation schema should not have a required field")
		}
		break
	}
	if !found {
		t.Fatal("clear_conversation tool not found in Tools() list")
	}
}

// TestClearConversationToolDescription verifies the exact description text.
func TestClearConversationToolDescription(t *testing.T) {
	svc, _ := newService(t)
	wantDesc := "Clear all messages from the current conversation, leaving canvas and other chats untouched."
	for _, tool := range svc.Tools() {
		if tool.Name == "clear_conversation" {
			if tool.Description != wantDesc {
				t.Errorf("description mismatch: expected %q, got %q", wantDesc, tool.Description)
			}
			return
		}
	}
	t.Fatal("clear_conversation tool not found in Tools() list")
}
