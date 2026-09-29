package server_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/cli"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

func withKey(h http.Handler, key, method, path, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+key)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Sameway-Agent", "Somebody Else")
	res := httptest.NewRecorder()
	h.ServeHTTP(res, req)
	return res
}

// An agent let in with a key is who its key says, may do what its key
// says, and is gone when the key is taken away, until that is undone.
func TestAnAgentsKeyIsWhoItIs(t *testing.T) {
	a, h := newApp(t)
	me := chat.Who{Actor: "human", Via: chat.ThroughCLI}
	editKey, _, err := chat.LetAgentIn(a.Store, me, "Claude Code", "edit")
	if err != nil {
		t.Fatal(err)
	}
	viewKey, _, _ := chat.LetAgentIn(a.Store, me, "Reader", "view")

	if res := withKey(h, editKey, http.MethodPost, "/api/note", `{"title":"From a key"}`); res.Code != http.StatusCreated {
		t.Fatalf("an edit key writes: %d %s", res.Code, res.Body.String())
	}
	entries, _ := a.Store.List(chat.ActivityType, store.ListOptions{OrderBy: "created_at", Desc: true, Limit: 1})
	if s, _ := entries[0].Fields["summary"].(string); entries[0].Fields["actor"] != chat.ActorAgent || !strings.Contains(s, "Claude Code") || strings.Contains(s, "Somebody Else") {
		t.Errorf("the log names the key's agent, not what it calls itself: %v", entries[0].Fields)
	}
	if res := withKey(h, viewKey, http.MethodPost, "/api/note", `{"title":"No"}`); res.Code != http.StatusForbidden {
		t.Errorf("a view key does not write: %d", res.Code)
	}
	if res := withKey(h, viewKey, http.MethodGet, "/api/note", ""); res.Code != http.StatusOK {
		t.Errorf("a view key reads: %d", res.Code)
	}
	if res := withKey(h, editKey, http.MethodGet, "/workspaces", ""); res.Code != http.StatusForbidden {
		t.Errorf("an edit key is not the owner: %d", res.Code)
	}
	if res := withKey(h, "sw_notakey", http.MethodGet, "/api/note", ""); res.Code != http.StatusUnauthorized {
		t.Errorf("a key nobody made is refused, not taken for none: %d", res.Code)
	}

	c, err := chat.TakeAgentAway(a.Store, "claude code")
	if err != nil {
		t.Fatal(err)
	}
	entry := chat.Record(a.Store, "human", c)
	if res := withKey(h, editKey, http.MethodGet, "/api/note", ""); res.Code != http.StatusUnauthorized {
		t.Errorf("a key taken away no longer works: %d", res.Code)
	}
	if err := a.Chat.UndoAs("human", entry); err != nil {
		t.Fatal(err)
	}
	if res := withKey(h, editKey, http.MethodGet, "/api/note", ""); res.Code != http.StatusOK {
		t.Errorf("undoing lets it back in with the same key: %d", res.Code)
	}

	// Over MCP too: the key says what it may call.
	all := cli.Handler(a, "")
	list := func(key string) []string {
		res := withKey(all, key, http.MethodPost, "/mcp", `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`)
		var out struct {
			Result struct {
				Tools []struct{ Name string } `json:"tools"`
			} `json:"result"`
		}
		json.Unmarshal(res.Body.Bytes(), &out)
		var names []string
		for _, tool := range out.Result.Tools {
			names = append(names, tool.Name)
		}
		return names
	}
	edit, view := strings.Join(list(editKey), ","), strings.Join(list(viewKey), ",")
	if !strings.Contains(edit, "create_record") || strings.Contains(view, "create_record") || !strings.Contains(view, "find_records") {
		t.Errorf("over MCP an edit key writes and a view key reads:\nedit: %s\nview: %s", edit, view)
	}
}
