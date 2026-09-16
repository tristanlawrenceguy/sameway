package mcp_test

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/examples"
	"github.com/tristanlawrenceguy/sameway/internal/app"
	"github.com/tristanlawrenceguy/sameway/internal/mcp"
	"github.com/tristanlawrenceguy/sameway/internal/store"
	"github.com/tristanlawrenceguy/sameway/internal/workspace"
)

// drive feeds newline-delimited requests to a server over a starter
// workspace and returns one decoded reply per request that had an id.
func drive(t *testing.T, lines ...string) (*app.App, []map[string]any) {
	t.Helper()
	dir := t.TempDir()
	if err := workspace.Init(dir, examples.FS, examples.StarterRoot, false); err != nil {
		t.Fatal(err)
	}
	a, err := app.Load(dir, true)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { a.Close() })
	var out bytes.Buffer
	srv := &mcp.Server{App: a, Version: "test", In: strings.NewReader(strings.Join(lines, "\n") + "\n"), Out: &out}
	if err := srv.Serve(context.Background()); err != nil {
		t.Fatal(err)
	}
	var replies []map[string]any
	for _, line := range strings.Split(strings.TrimSpace(out.String()), "\n") {
		var m map[string]any
		if err := json.Unmarshal([]byte(line), &m); err != nil {
			t.Fatalf("reply is not JSON: %q", line)
		}
		replies = append(replies, m)
	}
	return a, replies
}

func result(t *testing.T, reply map[string]any) map[string]any {
	t.Helper()
	if reply["error"] != nil {
		t.Fatalf("expected a result, got error %v", reply["error"])
	}
	r, _ := reply["result"].(map[string]any)
	return r
}

func text(t *testing.T, reply map[string]any) (string, bool) {
	t.Helper()
	r := result(t, reply)
	content, _ := r["content"].([]any)
	if len(content) == 0 {
		t.Fatalf("no content in %v", r)
	}
	first, _ := content[0].(map[string]any)
	isErr, _ := r["isError"].(bool)
	return first["text"].(string), isErr
}

// An MCP host does the handshake, lists the tools, and calls them: the
// assistant's own verbs plus reading, all from one list.
func TestAnAgentDrivesTheWorkspaceOverMCP(t *testing.T) {
	a, replies := drive(t,
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"test","version":"0"}}}`,
		`{"jsonrpc":"2.0","method":"notifications/initialized"}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/list"}`,
		`{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"create_record","arguments":{"type":"note","fields":{"title":"Water the plants","body":"Every Sunday."}}}}`,
		`{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"find_records","arguments":{"type":"note","query":"plants"}}}`,
		`{"jsonrpc":"2.0","id":5,"method":"ping"}`,
	)
	if len(replies) != 5 {
		t.Fatalf("five requests had ids, got %d replies", len(replies))
	}
	init := result(t, replies[0])
	if init["protocolVersion"] != "2025-06-18" || init["serverInfo"].(map[string]any)["name"] != "sameway" {
		t.Errorf("initialize should say who this is and which protocol: %v", init)
	}
	tools, _ := result(t, replies[1])["tools"].([]any)
	names := map[string]bool{}
	for _, tl := range tools {
		m := tl.(map[string]any)
		names[m["name"].(string)] = true
		if m["inputSchema"].(map[string]any)["type"] != "object" {
			t.Errorf("tool %v needs an object inputSchema", m["name"])
		}
	}
	for _, want := range []string{"describe", "get_record", "create_record", "find_records", "add_component", "propose_change"} {
		if !names[want] {
			t.Errorf("tools/list should offer %s, got %v", want, names)
		}
	}
	created, isErr := text(t, replies[2])
	if isErr || !strings.Contains(created, "/t/note/") {
		t.Errorf("create_record over MCP should make the note and say where it is: %q", created)
	}
	notes, _ := a.Store.List("note", store.ListOptions{})
	if len(notes) != 1 || notes[0].Fields["title"] != "Water the plants" {
		t.Fatalf("the note should be in the store, got %+v", notes)
	}
	if found, _ := text(t, replies[3]); !strings.Contains(found, notes[0].ID) {
		t.Errorf("find_records should list the note's id: %q", found)
	}
	activity, _ := a.Store.List("activity", store.ListOptions{})
	if len(activity) == 0 || activity[len(activity)-1].Fields["action"] != "created" {
		t.Errorf("a change over MCP lands in the activity log like any other, got %+v", activity)
	}
	if _, ok := result(t, replies[4])["protocolVersion"]; ok {
		t.Error("ping answers with an empty result")
	}

	// Reading: get_record returns the whole record, describe the whole workspace.
	_, more := drive(t,
		`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"describe","arguments":{}}}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"get_record","arguments":{"type":"note","id":"nope"}}}`,
	)
	if described, isErr := text(t, more[0]); isErr || !strings.Contains(described, `"tools"`) || !strings.Contains(described, `"mcp"`) {
		t.Errorf("describe over MCP should carry the tools and the mcp route: err=%v", isErr)
	}
	if msg, isErr := text(t, more[1]); !isErr || !strings.Contains(msg, "find_records") {
		t.Errorf("a missing record is a tool error that says what to do: %q", msg)
	}
}

