package server_test

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	"golang.org/x/net/html"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/render/htmltest"
)

// TestStateLanguageAfterATurn checks the three-way state contract on the
// home page after the assistant edits the canvas: provenance attributes,
// change markers, the status live region, the receipt, and the activity log.
func TestStateLanguageAfterATurn(t *testing.T) {
	a, h := newApp(t)
	a.Chat.Provider, a.Chat.ProviderErr = &scripted{steps: []*llm.Response{
		toolCall("add_component", map[string]any{"component": "heading", "props": map[string]any{"text": "Shopping"}}),
		toolCall("add_component", map[string]any{"component": "list", "props": map[string]any{"items": []string{"milk"}}}),
		{Text: "Added a heading and a list."},
	}}, nil
	// Opening the canvas seeds its chat block first.
	wantStatus(t, get(t, h, "/"), http.StatusOK)
	postForm(t, h, "/chat", url.Values{"message": {"make a shopping list"}})

	page := parse(t, get(t, h, "/"))

	// Provenance: blocks and messages say who did it.
	var blocks []*html.Node
	for _, b := range page.WithAttr("data-block-id", "") {
		if c, _ := htmltest.Attr(b, "data-block-component"); c != "chat" {
			blocks = append(blocks, b)
		}
	}
	if len(blocks) != 2 {
		t.Fatalf("expected 2 content blocks, got %d", len(blocks))
	}
	for _, b := range blocks {
		if actor, _ := htmltest.Attr(b, "data-actor"); actor != "assistant" {
			t.Errorf("block should be attributed to the assistant, got %q", actor)
		}
		if changed, _ := htmltest.Attr(b, "data-changed"); changed != "added" {
			t.Errorf("block from the last turn should be marked added, got %q", changed)
		}
		if vt, _ := htmltest.Attr(b, "style"); !strings.Contains(vt, "view-transition-name: block-") {
			t.Errorf("block needs a view-transition-name for animated navigation: %q", vt)
		}
	}
	badges := page.WithAttr("data-component", "badge")
	if len(badges) < 2 || strings.TrimSpace(htmltest.Text(badges[0])) != "Assistant" {
		t.Errorf("each block should carry a provenance badge, got %d", len(badges))
	}
	actors := map[string]int{}
	for _, m := range page.WithAttr("data-component", "message") {
		actor, _ := htmltest.Attr(m, "data-actor")
		actors[actor]++
	}
	if actors["human"] != 1 || actors["assistant"] != 1 {
		t.Errorf("messages should be attributed human and assistant, got %v", actors)
	}

	// Receipt: the reply lists what changed.
	receipt := page.WithAttr("data-action", "added")
	if len(receipt) < 2 {
		t.Errorf("assistant message should carry a receipt with 2 added entries, got %d", len(receipt))
	}
	if page.ByID("chat-status") == nil {
		t.Fatalf("status live region missing")
	}
	status := page.ByID("chat-status")
	if state, _ := htmltest.Attr(status, "data-state"); state != "done" || !strings.Contains(htmltest.Text(status), "2 changes") {
		t.Errorf("status should say done with 2 changes: %q %q", state, htmltest.Text(status))
	}
	if role, _ := htmltest.Attr(status, "role"); role != "status" {
		t.Errorf("status must be a live region")
	}

	// Busy enhancement hooks on the compose form.
	forms := page.WithAttr("data-busy-target", "chat-status")
	if len(forms) != 1 {
		t.Errorf("compose form should point at the status region")
	}
	if len(page.WithAttr("data-region", "chat")) != 1 {
		t.Errorf("the chat region should be marked for agents with data-region")
	}

	// Activity: both actors appear in the log, on the page and in the API.
	events := page.WithAttr("data-component", "event")
	if len(events) < 3 {
		t.Errorf("recent activity should list the person's message and the assistant's changes, got %d events", len(events))
	}
	var log struct {
		Records []struct{ Fields map[string]any }
	}
	decode(t, get(t, h, "/api/activity"), &log)
	seen := map[string]bool{}
	for _, r := range log.Records {
		seen[r.Fields["actor"].(string)+":"+r.Fields["action"].(string)] = true
	}
	for _, want := range []string{"human:said", "assistant:added"} {
		if !seen[want] {
			t.Errorf("activity log missing %s: %v", want, seen)
		}
	}
	wantStatus(t, get(t, h, "/activity"), http.StatusOK)
}

// TestHumanActionsAreAttributed checks a person's removal and edit of a
// block show up as human activity and human provenance.
func TestHumanActionsAreAttributed(t *testing.T) {
	a, h := newApp(t)
	wantStatus(t, get(t, h, "/"), http.StatusOK)
	rec, err := a.Store.Create("block", map[string]any{"component": "text", "props": map[string]any{"content": "hi"}})
	if err != nil {
		t.Fatal(err)
	}
	// Edit through the generic form: attributed to the person.
	upd := postForm(t, h, "/t/block/"+rec.ID, url.Values{"component": {"text"}, "props": {`{"content":"edited by a person"}`}, "position": {"0"}, "actor": {"assistant"}, "created_by": {"assistant"}})
	wantStatus(t, upd, http.StatusSeeOther)
	page := parse(t, get(t, h, "/"))
	blocks := page.WithAttr("data-block-id", rec.ID)
	if len(blocks) != 1 {
		t.Fatalf("block missing after edit")
	}
	if actor, _ := htmltest.Attr(blocks[0], "data-actor"); actor != "human" {
		t.Errorf("edited block should be attributed to the person, got %q", actor)
	}
	if !strings.Contains(htmltest.Text(blocks[0]), "edited by you") {
		t.Errorf("the block's badge should say edited by you")
	}

	wantStatus(t, postForm(t, h, "/canvas/"+rec.ID+"/delete", nil), http.StatusSeeOther)
	var log struct {
		Records []struct{ Fields map[string]any }
	}
	decode(t, get(t, h, "/api/activity"), &log)
	var actions []string
	for _, r := range log.Records {
		actions = append(actions, r.Fields["actor"].(string)+":"+r.Fields["action"].(string))
	}
	joined := strings.Join(actions, ",")
	if !strings.Contains(joined, "human:removed") || !strings.Contains(joined, "human:updated") {
		t.Errorf("expected human removed and updated in the log, got %v", actions)
	}
}

func TestDesignPageRendersEveryComponent(t *testing.T) {
	a, h := newApp(t)
	rec := get(t, h, "/design")
	wantStatus(t, rec, http.StatusOK)
	doc := parse(t, rec)
	for _, c := range a.Registry.Components() {
		if doc.ByID("component-"+c.Manifest.Name) == nil {
			t.Errorf("styleguide missing %s", c.Manifest.Name)
		}
	}
	if len(doc.WithAttr("class", "sw-swatch")) < 20 {
		t.Errorf("styleguide should show the colour tokens")
	}
	if n := len(doc.Elements("h1")); n != 1 {
		t.Errorf("styleguide has %d h1", n)
	}
	js := get(t, h, "/design/sameway.js")
	wantStatus(t, js, http.StatusOK)
	if !strings.Contains(js.Body.String(), "data-busy-target") {
		t.Errorf("script bundle should include the status enhancement")
	}
}
