package mcp_test

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/mcp"
	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/server"
)

// Over MCP too an agent's key changes things at its pace, and a tool that
// only reads is never held back.
func TestAnAgentsToolCallsArePaced(t *testing.T) {
	a, id := starter(t)
	key, _, err := records.LetAgentIn(a.Store, records.Who{Actor: "human", Via: records.ThroughCLI}, "Busy", "edit")
	if err != nil {
		t.Fatal(err)
	}
	h := server.WithAgentKeys(a, &mcp.Server{App: a, Version: "test"})
	call := func(n int, name, args string) string {
		body := `{"jsonrpc":"2.0","id":` + strconv.Itoa(n) + `,"method":"tools/call","params":{"name":"` + name + `","arguments":` + args + `}}`
		req := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+key)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json, text/event-stream")
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec.Body.String()
	}
	// The budget comes back a change a second, so send until refused.
	tick := func(n int) string {
		return call(n, "update_record", `{"type":"task","id":"`+id+`","fields":{"done":`+strconv.FormatBool(n%2 == 0)+`}}`)
	}
	made, out := 0, tick(0)
	for ; !strings.Contains(out, `"isError":true`) && made < 3*records.PaceChanges; made++ {
		out = tick(made + 1)
	}
	if made < records.PaceChanges || !strings.Contains(out, "wait") {
		t.Errorf("past %d changes the tool says to wait; refused after %d: %s", records.PaceChanges, made, out)
	}
	if out := call(100, "get_record", `{"type":"task","id":"`+id+`"}`); strings.Contains(out, `"isError":true`) {
		t.Errorf("a tool that only reads is not paced: %s", out)
	}
}
