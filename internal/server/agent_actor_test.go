package server_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// apiAs sends a JSON body to the API with the given headers.
func apiAs(t *testing.T, h http.Handler, method, path, body string, header map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	for k, v := range header {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func newestEntry(t *testing.T, st *store.Store) *store.Record {
	t.Helper()
	recs, err := st.List(chat.ActivityType, store.ListOptions{OrderBy: "created_at", Desc: true, Limit: 1})
	if err != nil || len(recs) == 0 {
		t.Fatalf("the log should have an entry: %v", err)
	}
	return recs[0]
}

// A program calling the API is an agent, named by its X-Sameway-Agent
// header, or else "An agent": a User-Agent names a library, not who called.
func TestAnAPIWriteIsLoggedByTheAgentsName(t *testing.T) {
	a, h := newApp(t)
	cases := []struct {
		header map[string]string
		want   string
	}{
		{map[string]string{"X-Sameway-Agent": "backup", "User-Agent": "curl/8.4.0"}, "backup (through the API) created note Plan"},
		{map[string]string{"User-Agent": "python-requests/2.31.0"}, "An agent (through the API) created note Plan"},
		{map[string]string{"User-Agent": "Mozilla/5.0 (Windows NT 10.0)"}, "An agent (through the API) created note Plan"},
		{nil, "An agent (through the API) created note Plan"},
	}
	for _, c := range cases {
		wantStatus(t, apiAs(t, h, http.MethodPost, "/api/note", `{"title":"Plan"}`, c.header), http.StatusCreated)
		e := newestEntry(t, a.Store)
		if e.Fields["actor"] != chat.ActorAgent || e.Fields["summary"] != c.want {
			t.Errorf("with %v the log should say %q, got %v", c.header, c.want, e.Fields)
		}
	}
}

// A record and a block written through the API by the same caller say
// the same who: the agent, by its name.
func TestAPIRecordAndBlockWritesAgree(t *testing.T) {
	a, h := newApp(t)
	who := map[string]string{"X-Sameway-Agent": "backup"}
	wantStatus(t, apiAs(t, h, http.MethodPost, "/api/note", `{"title":"Plan"}`, who), http.StatusCreated)
	rec := apiAs(t, h, http.MethodPost, "/api/block", `{"component":"card","props":{"title":"Plan"},"actor":"human","created_by":"human"}`, who)
	wantStatus(t, rec, http.StatusCreated)
	var block struct{ ID string }
	decode(t, rec, &block)
	b, err := a.Store.Get(chat.BlockType, block.ID)
	if err != nil {
		t.Fatal(err)
	}
	e := newestEntry(t, a.Store)
	if e.Fields["actor"] != b.Fields["actor"] || b.Fields["created_by"] != chat.ActorAgent || e.Fields["by"] != b.Fields["agent"] || b.Fields["agent"] != "backup" {
		t.Errorf("the note's entry and the block say the same who: entry %v, block %v", e.Fields, b.Fields)
	}
	home := get(t, h, "/").Body.String()
	if !strings.Contains(home, "Added by backup.") {
		t.Errorf("the block says which agent added it:\n%s", truncate(said(home)))
	}
	wantStatus(t, apiAs(t, h, http.MethodPut, "/api/block/"+block.ID, `{"span":12}`, map[string]string{"X-Sameway-Agent": "tidy"}), http.StatusOK)
	if b, _ = a.Store.Get(chat.BlockType, block.ID); b.Fields["actor"] != chat.ActorAgent || b.Fields["agent"] != "tidy" || b.Fields["created_by"] != chat.ActorAgent {
		t.Errorf("an update marks who changed it last and keeps who added it: %v", b.Fields)
	}
}

// An agent's entry in the log reads with its name and way in first, as a
// word, with an agent's mark, and offers Undo like any other.
func TestTheLogShowsAnAgentsChange(t *testing.T) {
	a, h := newApp(t)
	task, _ := a.Store.Create("task", map[string]any{"title": "Call plumber"})
	wantStatus(t, apiAs(t, h, http.MethodPut, "/api/task/"+task.ID, `{"done":true}`, map[string]string{"X-Sameway-Agent": "Claude Code"}), http.StatusOK)
	chat.Record(a.Store, chat.ActorAgent, chat.Change{Action: "added", Component: "note", Detail: "Shopping", Via: chat.ThroughMCP})

	log := get(t, h, "/activity").Body.String()
	if !strings.Contains(said(log), "Claude Code (through the API) updated task Call plumber") {
		t.Fatalf("the entry names the agent:\n%s", truncate(said(log)))
	}
	if !strings.Contains(said(log), "An agent (through MCP) added note Shopping") {
		t.Errorf("an agent that gave no name is An agent:\n%s", truncate(said(log)))
	}
	if !regexp.MustCompile(`data-actor="agent"[^>]*>.*?<span class="sw-event__actor">Claude Code \(through the API\)</span>`).MatchString(log) {
		t.Error("the entry is marked as an agent's, and says so in words")
	}
	if strings.Contains(log, `<span class="sw-event__actor">Assistant</span> updated`) || strings.Contains(log, `<span class="sw-event__actor">You</span> updated`) {
		t.Error("an agent's change is neither the assistant's nor the person's")
	}
	entries, _ := a.Store.List(chat.ActivityType, store.ListOptions{})
	var tick string
	for _, e := range entries {
		if e.Fields["target_id"] == task.ID {
			tick = e.ID
		}
	}
	if !strings.Contains(log, `action="/activity/`+tick+`/undo"`) {
		t.Fatal("the agent's change offers Undo")
	}
	res := postForm(t, h, "/activity/"+tick+"/undo", url.Values{"from": {"/activity"}})
	if got, _ := a.Store.Get("task", task.ID); got.Fields["done"] == true {
		t.Error("Undo takes the agent's tick back")
	}
	if got := newestEntry(t, a.Store).Fields["summary"]; got != "You undid: Claude Code (through the API) updated task Call plumber" {
		t.Errorf("the undo is the person's and names what it took back: %q", got)
	}
	if res.Code != http.StatusSeeOther {
		t.Errorf("Undo returns to the page: %d", res.Code)
	}
}

// The assistant in the app still reads as the assistant.
func TestTheAssistantStillReadsAsTheAssistant(t *testing.T) {
	a, h := newApp(t)
	chat.Record(a.Store, "assistant", chat.Change{Action: "added", Component: "card", Detail: "Shopping"})
	log := get(t, h, "/activity").Body.String()
	if !strings.Contains(log, `<span class="sw-event__actor">Assistant</span>`) || strings.Contains(log, `data-actor="agent"`) {
		t.Errorf("the assistant's entry is the assistant's:\n%s", truncate(said(log)))
	}
}

// Back after a while, what an agent did meanwhile is news, with its name.
func TestSinceCountsWhatAgentsDid(t *testing.T) {
	a, h := newApp(t)
	a.Store.SetMeta("last:owner", time.Now().Add(-2*time.Hour).UTC().Format(time.RFC3339Nano))
	chat.Record(a.Store, chat.ActorAgent, chat.Change{Action: "updated", Component: "task", Detail: "Call plumber", By: "Claude Code", Via: chat.ThroughMCP})
	chat.Record(a.Store, "human", chat.Change{Action: "added", Component: "note", Detail: "Mine"})

	notice := sinceSection(get(t, h, "/").Body.String())
	if !strings.Contains(notice, "1 change by others") || !strings.Contains(said(notice), "Claude Code (through MCP) updated task Call plumber") {
		t.Errorf("the agent's change is counted and named, the person's own is not:\n%s", notice)
	}
}
