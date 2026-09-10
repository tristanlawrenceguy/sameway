package server_test

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/render/htmltest"
)

// TestDescribeIsCompleteForAgents checks an agent can learn everything from
// one call: types with schemas, components with contracts, routes, model state.
func TestDescribeIsCompleteForAgents(t *testing.T) {
	_, h := newApp(t)
	rec := get(t, h, "/api/describe")
	wantStatus(t, rec, http.StatusOK)
	var d struct {
		Workspace string
		LLM       struct {
			Provider string
			Ready    bool
			Problem  string
		}
		Types []struct {
			Name   string
			Schema map[string]any
			Fields []struct{ Name, Type string }
		}
		Components []struct {
			Name    string
			Source  string
			Props   map[string]any
			A11y    map[string]any
			Machine map[string]string
		}
		Routes map[string]string
	}
	decode(t, rec, &d)
	if d.LLM.Ready || d.LLM.Problem == "" {
		t.Errorf("describe should say the model is not ready and why: %+v", d.LLM)
	}
	names := map[string]bool{}
	for _, ty := range d.Types {
		names[ty.Name] = true
		if ty.Schema["type"] != "object" || len(ty.Fields) == 0 {
			t.Errorf("type %s lacks a schema", ty.Name)
		}
	}
	for _, want := range []string{"note", "message", "block"} {
		if !names[want] {
			t.Errorf("describe missing type %s", want)
		}
	}
	if len(d.Components) < 13 {
		t.Errorf("expected at least 13 components, got %d", len(d.Components))
	}
	for _, c := range d.Components {
		if c.Props["type"] != "object" || c.A11y["role"] == nil || c.Machine["selector"] == "" {
			t.Errorf("component %s is missing props/a11y/machine in describe", c.Name)
		}
	}
	for _, key := range []string{"describe", "list", "create", "get", "update", "delete", "chat"} {
		if d.Routes[key] == "" {
			t.Errorf("routes missing %s", key)
		}
	}
}

// TestAPILifecycle is the agent's CRUD path with stable error shapes.
func TestAPILifecycle(t *testing.T) {
	_, h := newApp(t)

	bad := postJSON(t, h, http.MethodPost, "/api/note", map[string]any{"status": "bogus", "extra": 1})
	wantStatus(t, bad, http.StatusUnprocessableEntity)
	var e struct {
		Error struct {
			Code   string
			Fields map[string]string
		}
	}
	decode(t, bad, &e)
	if e.Error.Code != "invalid" || e.Error.Fields["title"] == "" || e.Error.Fields["status"] == "" || e.Error.Fields["extra"] == "" {
		t.Errorf("validation error shape wrong: %+v", e)
	}

	created := postJSON(t, h, http.MethodPost, "/api/note", map[string]any{"title": "Via API", "tags": []string{"x"}, "pinned": true})
	wantStatus(t, created, http.StatusCreated)
	var rec struct {
		ID     string
		Fields map[string]any
	}
	decode(t, created, &rec)
	if rec.ID == "" || created.Header().Get("Location") != "/api/note/"+rec.ID {
		t.Fatalf("create should return id and Location: %+v %q", rec, created.Header().Get("Location"))
	}
	if rec.Fields["status"] != "draft" || rec.Fields["pinned"] != true {
		t.Errorf("defaults and booleans wrong: %+v", rec.Fields)
	}

	got := get(t, h, "/api/note/"+rec.ID)
	wantStatus(t, got, http.StatusOK)

	upd := postJSON(t, h, http.MethodPut, "/api/note/"+rec.ID, map[string]any{"status": "published"})
	wantStatus(t, upd, http.StatusOK)
	decode(t, upd, &rec)
	if rec.Fields["status"] != "published" || rec.Fields["title"] != "Via API" {
		t.Errorf("update must merge: %+v", rec.Fields)
	}

	list := get(t, h, "/api/note")
	wantStatus(t, list, http.StatusOK)
	var l struct {
		Count   int
		Records []any
	}
	decode(t, list, &l)
	if l.Count != 1 || len(l.Records) != 1 {
		t.Errorf("list: %+v", l)
	}

	wantStatus(t, do(t, h, http.MethodDelete, "/api/note/"+rec.ID, nil, ""), http.StatusOK)
	wantStatus(t, get(t, h, "/api/note/"+rec.ID), http.StatusNotFound)
	wantStatus(t, get(t, h, "/api/nothing"), http.StatusBadRequest)
	wantStatus(t, do(t, h, http.MethodPost, "/api/note", strings.NewReader("not json"), "application/json"), http.StatusBadRequest)
}

