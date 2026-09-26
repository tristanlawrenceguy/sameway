package mcp_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/examples"
	"github.com/tristanlawrenceguy/sameway/internal/app"
	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/mcp"
	"github.com/tristanlawrenceguy/sameway/internal/workspace"
)

func httpServer(t *testing.T) (*app.App, http.Handler) {
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
	return a, mcp.Bearer("secret", &mcp.Server{App: a, Version: "test"})
}

// post sends one JSON-RPC request from where, with the token or not, as
// the visitor the tailnet says, if any.
func post(t *testing.T, h http.Handler, from, token string, v *chat.Visitor, body string) (int, map[string]any) {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(body))
	req.RemoteAddr = from
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if v != nil {
		req = req.WithContext(chat.WithVisitor(req.Context(), *v))
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	var reply map[string]any
	json.Unmarshal(rec.Body.Bytes(), &reply)
	return rec.Code, reply
}

func toolNames(reply map[string]any) []string {
	r, _ := reply["result"].(map[string]any)
	list, _ := r["tools"].([]any)
	var out []string
	for _, x := range list {
		m, _ := x.(map[string]any)
		name, _ := m["name"].(string)
		out = append(out, name)
	}
	return out
}

const list = `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`
const create = `{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"create_record","arguments":{"type":"note","fields":{"title":"From afar"}}}}`
const find = `{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"find_records","arguments":{"type":"note"}}}`

// At the computer itself, an agent with the token may do everything.
func TestAtTheComputerAnAgentMayChangeThings(t *testing.T) {
	a, h := httpServer(t)
	_, reply := post(t, h, "127.0.0.1:5000", "secret", nil, list)
	if names := strings.Join(toolNames(reply), ","); !strings.Contains(names, "create_record") {
		t.Errorf("every tool is offered at the computer: %s", names)
	}
	post(t, h, "127.0.0.1:5000", "secret", nil, create)
	if n, _ := a.Store.Count("note"); n != 1 {
		t.Error("and a change is made")
	}
}

// From elsewhere on the network, with the token, an agent reads only.
func TestFromElsewhereAnAgentReadsOnly(t *testing.T) {
	a, h := httpServer(t)
	_, reply := post(t, h, "192.168.1.20:5000", "secret", nil, list)
	names := toolNames(reply)
	if strings.Join(names, ",") == "" || strings.Contains(strings.Join(names, ","), "create_record") {
		t.Errorf("only the reading tools are offered from elsewhere: %v", names)
	}
	_, reply = post(t, h, "192.168.1.20:5000", "secret", nil, create)
	r, _ := reply["result"].(map[string]any)
	if r["isError"] != true || !strings.Contains(r["content"].([]any)[0].(map[string]any)["text"].(string), "with only the token, it reads") {
		t.Errorf("a change from elsewhere is refused, saying why: %v", r)
	}
	if n, _ := a.Store.Count("note"); n != 0 {
		t.Error("nothing was changed")
	}
	if _, reply = post(t, h, "192.168.1.20:5000", "secret", nil, find); reply["result"].(map[string]any)["isError"] == true {
		t.Error("reading works from elsewhere")
	}
}

// Over the tailnet, who Tailscale says is asking is the key, and the
// connection may do what that person may: Hana, who may edit, what her
// own assistant may (records, not settings); Vi, who may look, reading
// only; the owner's own laptop, everything. Without a Tailscale identity,
// the token is still needed.
func TestTailscaleIsTheKeyAndTheRoleIsTheReach(t *testing.T) {
	a, h := httpServer(t)
	hana := &chat.Visitor{Name: "Hana", Login: "hana@example.com", Access: chat.Edit, Device: "laptop"}
	code, reply := post(t, h, "100.64.0.7:5000", "", hana, list)
	names := strings.Join(toolNames(reply), ",")
	if code != http.StatusOK || !strings.Contains(names, "create_record") || strings.Contains(names, "set_setting") {
		t.Errorf("Hana gets her own assistant's tools, with no token: %d %s", code, names)
	}
	post(t, h, "100.64.0.7:5000", "", hana, create)
	if n, _ := a.Store.Count("note"); n != 1 {
		t.Error("Hana, who may edit, can make a note")
	}
	setting := `{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"set_setting","arguments":{"key":"ui.pace","value":"still"}}}`
	if _, reply := post(t, h, "100.64.0.7:5000", "", hana, setting); reply["result"].(map[string]any)["isError"] != true {
		t.Error("settings are the owner's, not Hana's")
	}

	vi := &chat.Visitor{Name: "Vi", Login: "vi@example.com", Access: chat.View}
	_, reply = post(t, h, "100.64.0.8:5000", "", vi, list)
	if names := strings.Join(toolNames(reply), ","); strings.Contains(names, "create_record") || !strings.Contains(names, "get_record") {
		t.Errorf("Vi, who may look, reads only: %s", names)
	}

	mine := &chat.Visitor{Login: "me@example.com", Access: chat.Owner, Device: "my-laptop"}
	if _, reply = post(t, h, "100.64.0.9:5000", "", mine, list); !strings.Contains(strings.Join(toolNames(reply), ","), "set_setting") {
		t.Error("the owner's own laptop may do everything")
	}

	if code, _ := post(t, h, "192.168.1.20:5000", "", nil, list); code != http.StatusUnauthorized {
		t.Errorf("without a token or a Tailscale identity, no way in: %d", code)
	}
}
