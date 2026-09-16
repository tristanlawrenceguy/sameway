package server_test

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
)

// A question the assistant asks can be answered wherever it is met, and the
// person stays where they were: under the conversation, or on the proposal's
// own page. Clearing the conversation clears its questions with it, and a
// listing says which proposals still wait.
func TestQuestionsAreAnsweredWhereTheyAreMet(t *testing.T) {
	a, h := newApp(t)
	a.Chat.Provider, a.Chat.ProviderErr = &scripted{steps: []*llm.Response{
		toolCall("add_component", map[string]any{"component": "card", "props": map[string]any{"title": "Old"}}),
		toolCall("propose_change", map[string]any{"summary": "Remove the Old card?", "tool": "remove_component", "id": "will-be-fixed"}),
		{Text: "Asked."},
	}}, nil
	get(t, h, "/")
	postForm(t, h, "/chat", url.Values{"message": {"tidy up"}, "from": {"/"}})
	var blocks struct {
		Records []struct {
			ID     string
			Fields map[string]any
		}
	}
	decode(t, get(t, h, "/api/block"), &blocks)
	var proposals struct {
		Records []struct {
			ID     string
			Fields map[string]any
		}
	}
	decode(t, get(t, h, "/api/proposal"), &proposals)
	if len(proposals.Records) != 1 {
		t.Fatalf("expected one proposal, got %d", len(proposals.Records))
	}
	pid := proposals.Records[0].ID

	// Under the conversation the answers carry the page they were asked on.
	if page := get(t, h, "/chat").Body.String(); !strings.Contains(page, `name="from" value="/chat"`) {
		t.Error("answers under the conversation should say which page they were given on")
	}

	// The proposal's own page offers the same two answers, and the listing
	// says it is pending.
	own := get(t, h, "/t/proposal/"+pid).Body.String()
	if !strings.Contains(own, `data-component="proposal"`) || !strings.Contains(own, `action="/proposal/`+pid+`/dismiss"`) {
		t.Error("a pending proposal's page should carry its two answers")
	}
	if list := get(t, h, "/t/proposal").Body.String(); !strings.Contains(list, "Pending ·") {
		t.Error("the listing should say the proposal is pending")
	}

	// Answering from its own page brings the person back to it, answered.
	rec := postForm(t, h, "/proposal/"+pid+"/dismiss", url.Values{"from": {"/t/proposal/" + pid}})
	wantStatus(t, rec, http.StatusSeeOther)
	if loc := rec.Header().Get("Location"); loc != "/t/proposal/"+pid {
		t.Errorf("after answering, the person should be back on the page they answered from, got %q", loc)
	}
	own = get(t, h, "/t/proposal/"+pid).Body.String()
	if strings.Contains(own, `action="/proposal/`+pid+`/dismiss"`) || !strings.Contains(own, "dismissed") {
		t.Error("an answered proposal offers no answers and says how it was answered")
	}
	if list := get(t, h, "/t/proposal").Body.String(); !strings.Contains(list, "Dismissed ·") {
		t.Error("the listing should say the proposal was dismissed")
	}

	// A conversation cleared takes its open questions with it.
	a.Chat.Provider = &scripted{steps: []*llm.Response{
		toolCall("propose_change", map[string]any{"summary": "Remove it after all?", "tool": "remove_component", "id": blocks.Records[0].ID}),
		{Text: "Asked again."},
	}}
	postForm(t, h, "/chat", url.Values{"message": {"and now?"}, "from": {"/"}})
	if page := get(t, h, "/chat").Body.String(); !strings.Contains(page, "Remove it after all?") {
		t.Fatal("the new question should be waiting under the conversation")
	}
	wantStatus(t, postForm(t, h, "/chat/clear", url.Values{"from": {"/chat"}}), http.StatusSeeOther)
	// The activity log still remembers the question was asked; the page no
	// longer asks it.
	if page := get(t, h, "/chat").Body.String(); strings.Contains(page, `data-component="proposal"`) {
		t.Error("clearing the conversation should clear the questions asked in it")
	}
	decode(t, get(t, h, "/api/proposal"), &proposals)
	for _, p := range proposals.Records {
		if p.Fields["state"] == "pending" {
			t.Errorf("proposal %s should have been dismissed with the conversation", p.ID)
		}
	}
}
