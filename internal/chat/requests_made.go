package chat

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/store"
	"github.com/tristanlawrenceguy/sameway/internal/trim"
)

// Every way someone asks the owner for something ends here, the same:
// the assistant working for them (requests.go), a visitor's question on a
// published page (answer_once.go) and the booking link (booking.go). Each
// is one request, in Sameway's words, on Today, with its own address that
// says what became of it, and the owner is told it waits.

// putRequest puts a change to the owner as a request from whoever this
// service speaks for: what is the change as the owner reads it, from is
// the way it came. It gives the request's summary and key.
func (s *Service) putRequest(action map[string]any, what, from string) (summary, token string, err error) {
	who := s.who.Who()
	if s.who.Login != "" && !strings.HasPrefix(s.who.Login, "device:") && s.who.Device != ByEmail { // by email, their name is their address
		who += " (" + s.who.Login + ")"
	}
	summary = trim.Line(who+" asks you to "+what+". Make it?", 300)
	token = requestToken()
	_, err = s.Apply(records.Op{Type: records.ProposalType, After: map[string]any{"summary": summary, "detail": from,
		"action": action, "state": "pending", "by_action": ByRequest, "yes": "Make it", "no": "No",
		"asked": what, "asked_by": s.forWhom(), "token": token}})
	return summary, token, err
}

// RequestsSince are the requests whoever this service speaks for made
// since a moment, oldest first.
func (s *Service) RequestsSince(since time.Time) []*store.Record {
	who, _ := json.Marshal(s.who)
	recs, _ := s.Store.List(records.ProposalType, store.ListOptions{OrderBy: "created_at"})
	var out []*store.Record
	for _, p := range recs {
		if p.Fields["by_action"] == ByRequest && p.Fields["asked_by"] == string(who) && !p.CreatedAt.Before(since) && p.Fields["token"] != "" {
			out = append(out, p)
		}
	}
	return out
}

// tellOwner tells the owner of each request made since a moment, however
// it was made: by a tool here, or by one the assistant ran in a process of
// its own, which only the store shares.
func (s *Service) tellOwner(since time.Time) {
	if s.owner() {
		return
	}
	for _, p := range s.RequestsSince(since) {
		if RequestState(p) != "waiting" || s.answerAhead(p) { // standing_yes.go: the owner's yes, given ahead
			continue
		}
		if summary, _ := p.Fields["summary"].(string); s.Requested != nil {
			s.Requested(summary)
		}
	}
}