// TestChatBuildsCanvasForPersonAndAgent runs one conversation through the
// JSON API and checks the result shows up on the HTML page, in the block
// API, and can be removed from the page by a person.
func TestChatBuildsCanvasForPersonAndAgent(t *testing.T) {
	a, h := newApp(t)
	model := &scripted{steps: []*llm.Response{
		toolCall("add_component", map[string]any{"component": "table", "props": map[string]any{"caption": "Plan", "columns": []string{"Task", "When"}, "rows": [][]string{{"Write tests", "today"}}}}),
		{Text: "Added a plan table."},
	}}
	a.Chat.Provider, a.Chat.ProviderErr = model, nil

	// Opening the canvas seeds its chat block, the way a person would start.
	wantStatus(t, get(t, h, "/"), http.StatusOK)

	rec := postJSON(t, h, http.MethodPost, "/api/chat", map[string]any{"message": "make a plan table"})
	wantStatus(t, rec, http.StatusOK)
	var reply struct {
		OK    bool
		Reply struct{ Fields map[string]any }
	}
	decode(t, rec, &reply)
	if !reply.OK || reply.Reply.Fields["content"] != "Added a plan table." {
		t.Fatalf("chat reply: %+v", reply)
	}
	if len(model.seen) != 2 || !strings.Contains(model.seen[1].System, "Current canvas") || !strings.Contains(model.seen[1].System, "table") {
		t.Errorf("second model call should see the updated canvas in the system prompt")
	}

	var blocks struct {
		Records []struct {
			ID     string
			Fields map[string]any
		}
	}
	decode(t, get(t, h, "/api/block"), &blocks)
	tableID := ""
	for _, b := range blocks.Records {
		if b.Fields["component"] == "table" {
			tableID = b.ID
		}
	}
	if tableID == "" {
		t.Fatalf("block API has no table: %+v", blocks)
	}

	page := parse(t, get(t, h, "/"))
	items := page.WithAttr("data-block-component", "table")
	if len(items) != 1 {
		t.Fatalf("canvas should render the table block once, got %d", len(items))
	}
	if len(page.WithAttr("data-component", "table")) != 1 || !strings.Contains(htmltest.Text(page.Root), "Write tests") {
		t.Errorf("table content missing from the page")
	}
	msgs := page.WithAttr("data-component", "message")
	if len(msgs) != 2 {
		t.Errorf("expected user and assistant messages on the page, got %d", len(msgs))
	}

	del := postForm(t, h, "/canvas/"+tableID+"/delete", nil)
	wantStatus(t, del, http.StatusSeeOther)
	if len(parse(t, get(t, h, "/")).WithAttr("data-block-component", "table")) != 0 {
		t.Errorf("the table block should be gone after a person removes it")
	}

	wantStatus(t, postForm(t, h, "/chat/clear", nil), http.StatusSeeOther)
	if n := len(parse(t, get(t, h, "/")).WithAttr("data-component", "message")); n != 0 {
		t.Errorf("clear should remove messages, %d left", n)
	}
}

// TestChatFormWithoutModelRecordsAnError covers the person's path when no
// model is reachable: the failure lands in the transcript, not a 500.
func TestChatFormWithoutModelRecordsAnError(t *testing.T) {
	_, h := newApp(t)
	wantStatus(t, get(t, h, "/"), http.StatusOK)
	rec := postForm(t, h, "/chat", url.Values{"message": {"hello"}, "from": {"/"}})
	wantStatus(t, rec, http.StatusSeeOther)
	if !strings.HasPrefix(rec.Header().Get("Location"), "/#msg-") {
		t.Errorf("redirect should jump to the newest message, got %q", rec.Header().Get("Location"))
	}
	page := parse(t, get(t, h, "/"))
	errs := page.WithAttr("data-role", "error")
	if len(errs) != 1 || !strings.Contains(htmltest.Text(errs[0]), "no model configured") {
		t.Errorf("expected one error message explaining the missing model")
	}
	skips := page.WithAttr("class", "sw-skip")
	if len(skips) < 2 {
		t.Errorf("with messages present there should be a skip link to the latest one")
	}
	empty := postForm(t, h, "/chat", url.Values{"message": {"   "}})
	wantStatus(t, empty, http.StatusSeeOther)
	if n := len(parse(t, get(t, h, "/")).WithAttr("data-component", "message")); n != 2 {
		t.Errorf("a blank message must not be recorded; have %d messages", n)
	}
}
