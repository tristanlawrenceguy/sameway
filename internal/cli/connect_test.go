package cli_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/app"
	"github.com/tristanlawrenceguy/sameway/internal/cli"
)

// connect prints the exact configuration a tool needs, or writes it into
// the tool's file keeping every other server there.
func TestConnectPrintsOrWritesTheToolsConfig(t *testing.T) {
	dir := initWorkspace(t)
	r := run(t, dir, "connect", "cursor")
	if r.code != 0 || !strings.Contains(r.stdout, `"mcpServers"`) || !strings.Contains(r.stdout, `"--workspace"`) || !strings.Contains(r.stdout, "mcp.json") {
		t.Errorf("connect should print the snippet and where it goes: %d %s %s", r.code, r.stdout, r.stderr)
	}
	cfg := filepath.Join(t.TempDir(), "mcp.json")
	os.WriteFile(cfg, []byte(`{"mcpServers": {"other": {"command": "x"}}}`), 0o644)
	r = run(t, dir, "connect", "windsurf", "--write", "--config", cfg)
	if r.code != 0 {
		t.Fatalf("write failed: %s", r.stderr)
	}
	var doc struct {
		Servers map[string]struct {
			Command string
			Args    []string
		} `json:"mcpServers"`
	}
	raw, _ := os.ReadFile(cfg)
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	if _, kept := doc.Servers["other"]; !kept || doc.Servers["sameway"].Command == "" || len(doc.Servers["sameway"].Args) != 3 || doc.Servers["sameway"].Args[2] != "mcp" {
		t.Errorf("the file should keep the other server and gain sameway: %s", raw)
	}
	toml := filepath.Join(t.TempDir(), "config.toml")
	os.WriteFile(toml, []byte("model = \"x\"\n"), 0o644)
	if r = run(t, dir, "connect", "codex", "--write", "--config", toml); r.code != 0 {
		t.Fatalf("codex write failed: %s", r.stderr)
	}
	if raw, _ := os.ReadFile(toml); !strings.Contains(string(raw), "model = \"x\"") || !strings.Contains(string(raw), "[mcp_servers.sameway]") {
		t.Errorf("the toml keeps what it had and gains the section: %s", raw)
	}
	if r = run(t, dir, "connect", "vscode"); !strings.Contains(r.stdout, `"servers"`) || !strings.Contains(r.stdout, `"type": "stdio"`) {
		t.Errorf("VS Code's shape names the transport: %s", r.stdout)
	}
	if r = run(t, dir, "connect", "chatgpt"); !strings.Contains(r.stdout, "/mcp") || !strings.Contains(r.stdout, "SAMEWAY_MCP_TOKEN") {
		t.Errorf("a client elsewhere is told about HTTP and the token: %s", r.stdout)
	}
	if r = run(t, dir, "connect", "emacs"); r.code == 0 || !strings.Contains(r.stderr, "cursor") {
		t.Errorf("an unknown tool lists the known ones: %s", r.stderr)
	}
}

// MCP over HTTP is the same server for a client elsewhere: off without a
// token, refused without the right one, and the tools with it.
func TestMCPOverHTTPNeedsTheToken(t *testing.T) {
	dir := initWorkspace(t)
	a, err := app.Load(dir, false)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	post := func(h http.Handler, token, body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(body))
		req.RemoteAddr = "127.0.0.1:5000" // an agent on this computer; from elsewhere it reads only
		req.Header.Set("Content-Type", "application/json")
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec
	}
	off := cli.Handler(a, "")
	if rec := post(off, "", `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`); rec.Code != http.StatusForbidden || !strings.Contains(rec.Body.String(), "SAMEWAY_MCP_TOKEN") {
		t.Errorf("without a token the route is off and says how to turn it on: %d %s", rec.Code, rec.Body.String())
	}
	on := cli.Handler(a, "secret")
	if rec := post(on, "wrong", `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`); rec.Code != http.StatusUnauthorized {
		t.Errorf("a wrong token is refused: %d", rec.Code)
	}
	rec := post(on, "secret", `[{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}},{"jsonrpc":"2.0","method":"notifications/initialized"},{"jsonrpc":"2.0","id":2,"method":"tools/list"}]`)
	var replies []map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &replies); err != nil || rec.Code != http.StatusOK || len(replies) != 2 || !strings.Contains(rec.Body.String(), `"protocolVersion"`) || !strings.Contains(rec.Body.String(), `"find_records"`) {
		t.Errorf("a batch gets one reply per request and none for the notification: %d %v %.300s", rec.Code, err, rec.Body.String())
	}
	if rec := post(on, "secret", `{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"create_record","arguments":{"type":"note","fields":{"title":"From afar"}}}}`); rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "/t/note/") {
		t.Errorf("a tool call works over HTTP: %d %s", rec.Code, rec.Body.String())
	}
	if rec := httptest.NewRecorder(); true {
		on.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/mcp", nil))
		if rec.Code != http.StatusMethodNotAllowed && rec.Code != http.StatusUnauthorized {
			t.Errorf("GET is not how this server is used: %d", rec.Code)
		}
	}
	if rec := httptest.NewRecorder(); true {
		on.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/describe", nil))
		if rec.Code != http.StatusOK {
			t.Errorf("the pages and the API are still there beside /mcp: %d", rec.Code)
		}
	}
}
