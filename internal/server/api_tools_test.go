package server_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// Any of the assistant's tools is an agent's over REST, as over MCP: the
// same op, so a record made lands in the store and the log, named for the
// agent, and what is asked first is asked.
func TestAnAgentCallsATool(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)
	req := httptest.NewRequest(http.MethodPost, "/api/tools/create_record", strings.NewReader(`{"type":"note","fields":{"title":"Made over REST"}}`))
	req.Header.Set("X-Sameway-Agent", "helper")
	res := httptest.NewRecorder()
	h.ServeHTTP(res, req)
	if res.Code != http.StatusOK || !strings.Contains(res.Body.String(), `"result"`) {
		t.Fatalf("create_record over REST: %d %s", res.Code, res.Body.String())
	}
	notes, _ := a.Store.List("note", store.ListOptions{})
	found := false
	for _, n := range notes {
		found = found || n.Fields["title"] == "Made over REST"
	}
	if !found {
		t.Error("the note was not made")
	}
	log, _ := a.Store.List(records.ActivityType, store.ListOptions{OrderBy: "created_at", Desc: true, Limit: 1})
	if len(log) == 0 || log[0].Fields["by"] != "helper" || log[0].Fields["actor"] != records.ActorAgent {
		t.Errorf("the log does not name the agent: %+v", log[0].Fields)
	}

	read := do(t, h, http.MethodPost, "/api/tools/find_records", strings.NewReader(`{"type":"note","query":"REST"}`), "application/json")
	if read.Code != http.StatusOK || !strings.Contains(read.Body.String(), "Made over REST") {
		t.Errorf("find_records over REST: %d %s", read.Code, read.Body.String())
	}
	if bad := do(t, h, http.MethodPost, "/api/tools/create_record", strings.NewReader(`{"type":"nothing","fields":{}}`), "application/json"); bad.Code != http.StatusUnprocessableEntity || !strings.Contains(bad.Body.String(), `"invalid"`) {
		t.Errorf("a refused call is 422 with what the tool said: %d %s", bad.Code, bad.Body.String())
	}
	if none := do(t, h, http.MethodPost, "/api/tools/fly", strings.NewReader(`{}`), "application/json"); none.Code != http.StatusNotFound {
		t.Errorf("a tool that is not there: %d", none.Code)
	}
	if notJSON := do(t, h, http.MethodPost, "/api/tools/search", strings.NewReader(`words`), "application/json"); notJSON.Code != http.StatusBadRequest {
		t.Errorf("a body that is not JSON: %d", notJSON.Code)
	}

	// What cannot be taken back is put to the person, not done.
	asked := do(t, h, http.MethodPost, "/api/tools/set_setting", strings.NewReader(`{"key":"llm.base_url","value":"https://elsewhere.example/v1"}`), "application/json")
	if asked.Code != http.StatusOK || !strings.Contains(asked.Body.String(), "put to them") || a.Workspace.Config.LLM.BaseURL == "https://elsewhere.example/v1" {
		t.Errorf("an outward setting is asked first: %d %s", asked.Code, asked.Body.String())
	}
	// And a dry run does not try what reaches outside.
	dry := httptest.NewRequest(http.MethodPost, "/api/tools/run_action", strings.NewReader(`{"id":"x"}`))
	dry.Header.Set("Sameway-Dry-Run", "1")
	dres := httptest.NewRecorder()
	h.ServeHTTP(dres, dry)
	if dres.Code != http.StatusUnprocessableEntity || !strings.Contains(dres.Body.String(), "cannot_try") {
		t.Errorf("a dry run of run_action: %d %s", dres.Code, dres.Body.String())
	}
}

// Who may call a tool over REST is who may have the assistant do it: an
// editor the tools that are not the owner's, someone who may look nothing
// that is sent, as with every change.
func TestAToolOverRESTIsWhoeverMayHaveIt(t *testing.T) {
	t.Parallel()
	_, h := newApp(t)
	editor := records.Visitor{Name: "Bob", Login: "bob@example.com", Access: records.Edit}
	viewer := records.Visitor{Name: "Vi", Login: "vi@example.com", Access: records.View}
	if res := as(t, h, editor, http.MethodPost, "/api/tools/create_record", `{"type":"note","fields":{"title":"Bob's"}}`, "application/json"); res.Code != http.StatusOK {
		t.Errorf("an editor makes a record: %d %s", res.Code, res.Body.String())
	}
	if res := as(t, h, editor, http.MethodPost, "/api/tools/set_setting", `{"key":"ui.pace","value":"calm"}`, "application/json"); res.Code != http.StatusNotFound {
		t.Errorf("an editor changes a setting, the owner's: %d %s", res.Code, res.Body.String())
	}
	if res := as(t, h, viewer, http.MethodPost, "/api/tools/find_records", `{"type":"note"}`, "application/json"); res.Code != http.StatusForbidden {
		t.Errorf("a viewer sends a call: %d", res.Code)
	}
}

// describe says of each tool who may call it, what it is like, and where.
func TestDescribeSaysHowToCallATool(t *testing.T) {
	t.Parallel()
	_, h := newApp(t)
	res := do(t, h, http.MethodGet, "/api/describe/tools/set_setting", nil, "")
	var tool map[string]any
	json.Unmarshal(res.Body.Bytes(), &tool)
	traits, _ := tool["traits"].(map[string]any)
	if tool["who"] != "owner" || tool["call"] != "POST /api/tools/set_setting" || tool["title"] == "" || traits["idempotent"] != true {
		t.Errorf("set_setting described as %s", res.Body.String())
	}
	route := do(t, h, http.MethodGet, "/api/describe/routes/tools", nil, "")
	if !strings.Contains(route.Body.String(), "POST /api/tools/{name}") {
		t.Errorf("the route is not described: %s", route.Body.String())
	}
}
