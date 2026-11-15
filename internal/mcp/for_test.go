package mcp_test

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/mcp"
	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// The MCP server a model like Claude Code runs the assistant's tools
// through has the permissions of whom the turn works for: someone who may
// only look is offered only what the assistant has for them, none of the
// server's own tools that read as the owner, a change they ask for becomes
// a request, and the owner's own kinds are not there.
func TestTheAssistantsMCPServerWorksForWhomTheTurnIsFor(t *testing.T) {
	t.Parallel()
	a, _ := starter(t)
	vi, _ := a.Store.Create("person", map[string]any{"name": "Vi", "email": "vi@example.com", "access": "view"})
	a.Store.Create(records.EnquiryType, map[string]any{"question": "When you are free", "state": "answered"})
	viewer := records.Visitor{Person: vi.ID, Name: "Vi", Login: "vi@example.com", Access: records.View}
	lines := []string{
		`{"jsonrpc":"2.0","id":1,"method":"tools/list"}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"create_record","arguments":{"type":"note","fields":{"title":"Milk for Saturday"}}}}`,
		`{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"find_records","arguments":{"type":"enquiry"}}}`,
		`{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"look","arguments":{"path":"/t/enquiry"}}}`,
	}
	var out bytes.Buffer
	srv := &mcp.Server{App: a, Version: "test", In: strings.NewReader(strings.Join(lines, "\n") + "\n"), Out: &out, Assistant: true, For: &viewer}
	if err := srv.Serve(context.Background()); err != nil {
		t.Fatal(err)
	}
	var replies []map[string]any
	for _, line := range strings.Split(strings.TrimSpace(out.String()), "\n") {
		var m map[string]any
		json.Unmarshal([]byte(line), &m)
		replies = append(replies, m)
	}
	var names []string
	for _, raw := range result(t, replies[0])["tools"].([]any) {
		names = append(names, raw.(map[string]any)["name"].(string))
	}
	listed := "," + strings.Join(names, ",") + ","
	for _, not := range []string{"look", "today", "try", "set_setting", "update_sameway"} {
		if strings.Contains(listed, ","+not+",") {
			t.Errorf("working for someone who may only look, it offers %s: %s", not, listed)
		}
	}
	if !strings.Contains(listed, ",find_records,") || !strings.Contains(listed, ",create_record,") {
		t.Errorf("it offers what the assistant has for them: %s", listed)
	}
	said := func(i int) string {
		content := result(t, replies[i])["content"].([]any)
		return content[0].(map[string]any)["text"].(string)
	}
	if !strings.Contains(said(1), "Nothing has changed") {
		t.Errorf("a change they ask for is a request: %s", said(1))
	}
	if notes, _ := a.Store.List("note", store.ListOptions{}); len(notes) != 0 {
		t.Errorf("and nothing is made: %d notes", len(notes))
	}
	if len(a.Chat.Suggestions()) != 1 {
		t.Error("the request waits for the owner")
	}
	if s := said(2); strings.Contains(s, "When you are free") {
		t.Errorf("the owner's own kinds are not there: %s", s)
	}
	if s := said(3); !strings.Contains(s, "may not") && !strings.Contains(s, "not") {
		t.Errorf("the owner's look is refused: %s", s)
	}
}
