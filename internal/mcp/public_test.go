package mcp_test

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/mcp"
	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/server"
)

// From the internet the list is the published reading and nothing else,
// and every tool on it answers when called.
func TestEveryToolTheInternetIsOfferedAnswers(t *testing.T) {
	t.Parallel()
	a, _ := httpServer(t)
	note, err := a.Store.Create("note", map[string]any{"title": "Opening hours", "body": "Nine to five."})
	if err != nil {
		t.Fatal(err)
	}
	a.Chat.Provider = &readsOnce{call: llm.ToolCall{ID: "1", Name: "find_records", Args: json.RawMessage(`{"type":"note"}`)}}
	a.Workspace.Config.Publish.Types = "note"
	web := server.New(a) // its hooks tell the assistant what is published
	h := mcp.Bearer("secret", &mcp.Server{App: a, Version: "test", Published: func() map[string]bool { return web.Published().Types }})
	anyone := &records.Visitor{Access: records.Public}
	args := map[string]string{
		"describe":     `{}`,
		"find_records": `{"type":"note"}`,
		"get_record":   `{"type":"note","id":"` + note.ID + `"}`,
		"ask":          `{"question":"When are you open?"}`,
		"search":       `{"query":"hours"}`,
		"fetch":        `{"id":"note/` + note.ID + `"}`,
	}
	_, listed := post(t, h, "203.0.113.9:5000", "secret", anyone, list)
	names := toolNames(listed)
	if len(names) != len(args) {
		t.Errorf("the internet is offered %v, and this test calls %d of them", names, len(args))
	}
	for _, name := range names {
		_, reply := post(t, h, "203.0.113.9:5000", "secret", anyone,
			fmt.Sprintf(`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":%q,"arguments":%s}}`, name, args[name]))
		if body, isErr := text(t, reply); isErr || body == "" {
			t.Errorf("%s did not answer: %.200s", name, body)
		}
	}
	// What is not published is not so much as named, and not read.
	if raw, _ := json.Marshal(listed); strings.Contains(string(raw), `"task"`) {
		t.Error("a type that is not published is listed")
	}
	_, reply := post(t, h, "203.0.113.9:5000", "secret", anyone, `{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"find_records","arguments":{"type":"task"}}}`)
	if body, isErr := text(t, reply); !isErr || !strings.Contains(body, "only published content can be read here: note") {
		t.Errorf("a type that is not published is refused, naming what is: %q", body)
	}
}
