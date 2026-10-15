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
	"github.com/tristanlawrenceguy/sameway/internal/workspace"
)

// The assistant whose model runs its tools over MCP is given its turn's
// tools and more_tools, as in the app: asked for set_setting, more_tools
// says what it does, the client is told the list changed, and the list
// has it from then on. Another client is given every tool.
func TestTheAssistantOverMCPIsGivenItsTurnsTools(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if err := workspace.Init(dir, examples.FS, examples.StarterRoot, false); err != nil {
		t.Fatal(err)
	}
	a, err := app.Load(dir, true)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { a.Close() })
	lines := []string{
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/list"}`,
		`{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"more_tools","arguments":{"name":"set_setting"}}}`,
		`{"jsonrpc":"2.0","id":4,"method":"tools/list"}`,
	}
	var out bytes.Buffer
	srv := &mcp.Server{App: a, Version: "test", Assistant: true, Only: map[string]bool{"create_record": true, "find_records": true},
		In: strings.NewReader(strings.Join(lines, "\n") + "\n"), Out: &out}
	if err := srv.Serve(context.Background()); err != nil {
		t.Fatal(err)
	}
	var said []map[string]any
	for _, l := range strings.Split(strings.TrimSpace(out.String()), "\n") {
		var m map[string]any
		json.Unmarshal([]byte(l), &m)
		said = append(said, m)
	}
	names := func(m map[string]any) string {
		var n []string
		for _, tl := range m["result"].(map[string]any)["tools"].([]any) {
			n = append(n, tl.(map[string]any)["name"].(string))
		}
		return "," + strings.Join(n, ",") + ","
	}
	if caps := said[0]["result"].(map[string]any)["capabilities"].(map[string]any)["tools"].(map[string]any); caps["listChanged"] != true {
		t.Error("the server says its list changes")
	}
	first := names(said[1])
	if !strings.Contains(first, ",create_record,") || !strings.Contains(first, ",more_tools,") || strings.Contains(first, ",set_setting,") || !strings.Contains(first, ",describe,") {
		t.Errorf("the turn's tools, this server's own, and more_tools: %s", first)
	}
	if text, _ := text(t, said[3]); !strings.Contains(text, "set_setting is yours") {
		t.Errorf("more_tools says what the tool does: %s", text)
	}
	if said[2]["method"] != "notifications/tools/list_changed" {
		t.Errorf("the client is told the list changed, before the answer: %v", said[2])
	}
	if next := names(said[4]); !strings.Contains(next, ",set_setting,") {
		t.Errorf("brought, set_setting is listed: %s", next)
	}

	_, replies := drive(t, `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`)
	if all := names(replies[0]); !strings.Contains(all, ",set_setting,") || strings.Contains(all, ",more_tools,") {
		t.Errorf("another client is given every tool, and no more_tools: %s", all)
	}
}
