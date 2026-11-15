package server

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/records"
)

// POST /ask is how another person's own Sameway asks this one a question
// for them, over Tailscale, which says who they are. It is the one thing
// someone let to ask may do here (access.go): it opens nothing.
//
// A question is a kind and two times, read as exactly that: anything
// else in the request is dropped unread, and no model sees any of it. The
// answer is a state and the times the owner is busy, worked out by
// Sameway itself (chat/enquiry.go). What the owner said they may ask is
// answered at once; the rest is put to the owner, and the same question
// asked again says where that stands.

// askRoute is that route, as the route table names it.
const askRoute = "POST /ask"

func (s *Server) asked(w http.ResponseWriter, r *http.Request) {
	v := records.VisitorOf(r.Context())
	p, _ := s.app.Store.Get(records.PersonType, v.Person)
	if _, ok := s.app.Types.Get(records.EnquiryType); !ok || p == nil {
		writeJSON(w, http.StatusForbidden, map[string]any{"error": "this Sameway takes questions only from the people its owner named"})
		return
	}
	var in struct {
		Ask  string    `json:"ask"`
		From time.Time `json:"from"`
		To   time.Time `json:"to"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<10)).Decode(&in); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": `send JSON with the kind of question and its times: {"ask": "free", "from": "2026-11-19T17:00:00Z", "to": "2026-11-19T23:00:00Z"}`})
		return
	}
	state, busy, put, err := s.app.Chat.Enquire(p, chat.Enquiry{Ask: in.Ask, From: in.From, To: in.To})
	if err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]any{"error": err.Error()})
		return
	}
	if put {
		s.Changed()
		if s.notify != nil {
			name, _ := p.Fields["name"].(string)
			go s.notify(name+"'s Sameway asked when you are free", "It waits for your yes or no.", s.linkTo("/"))
		}
	}
	out := map[string]any{"state": state}
	if state == chat.EnquiryAnswered {
		if busy == nil {
			busy = []records.Span{}
		}
		out["busy"] = busy
	}
	writeJSON(w, http.StatusOK, out)
}
