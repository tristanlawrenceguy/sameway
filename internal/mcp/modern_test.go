package mcp_test

import (
	"encoding/json"
	"fmt"
	"testing"
)

// modernMeta is the _meta an MCP 2026-07-28 client sends on every request.
const modernMeta = `"_meta":{"io.modelcontextprotocol/protocolVersion":"2026-07-28","io.modelcontextprotocol/clientCapabilities":{},"io.modelcontextprotocol/clientInfo":{"name":"claude-code","version":"3.0.0"}}`

// A client on MCP 2026-07-28 never initializes: it discovers, lists the
// tools and calls them, each request saying its revision and who it is.
// Each result says it is complete, the list is the same order every time,
// and the change is logged by the client's name. A revision this server
// lacks is refused with the ones it has.
func TestAModernClientUsesTheServerWithoutAHandshake(t *testing.T) {
	t.Parallel()
	a, id := starter(t)
	_, replies := drive(t,
		`{"jsonrpc":"2.0","id":1,"method":"server/discover","params":{`+modernMeta+`}}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{`+modernMeta+`}}`,
		`{"jsonrpc":"2.0","id":3,"method":"tools/list","params":{`+modernMeta+`}}`,
		`{"jsonrpc":"2.0","id":4,"method":"tools/list","params":{"_meta":{"io.modelcontextprotocol/protocolVersion":"1900-01-01"}}}`,
	)
	found := result(t, replies[0])
	if found["resultType"] != "complete" || fmt.Sprint(found["supportedVersions"]) != "[2026-07-28]" || found["capabilities"].(map[string]any)["tools"] == nil {
		t.Errorf("server/discover should say the revision and that it has tools: %v", found)
	}
	first, second := result(t, replies[1]), result(t, replies[2])
	if first["resultType"] != "complete" || first["cacheScope"] != "private" {
		t.Errorf("tools/list should be complete and cacheable by this client alone: %v", first["resultType"])
	}
	a1, _ := json.Marshal(first["tools"])
	a2, _ := json.Marshal(second["tools"])
	if len(a1) < 100 || string(a1) != string(a2) {
		t.Error("tools/list should be the same, in the same order, every time")
	}
	refused, _ := replies[3]["error"].(map[string]any)
	if refused["code"] != float64(-32022) || fmt.Sprint(refused["data"].(map[string]any)["supported"]) != "[2026-07-28]" {
		t.Errorf("a revision this server lacks is UnsupportedProtocolVersionError naming the one it has: %v", replies[3])
	}

	// A change from it is logged by the name in its _meta.
	serve(t, a, false, `{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"update_record","arguments":{"type":"task","id":"`+id+`","fields":{"done":true}},`+modernMeta+`}}`)
	if got := latest(t, a).Fields["by"]; got != "Claude Code" {
		t.Errorf("the change should be Claude Code's, got %v", got)
	}
}
