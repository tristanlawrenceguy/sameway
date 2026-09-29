package mcp_test

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/examples"
	"github.com/tristanlawrenceguy/sameway/internal/app"
	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/mcp"
	"github.com/tristanlawrenceguy/sameway/internal/store"
	"github.com/tristanlawrenceguy/sameway/internal/workspace"
)

// starter is a starter workspace with one task in it, Call plumber.
func starter(t *testing.T) (*app.App, string) {
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
	task, err := a.Store.Create("task", map[string]any{"title": "Call plumber"})
	if err != nil {
		t.Fatal(err)
	}
	return a, task.ID
}

// serve runs a stdio server over a to the end of lines.
func serve(t *testing.T, a *app.App, assistant bool, lines ...string) {
	t.Helper()
	var out bytes.Buffer
	srv := &mcp.Server{App: a, Version: "test", Assistant: assistant, In: strings.NewReader(strings.Join(lines, "\n") + "\n"), Out: &out}
	if err := srv.Serve(context.Background()); err != nil {
		t.Fatal(err)
	}
}

// latest is the newest activity entry.
func latest(t *testing.T, a *app.App) *store.Record {
	t.Helper()
	recs, err := a.Store.List(chat.ActivityType, store.ListOptions{OrderBy: "created_at", Desc: true, Limit: 1})
	if err != nil || len(recs) == 0 {
		t.Fatalf("the log should have an entry: %v", err)
	}
	return recs[0]
}

func initialize(clientInfo string) string {
	return `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":` + clientInfo + `}}`
}

func tick(id string) string {
	return `{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"update_record","arguments":{"type":"task","id":"` + id + `","fields":{"done":true}}}}`
}

const addCard = `{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"add_component","arguments":{"component":"card","props":{"title":"Plumber"}}}}`

// The name a client gives on initialize is on every change it makes: in
// the log, as an agent, and on the blocks it places. And what it did is
// undone like anything else.
func TestAnMCPClientsChangesAreLoggedByItsName(t *testing.T) {
	a, id := starter(t)
	serve(t, a, false, initialize(`{"name":"claude-code","version":"2.1.0"}`), tick(id), addCard)

	blocks, _ := a.Store.List(chat.BlockType, store.ListOptions{OrderBy: "created_at", Desc: true, Limit: 1})
	if len(blocks) != 1 || blocks[0].Fields["created_by"] != chat.ActorAgent || blocks[0].Fields["actor"] != chat.ActorAgent || blocks[0].Fields["agent"] != "Claude Code" {
		t.Errorf("a block an agent places is marked as that agent's: %+v", blocks)
	}
	recs, _ := a.Store.List(chat.ActivityType, store.ListOptions{OrderBy: "created_at"})
	var ticked *store.Record
	for _, r := range recs {
		if r.Fields["target_id"] == id {
			ticked = r
		}
	}
	if ticked == nil {
		t.Fatal("the tick should be in the log")
	}
	if ticked.Fields["actor"] != chat.ActorAgent || ticked.Fields["by"] != "Claude Code" || ticked.Fields["via"] != chat.ThroughMCP {
		t.Errorf("the tick is the agent's, by its name, through MCP: %v", ticked.Fields)
	}
	if got := ticked.Fields["summary"]; got != "Claude Code (through MCP) updated task Call plumber" {
		t.Errorf("the log's sentence names the agent: %q", got)
	}

	if err := a.Chat.UndoAs("human", ticked.ID); err != nil {
		t.Fatal(err)
	}
	if task, _ := a.Store.Get("task", id); task.Fields["done"] == true {
		t.Error("undoing an agent's tick unticks it")
	}
	if got := latest(t, a).Fields["summary"]; got != "You undid: Claude Code (through MCP) updated task Call plumber" {
		t.Errorf("the undo says whose change it took back: %q", got)
	}
}

// A client that gives no name is an agent all the same.
func TestAnUnnamedClientIsAnAgent(t *testing.T) {
	a, id := starter(t)
	serve(t, a, false, tick(id))
	e := latest(t, a)
	if e.Fields["actor"] != chat.ActorAgent || e.Fields["summary"] != "An agent (through MCP) updated task Call plumber" {
		t.Errorf("an unnamed client is An agent: %v", e.Fields)
	}
}

// The assistant in the app, whose model runs its tools over MCP in
// another program, stays the assistant whatever that program calls itself.
func TestTheAssistantInTheAppStaysTheAssistant(t *testing.T) {
	a, id := starter(t)
	serve(t, a, true, initialize(`{"name":"claude-code"}`), tick(id), addCard)
	e := latest(t, a)
	if e.Fields["actor"] != "assistant" || !strings.HasPrefix(e.Fields["summary"].(string), "Assistant added card") {
		t.Errorf("the assistant's change is the assistant's: %v", e.Fields)
	}
	blocks, _ := a.Store.List(chat.BlockType, store.ListOptions{OrderBy: "created_at", Desc: true, Limit: 1})
	if len(blocks) != 1 || blocks[0].Fields["created_by"] != "assistant" {
		t.Errorf("and so is its block: %+v", blocks)
	}
}

// Over HTTP each POST stands alone: the client is given a session on
// initialize, and is known by it after; a header can name it instead.
func TestAnHTTPClientIsKnownByItsSession(t *testing.T) {
	a, id := starter(t)
	h := mcp.Bearer("secret", &mcp.Server{App: a, Version: "test"})
	send := func(body string, header map[string]string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(body))
		req.RemoteAddr = "127.0.0.1:5000"
		req.Header.Set("Authorization", "Bearer secret")
		for k, v := range header {
			req.Header.Set(k, v)
		}
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec
	}
	session := send(initialize(`{"name":"openai-mcp","version":"1.0"}`), nil).Header().Get("Mcp-Session-Id")
	if session == "" {
		t.Fatal("initialize over HTTP gives the client a session")
	}
	send(tick(id), map[string]string{"Mcp-Session-Id": session})
	if got := latest(t, a).Fields["summary"]; got != "ChatGPT (through MCP) updated task Call plumber" {
		t.Errorf("a call in the session is the client's: %q", got)
	}
	send(tick(id), map[string]string{"X-Sameway-Agent": "nightly tidy"})
	if got := latest(t, a).Fields["summary"]; got != "nightly tidy (through MCP) updated task Call plumber" {
		t.Errorf("a header names the agent: %q", got)
	}
	send(tick(id), nil)
	if got := latest(t, a).Fields["summary"]; got != "An agent (through MCP) updated task Call plumber" {
		t.Errorf("without either it is An agent: %q", got)
	}
}