// The protocol's own errors: bad JSON, a method this server lacks, and a
// tool call that the schema refuses, each answered rather than dropped.
func TestMCPErrorsAreAnsweredNotDropped(t *testing.T) {
	_, replies := drive(t,
		`{not json`,
		`{"jsonrpc":"2.0","id":2,"method":"resources/list"}`,
		`{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"create_record","arguments":{"type":"note","fields":{"colour":"red"}}}}`,
		`{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"no_such_tool","arguments":{}}}`,
	)
	if len(replies) != 4 {
		t.Fatalf("expected 4 replies, got %d", len(replies))
	}
	if code := replies[0]["error"].(map[string]any)["code"]; code != float64(-32700) {
		t.Errorf("bad JSON is a parse error (-32700), got %v", code)
	}
	if code := replies[1]["error"].(map[string]any)["code"]; code != float64(-32601) {
		t.Errorf("an unknown method is -32601, got %v", code)
	}
	if msg, isErr := text(t, replies[2]); !isErr || !strings.Contains(msg, "title") || !strings.Contains(msg, "colour") {
		t.Errorf("a schema refusal is a tool error naming the fields: %q", msg)
	}
	if msg, isErr := text(t, replies[3]); !isErr || !strings.Contains(msg, "unknown tool") {
		t.Errorf("an unknown tool is a tool error: %q", msg)
	}
}

// The describe tool is cut by part the same way /api/describe is, so an
// MCP client reads the fields of one type without the whole document.
func TestDescribeOverMCPIsReadByPart(t *testing.T) {
	_, replies := drive(t,
		`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"describe","arguments":{"part":"types","name":"note"}}}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"describe","arguments":{"part":"nope"}}}`,
	)
	if len(replies) != 2 {
		t.Fatalf("two requests, got %d replies", len(replies))
	}
	if body, isErr := text(t, replies[0]); isErr || !strings.Contains(body, `"body"`) || strings.Contains(body, `"components"`) {
		t.Errorf("describe types note should be the note type alone, got err=%v %.200s", isErr, body)
	}
	if body, isErr := text(t, replies[1]); !isErr || !strings.Contains(body, "types, components, tools, routes, llm") {
		t.Errorf("an unknown part should be an error naming the parts, got err=%v %s", isErr, body)
	}
}

// The look tool reads a page through the same handler the API serves.
func TestAnAgentLooksAtAPageOverMCP(t *testing.T) {
	_, replies := drive(t,
		`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"look","arguments":{"path":"/t/note"}}}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"look","arguments":{"component":"card","props":{"title":"Plan"}}}}`,
	)
	if body, isErr := text(t, replies[0]); isErr || !strings.Contains(body, `"headings"`) || !strings.Contains(body, `"problems": []`) {
		t.Errorf("look at a page should give its outline with no problems, got err=%v %.300s", isErr, body)
	}
	if body, isErr := text(t, replies[1]); isErr || !strings.Contains(body, `data-component=\"card\"`) {
		t.Errorf("look at a component should render it from props, got err=%v %.300s", isErr, body)
	}
}
