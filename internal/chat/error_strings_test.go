package chat_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
)

// toolErrorsArePlain asserts that error strings returned by the chat tools
// use short, active language. These tests target the strings themselves via
// lastToolResult and will fail as soon as the developer changes them to match
// the acceptance criteria and pass once the change lands.

func TestAddComponentUnknownRefusesWithPlainLanguage(t *testing.T) {
	svc := newFullService(t)
	m := &scripted{steps: []*llm.Response{call("add_component", map[string]any{"component": "nonexistent"})}}
	svc.Provider = m
	svc.Send(context.Background(), "add a nonexistent thing")

	res := lastToolResult(m.seen[1])
	if !res.IsError {
		t.Errorf("unknown component should be an error, got %+v", res)
	}
	// The new message avoids passive voice: no "could not save the block"
	if strings.Contains(res.Content, "could not save") || strings.Contains(res.Content, "could not update") {
		t.Errorf("should not use 'could not' phrases; got: %s", res.Content)
	}
}

func TestUpdateComponentMissingBlockSaysNotFound(t *testing.T) {
	svc := newFullService(t)
	m := &scripted{steps: []*llm.Response{call("update_component", map[string]any{"id": "nope"})}}
	svc.Provider = m
	svc.Send(context.Background(), "update a missing block")

	res := lastToolResult(m.seen[1])
	if !res.IsError {
		t.Errorf("missing block should be an error, got %+v", res)
	}
	// After change: no passive voice like "could not update"
	if strings.Contains(res.Content, "could not") {
		t.Errorf("should not use 'could' phrases; got: %s", res.Content)
	}
}

func TestUpdateComponentNothingToChange(t *testing.T) {
	svc := newFullService(t)
	m := &scripted{steps: []*llm.Response{call("update_component", map[string]any{"id": "abc"})}}
	svc.Provider = m
	svc.Send(context.Background(), "change nothing")

	res := lastToolResult(m.seen[1])
	if !res.IsError {
		t.Errorf("updating with no fields should be an error, got %+v", res)
	}
	// After change: simpler phrasing
	if strings.Contains(res.Content, "could not update block") || strings.Contains(res.Content, "could not save the block:") {
		t.Errorf("should use 'update failed' or similar; got: %s", res.Content)
	}
}

func TestJSONParseErrorIsPlain(t *testing.T) {
	svc := newFullService(t)
	m := &scripted{steps: []*llm.Response{&llm.Response{ToolCalls: []llm.ToolCall{{ID: "c", Name: "add_component", Args: json.RawMessage(`{bad`)}}}}}
	svc.Provider = m
	svc.Send(context.Background(), "send bad JSON")

	res := lastToolResult(m.seen[1])
	if !res.IsError {
		t.Errorf("bad JSON should be an error, got %+v", res)
	}
	// After change: simpler phrasing for tool errors
	if strings.Contains(res.Content, "could not save the block:") || strings.Contains(res.Content, "could not update") {
		t.Errorf("should use 'save failed' or similar; got: %s", res.Content)
	}
}
