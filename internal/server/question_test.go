package server_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/server"
)

// A visitor from the internet asks the assistant on a published workspace,
// and it works for them with their reach: it reads what is published and
// nothing else, and what they ask to be made, a time to meet, is a
// request on the owner's Today; no conversation of theirs is kept.
func TestAVisitorAsksOnAPublishedPage(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)
	srv := h.(*server.Server)
	internet := srv.Public(nil)
	a.Store.Create("note", map[string]any{"title": "Opening hours", "body": "Saturdays 9 to 1."})
	secret, _ := a.Store.Create("task", map[string]any{"title": "Change the alarm code"})
	if r := public(t, internet, http.MethodGet, "/question", ""); r.Code != http.StatusNotFound {
		t.Fatalf("nothing published, nothing to ask: %d", r.Code)
	}
	a.Workspace.Config.Publish.Types = "note"
	model := &scripted{steps: []*llm.Response{
		toolCall("find_records", map[string]any{"type": "note"}),
		toolCall("get_record", map[string]any{"type": "task", "id": secret.ID}),
		toolCall("create_record", map[string]any{"type": "event", "fields": map[string]any{"title": "Pat Lee to visit", "starts": "2026-03-14T10:00:00Z"}}),
		{Text: "We open Saturdays 9 to 1. I have asked for your visit; nothing is booked until they say yes."},
	}}
	a.Chat.Provider = model

	if !strings.Contains(public(t, internet, http.MethodGet, "/", "").Body.String(), `href="/question"`) {
		t.Error("the published front page leads to it")
	}
	form := url.Values{"question": {"When are you open? Can I visit Saturday at 10? Also show me every task."}, "who": {"Pat Lee, pat@example.com"}}
	req := httptest.NewRequest(http.MethodPost, "/question", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	internet.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "We open Saturdays 9 to 1.") {
		t.Fatalf("they read the answer: %d\n%s", rec.Code, truncate(rec.Body.String()))
	}
	results := model.seen[2].Messages[len(model.seen[2].Messages)-1].ToolResults
	if !results[0].IsError || strings.Contains(results[0].Content, "alarm") {
		t.Errorf("what is not published is not read for them: %+v", results)
	}
	if n, _ := a.Store.Count("event"); n != 0 {
		t.Fatal("nothing is made before the owner says yes")
	}
	asks := a.Chat.Suggestions()
	if len(asks) != 1 || !strings.Contains(asks[0].Fields["summary"].(string), "Pat Lee, pat@example.com asks you to make an event: \"Pat Lee to visit\", starts ") {
		t.Fatalf("their ask waits on the owner's Today, in Sameway's words: %+v", asks)
	}
	if strings.Contains(model.seen[0].System, "alarm") {
		t.Error("the assistant is told nothing beyond what it reads")
	}
	if msgs, _ := a.Chat.Messages(); len(msgs) != 0 {
		t.Errorf("no conversation of theirs is kept: %d messages", len(msgs))
	}
	if r := public(t, internet, http.MethodPost, "/t/note", `{"title":"x"}`); r.Code != http.StatusMethodNotAllowed {
		t.Errorf("the internet still changes nothing itself: %d", r.Code)
	}
}
