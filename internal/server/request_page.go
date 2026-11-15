package server

import (
	"encoding/json"
	"html/template"
	"net/http"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/store"
	"github.com/tristanlawrenceguy/sameway/internal/ui"
)

// What became of a request, at its own address, for whoever was given
// it: a visitor from the internet, or another person's agent checking
// back. Its key is in the address and is long enough that nobody finds
// one by guessing. It says what was asked and the answer, in Sameway's
// words, and nothing else.

const (
	requestPrefix = "/request/"
	requestGet    = "GET /request/{token}"
)

func (s *Server) requestPage(w http.ResponseWriter, r *http.Request) {
	p := s.app.Chat.RequestByToken(r.PathValue("token"))
	if p == nil {
		http.NotFound(w, r)
		return
	}
	asked, _ := p.Fields["asked"].(string)
	state := chat.RequestState(p)
	if wantsJSON(r) {
		writeJSON(w, http.StatusOK, map[string]any{"asked": asked, "state": state, "said": requestWords(state)})
		return
	}
	body := template.HTML(`<p>You asked to ` + template.HTMLEscapeString(asked) + `.</p>`)
	s.page(w, r, "Your request", body+s.part(ui.Status{Message: requestWords(state)}), pageOptions{})
}

// requestWords is the answer in a sentence.
func requestWords(state string) string {
	switch state {
	case "yes":
		return "They said yes: it is made."
	case "no":
		return "They said no: nothing was made."
	}
	return "It waits for their answer. Come back to this page to see it."
}

// requestsSince are the requests one visitor made since a moment, each
// with what was asked and its address: what a question page tells them it
// put to the owner.
func (s *Server) requestsSince(r *http.Request, v records.Visitor, since time.Time) []map[string]any {
	who, _ := json.Marshal(v)
	recs, _ := s.app.Store.List(records.ProposalType, store.ListOptions{OrderBy: "created_at"})
	var out []map[string]any
	for _, p := range recs {
		token, _ := p.Fields["token"].(string)
		if p.Fields["by_action"] != chat.ByRequest || p.Fields["asked_by"] != string(who) || p.CreatedAt.Before(since) || token == "" {
			continue
		}
		out = append(out, map[string]any{"asked": p.Fields["asked"], "status": seenFrom(r, chat.RequestPath(token))})
	}
	return out
}

// seenFrom is an address as the one asking reaches it: at the host they
// came by, over https, which is how the internet reaches a workspace.
func seenFrom(r *http.Request, path string) string {
	if r.Host == "" {
		return path
	}
	return "https://" + r.Host + path
}
