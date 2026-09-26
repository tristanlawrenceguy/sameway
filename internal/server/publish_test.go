package server_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/mcp"
	"github.com/tristanlawrenceguy/sameway/internal/server"
)

func public(t *testing.T, h http.Handler, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

// The internet reads what is published, and nothing else: no other type,
// no conversation, no log, no writing, no controls.
func TestTheInternetReadsOnlyWhatIsPublished(t *testing.T) {
	a, h := newApp(t)
	srv := h.(*server.Server)
	a.Chat.Owner = chat.Visitor{Access: chat.Owner, Login: "me@example.com", Name: "Me"}
	a.Chat.Say("the owner's private words")
	n, _ := a.Store.Create("note", map[string]any{"title": "Sourdough", "body": "flour, water, salt"})
	a.Store.Create("task", map[string]any{"title": "Secret errand"})
	pub := srv.Public(nil)

	if r := public(t, pub, http.MethodGet, "/", ""); r.Code != http.StatusNotFound || !strings.Contains(r.Body.String(), "Nothing here is published") {
		t.Errorf("with nothing published, the address says so: %d", r.Code)
	}

	a.Workspace.Config.Publish.Types = "note"
	if r := public(t, pub, http.MethodGet, "/t/note/"+n.ID, ""); r.Code != http.StatusOK || !strings.Contains(r.Body.String(), "flour, water, salt") || !strings.Contains(r.Body.String(), `data-controls="none"`) {
		t.Errorf("a published note is readable, with no controls: %d", r.Code)
	}
	for _, path := range []string{"/t/task", "/chat", "/activity", "/api/note", "/workspaces", "/events"} {
		if r := public(t, pub, http.MethodGet, path, ""); r.Code != http.StatusNotFound || strings.Contains(r.Body.String(), "Secret errand") {
			t.Errorf("%s is not published: %d", path, r.Code)
		}
	}
	if r := public(t, pub, http.MethodPost, "/t/note/add", ""); r.Code != http.StatusMethodNotAllowed {
		t.Errorf("nothing is written from the internet: %d", r.Code)
	}
	if n, _ := a.Store.Count("note"); n != 1 {
		t.Error("and nothing was")
	}
	if r := public(t, pub, http.MethodGet, "/design/sameway.css", ""); r.Code != http.StatusOK {
		t.Errorf("the look comes with it: %d", r.Code)
	}
	if a.Store.Meta("last:me@example.com") != "" || a.Store.Meta("last:owner") != "" {
		t.Error("a reader on the internet is not the owner coming back")
	}
	if page := get(t, h, "/").Body.String(); strings.Contains(page, "Also here:") {
		t.Error("someone on the internet is not someone here")
	}
}

// A published tab is its page, as it is shown, without the conversation;
// the front page lists what is published.
func TestAPublishedTabIsReadableWithoutTheConversation(t *testing.T) {
	a, h := newApp(t)
	srv := h.(*server.Server)
	a.Chat.Say("the owner's private words")
	raw, _ := json.Marshal(map[string]any{"name": "Recipes"})
	if text, isErr := a.Chat.Call("create_canvas", raw); isErr {
		t.Fatal(text)
	}
	a.Workspace.Config.Publish.Tabs = "Recipes"
	pub := srv.Public(nil)
	index := public(t, pub, http.MethodGet, "/", "").Body.String()
	i := strings.Index(index, `href="/c/`)
	if i < 0 {
		t.Fatalf("the front page lists the published tab:\n%s", truncate(index))
	}
	path := index[i+6 : i+6+strings.Index(index[i+6:], `"`)]
	r := public(t, pub, http.MethodGet, path, "")
	if r.Code != http.StatusOK || strings.Contains(r.Body.String(), "private words") {
		t.Errorf("the tab is readable, without the owner's conversation: %d", r.Code)
	}
}

// AI services read what is published, over MCP with no login, only when
// the owner said they may, and only the published types.
func TestAIServicesReadWhatIsPublishedWhenAllowed(t *testing.T) {
	a, h := newApp(t)
	srv := h.(*server.Server)
	a.Store.Create("note", map[string]any{"title": "Sourdough"})
	a.Store.Create("task", map[string]any{"title": "Secret errand"})
	a.Workspace.Config.Publish.Types = "note"
	pub := srv.Public(&mcp.Server{App: a, Version: "test", Published: func() map[string]bool { return srv.Published().Types }})
	list := `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`
	if r := public(t, pub, http.MethodPost, "/mcp", list); r.Code == http.StatusOK {
		t.Error("AI services are off until the owner says")
	}
	a.Workspace.Config.Publish.AI = "on"
	body := public(t, pub, http.MethodPost, "/mcp", list).Body.String()
	if !strings.Contains(body, "find_records") || strings.Contains(body, "create_record") || strings.Contains(body, `"search"`) {
		t.Errorf("reading the published types, and only that: %s", body)
	}
	notes := public(t, pub, http.MethodPost, "/mcp", `{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"find_records","arguments":{"type":"note"}}}`).Body.String()
	tasks := public(t, pub, http.MethodPost, "/mcp", `{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"find_records","arguments":{"type":"task"}}}`).Body.String()
	if !strings.Contains(notes, "Sourdough") || strings.Contains(tasks, "Secret errand") {
		t.Errorf("notes are published, tasks are not:\n%s\n%s", notes, tasks)
	}
}
