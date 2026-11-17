package server_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
)

// shopAgent is someone else's agent, not a Sameway: an A2A card, and an
// answer as a task with an artifact.
type shopAgent struct {
	cardURL string
	asked   string
}

func (s *shopAgent) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/.well-known/agent-card.json":
		json.NewEncoder(w).Encode(map[string]any{"name": "Shoe shop", "url": s.cardURL})
	case "/a2a":
		b, _ := io.ReadAll(r.Body)
		s.asked = string(b)
		json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": 1, "result": map[string]any{"kind": "task", "status": map[string]any{"state": "completed"},
			"artifacts": []any{map[string]any{"parts": []any{map[string]any{"kind": "text", "text": "We have size 9 in brown, 3 left."}}}}}})
	default:
		http.NotFound(w, r)
	}
}

// The owner's assistant asks any agent with an A2A card, not only another
// Sameway, put to the owner first; what comes back is told as theirs. A
// card that points off the host the owner said yes to is not followed.
func TestMyAssistantAsksAnyAgent(t *testing.T) {
	t.Parallel()
	me, h := newApp(t)
	shop := &shopAgent{cardURL: "https://shoes.example.com/a2a"}
	me.Automate.HTTP = &http.Client{Transport: served{shop}}
	me.Chat.Provider = &scripted{steps: []*llm.Response{
		toolCall("ask_page", map[string]any{"address": "https://shoes.example.com", "question": "Do you have the walking boots in size 9?", "who": "Tris"}),
		{Text: "I have asked you first."},
	}}
	if _, err := me.Chat.Send(context.Background(), "ask the shoe shop's agent if they have the boots in size 9"); err != nil {
		t.Fatal(err)
	}
	ps := me.Chat.Proposals()
	if len(ps) != 1 || shop.asked != "" {
		t.Fatalf("nothing is sent before the owner says yes: %d %q", len(ps), shop.asked)
	}
	wantStatus(t, postForm(t, h, "/proposal/"+ps[0].ID+"/accept", nil), http.StatusSeeOther)
	if !strings.Contains(shop.asked, `"method":"message/send"`) || !strings.Contains(shop.asked, "walking boots in size 9") {
		t.Fatalf("it is asked as A2A asks: %s", shop.asked)
	}
	msgs, _ := me.Chat.Messages()
	told := false
	for _, m := range msgs {
		told = told || strings.Contains(m.Fields["content"].(string), "We have size 9 in brown")
	}
	if !told {
		t.Error("what it answered is told, as theirs")
	}

	shop.cardURL, shop.asked = "https://elsewhere.example.com/a2a", ""
	me.Chat.Call("ask_page", json.RawMessage(`{"address":"https://shoes.example.com","question":"And in size 10?"}`))
	ps = me.Chat.Proposals()
	postForm(t, h, "/proposal/"+ps[len(ps)-1].ID+"/accept", nil)
	if shop.asked != "" {
		t.Errorf("a card pointing off the host is not followed: %s", shop.asked)
	}
}
