package server_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/server"
)

// Other people's agents find a published Sameway the ways they look: an
// A2A agent card at its well-known address and llms.txt, once something
// is published; and asking through A2A is asking through /question, a
// visitor's reach, with what is asked to be made put to the owner and the
// address that says what became of it.
func TestOtherAgentsFindAndAskAPublishedSameway(t *testing.T) {
	t.Parallel()
	a, h := newAppAt(t, tonight)
	internet := h.(*server.Server).Public(nil)
	for _, path := range []string{"/.well-known/agent-card.json", "/llms.txt"} {
		if r := public(t, internet, http.MethodGet, path, ""); r.Code != http.StatusNotFound {
			t.Errorf("%s says nothing while nothing is published: %d", path, r.Code)
		}
	}
	a.Workspace.Config.Publish.Types = "note"
	a.Workspace.Config.Publish.Book = "on"

	var card struct {
		Name, URL string
		Skills    []struct{ ID, Description string }
	}
	json.Unmarshal(public(t, internet, http.MethodGet, "/.well-known/agent-card.json", "").Body.Bytes(), &card)
	if card.URL != "https://example.com/a2a" || len(card.Skills) != 2 || card.Skills[0].ID != "ask" || !strings.Contains(card.Skills[0].Description, "notes") || card.Skills[1].ID != "book" {
		t.Fatalf("the card says what it can be asked, and where: %+v", card)
	}
	if txt := public(t, internet, http.MethodGet, "/llms.txt", "").Body.String(); !strings.Contains(txt, "/question") || !strings.Contains(txt, "agent-card.json") || !strings.Contains(txt, "Free hours") || !strings.Contains(txt, "notes") {
		t.Errorf("llms.txt says the same to a model reading the site: %s", txt)
	}

	a.Chat.Provider = &scripted{steps: []*llm.Response{
		toolCall("create_record", map[string]any{"type": "event", "fields": map[string]any{"title": "Meeting with Pat", "starts": "2026-03-17T15:00:00Z"}}),
		{Text: "I have asked the owner for Tuesday at 3pm."},
	}}
	ask := `{"jsonrpc":"2.0","id":7,"method":"message/send","params":{"message":{"role":"user","messageId":"m1","parts":[{"kind":"text","text":"Can we meet Tuesday at 3pm?"}],"metadata":{"who":"Pat, pat@example.com"}}}}`
	var got struct {
		ID     int
		Result struct {
			Kind, Role string
			Parts      []struct {
				Kind, Text string
				Data       map[string]any
			}
		}
	}
	res := public(t, internet, http.MethodPost, "/a2a", ask)
	json.Unmarshal(res.Body.Bytes(), &got)
	if got.ID != 7 || got.Result.Kind != "message" || got.Result.Role != "agent" || len(got.Result.Parts) != 2 ||
		!strings.Contains(got.Result.Parts[0].Text, "asked the owner") || !strings.Contains(got.Result.Parts[0].Text, "https://example.com/request/") {
		t.Fatalf("an A2A message is answered, with the address of what it asked for: %+v", got)
	}
	if asks := a.Chat.Suggestions(); len(asks) != 1 || !strings.Contains(asks[0].Fields["summary"].(string), "Pat, pat@example.com asks you to make an event") {
		t.Fatalf("what it asked for waits for the owner: %+v", asks)
	}
	if n, _ := a.Store.Count("event"); n != 0 {
		t.Error("and nothing is made")
	}
	if body := public(t, internet, http.MethodPost, "/a2a", `{"jsonrpc":"2.0","id":8,"method":"tasks/cancel"}`).Body.String(); !strings.Contains(body, "-32601") {
		t.Errorf("another method is said to be unknown: %s", body)
	}
}

// An A2A 1.0 client finds the same agent on the same card, among its
// supported interfaces, and asks with SendMessage: the answer comes back in
// 1.0's shape, and what it asks to be made still waits for the owner. A
// version this agent does not speak is said so.
func TestA2AVersionOneClientsAskToo(t *testing.T) {
	t.Parallel()
	a, h := newAppAt(t, tonight)
	internet := h.(*server.Server).Public(nil)
	a.Workspace.Config.Publish.Types = "note"

	var card struct {
		SupportedInterfaces []struct{ URL, ProtocolBinding, ProtocolVersion string }
	}
	json.Unmarshal(public(t, internet, http.MethodGet, "/.well-known/agent-card.json", "").Body.Bytes(), &card)
	if len(card.SupportedInterfaces) != 2 || card.SupportedInterfaces[0].ProtocolVersion != "1.0" || card.SupportedInterfaces[0].ProtocolBinding != "JSONRPC" || card.SupportedInterfaces[0].URL != "https://example.com/a2a" {
		t.Fatalf("the card offers 1.0 first, at the same address: %+v", card)
	}

	a.Chat.Provider = &scripted{steps: []*llm.Response{
		toolCall("create_record", map[string]any{"type": "event", "fields": map[string]any{"title": "Meeting with Sam", "starts": "2026-03-17T15:00:00Z"}}),
		{Text: "I have asked the owner for Tuesday at 3pm."},
	}}
	ask := `{"jsonrpc":"2.0","id":"a","method":"SendMessage","params":{"message":{"role":"ROLE_USER","messageId":"m1","parts":[{"text":"Can we meet Tuesday at 3pm?"}],"metadata":{"who":"Sam, sam@example.com"}}}}`
	var got struct {
		ID     string
		Result struct {
			Message struct {
				Role  string
				Parts []map[string]any
			}
		}
	}
	res := public(t, internet, http.MethodPost, "/a2a?A2A-Version=1.0", ask)
	json.Unmarshal(res.Body.Bytes(), &got)
	m := got.Result.Message
	if got.ID != "a" || m.Role != "ROLE_AGENT" || len(m.Parts) != 2 || !strings.Contains(m.Parts[0]["text"].(string), "https://example.com/request/") ||
		m.Parts[0]["kind"] != nil || m.Parts[1]["data"] == nil || m.Parts[1]["mediaType"] != "application/json" {
		t.Fatalf("a 1.0 message is answered in 1.0's shape: %s", res.Body)
	}
	if asks := a.Chat.Suggestions(); len(asks) != 1 {
		t.Fatalf("what it asked for waits for the owner: %+v", asks)
	}
	if n, _ := a.Store.Count("event"); n != 0 {
		t.Error("and nothing is made")
	}
	if body := public(t, internet, http.MethodPost, "/a2a?A2A-Version=2.0", ask).Body.String(); !strings.Contains(body, "-32009") {
		t.Errorf("a version it does not speak is said so: %s", body)
	}
}
