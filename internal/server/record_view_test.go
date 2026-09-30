package server_test

import (
	"encoding/json"
	"net/http"
	"sort"
	"strings"
	"testing"
)

// A record reads the same to an agent however it asks: the assistant's
// get_record (and so MCP) and GET /api/{type}/{id} give one view, with
// the same keys and the same whole title.
func TestARecordReadsTheSameEveryWay(t *testing.T) {
	a, h := newApp(t)
	var made map[string]any
	decode(t, postJSON(t, h, http.MethodPost, "/api/note", map[string]any{"title": strings.Repeat("A long title that goes on ", 4)}), &made)
	id := made["id"].(string)
	var api map[string]any
	decode(t, get(t, h, "/api/note/"+id), &api)
	text, isErr := a.Chat.Call("get_record", json.RawMessage(`{"type":"note","id":"`+id+`"}`))
	if isErr {
		t.Fatal(text)
	}
	var tool map[string]any
	if err := json.Unmarshal([]byte(text), &tool); err != nil {
		t.Fatalf("get_record answers JSON: %v", err)
	}
	keys := func(m map[string]any) string {
		var out []string
		for k := range m {
			out = append(out, k)
		}
		sort.Strings(out)
		return strings.Join(out, ",")
	}
	if keys(api) != keys(tool) || api["title"] != tool["title"] || api["version"] != tool["version"] {
		t.Errorf("one view:\napi:  %s %v\ntool: %s %v", keys(api), api["title"], keys(tool), tool["title"])
	}
}
