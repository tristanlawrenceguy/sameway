package server_test

import (
	"net/http"
	"testing"
)

// The assistant's tools come from the one list the model is given, so an
// agent reads what the model can do without anyone writing it up twice: a
// tool added to the chat service is on /api/describe the same moment.
func TestDescribeListsTheAssistantTools(t *testing.T) {
	_, h := newApp(t)
	rec := get(t, h, "/api/describe")
	wantStatus(t, rec, http.StatusOK)
	var d struct {
		Tools []struct {
			Name        string
			Description string
			Schema      map[string]any
		}
	}
	decode(t, rec, &d)
	tools := map[string]bool{}
	for _, tool := range d.Tools {
		tools[tool.Name] = true
		if tool.Description == "" || tool.Schema["type"] != "object" {
			t.Errorf("tool %s lacks a description or schema in describe", tool.Name)
		}
	}
	for _, want := range []string{"add_component", "clear_conversation", "propose_change", "create_record", "update_record", "find_records"} {
		if !tools[want] {
			t.Errorf("describe missing tool %s", want)
		}
	}
}
